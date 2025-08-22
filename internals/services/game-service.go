package services

import (
	"encoding/json"
	"log"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
	serviceClients "github.com/yomiAdenaike01/buzz-core/internals/service-clients"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
)

type GameService struct {
	store                     *store.Store
	notificationServiceClient *serviceClients.NotificationClient
	notificationChannel       chan map[string]string
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

func (g *GameService) CreateGame(input game.NewGameInput) *game.Game {
	newGame := game.NewGame(input, g.notificationChannel, func(gameId string) {
		close(g.notificationChannel)
		g.notificationChannel = nil
		g.store.RemoveGame(gameId)
	})
	g.notificationChannel <- <-newGame.OnNotification()

	g.store.AddNewGame(newGame)
	return newGame
}

func NewGameService(store *store.Store, clients *serviceClients.ServiceClients) *GameService {
	service := &GameService{
		store:                     store,
		notificationServiceClient: clients.Notification,
		notificationChannel:       make(chan map[string]string),
	}
	go func() {
		for {
			if service.notificationChannel == nil {
				return
			}
			select {
			case notification, ok := <-service.notificationChannel:
				if !ok {
					return
				}
				notificationJSON, err := json.Marshal(notification)
				if err != nil {
					log.Println("Failed to send notification")
				}
				var notificationFormatted serviceClients.NotificationContent
				if err := json.Unmarshal(notificationJSON, &notification); err != nil {
					return
				}
				service.notificationServiceClient.SendNotification(notificationFormatted)
			default:
			}

		}
	}()
	return service
}
