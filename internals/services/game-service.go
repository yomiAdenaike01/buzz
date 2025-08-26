package services

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
	serviceClients "github.com/yomiAdenaike01/buzz-core/internals/service-clients"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
)

type GameService struct {
	store                     *store.Store
	notificationServiceClient *serviceClients.NotificationClient
	notificationChannel       chan map[string]string
	sw                        sync.Mutex
}

func (g *GameService) FindGameById(id string) *game.Game {
	return g.store.FindGameById(id)
}

func (g *GameService) FindGameByShortCode(shortcode string) *game.Game {
	return g.store.FindGameByShortCode(shortcode)
}

func (g *GameService) GetQuestions() []config.Question {
	return g.store.GetGameQuestions()
}

func (g *GameService) Shutdown() {
	close(g.notificationChannel)
	g.notificationChannel = nil
}

func (g *GameService) shuffleQuestions() []config.Question {
	out := make([]config.Question, len(g.GetQuestions()))
	existingQuestions := g.store.GetGameQuestions()
	copy(out, existingQuestions)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (g *GameService) CreateGame(input game.NewGameInput) *game.Game {
	questions := g.shuffleQuestions()
	input.Questions = questions
	newGame := game.NewGame(input, g.notificationChannel, func(gameId string) {
		g.store.RemoveGame(gameId)
	})
	go func() {
		for msg := range newGame.OnNotification() {
			select {
			case g.notificationChannel <- msg:
			default:
			}
		}
	}()

	g.store.AddNewGame(newGame)
	return newGame
}

func NewGameService(store *store.Store, clients *serviceClients.ServiceClients) *GameService {
	service := &GameService{
		store:                     store,
		notificationServiceClient: clients.Notification,
		notificationChannel:       make(chan map[string]string),
		sw:                        sync.Mutex{},
	}
	go func() {
		for notification := range service.notificationChannel {
			notificationJSON, err := json.Marshal(notification)
			if err != nil {
				log.Println("Failed to send notification")
			}
			var notificationFormatted serviceClients.NotificationContent
			if err := json.Unmarshal(notificationJSON, &notification); err != nil {
				return
			}
			service.notificationServiceClient.SendNotification(notificationFormatted)
		}
	}()
	return service
}
