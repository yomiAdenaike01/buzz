package services

import (
	"github.com/yomiAdenaike01/buzz-core/internals/clients"
	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
)

type GameService struct {
	store              *store.Store
	notificationClient *clients.NotificationClient
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
	newGame := game.NewGame(input, g.notificationClient, func(gameId string) {
		g.store.RemoveGame(gameId)
	})
	g.store.AddNewGame(newGame)
	return newGame
}

func NewGameService(store *store.Store, notificationClient *clients.NotificationClient) *GameService {
	return &GameService{
		store:              store,
		notificationClient: notificationClient,
	}
}
