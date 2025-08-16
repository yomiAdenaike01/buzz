package game

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"maps"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yomiAdenaike01/buzz-core/internals/constants"
)

type Game struct {
	Name            string                `json:"name"`
	Id              string                `json:"id"`
	Players         map[string]*Player    `json:"players"`
	CurrentRound    int                   `json:"current_round"`
	AnsweredPlayers []int                 `json:"-"`
	DateCreated     time.Time             `json:"date_created"`
	Status          constants.GAME_STATUS `json:"status"`
	Out             chan gameMessage      `json:"-"`
	sw              sync.RWMutex          `json:"-"`
	OwnerId         string                `json:"owner_id"`
	Shortcode       string                `json:"shortcode"`
}

type GamePayload struct {
	Name            string                `json:"name"`
	Id              string                `json:"id"`
	Players         []PlayerDetails       `json:"players"`
	CurrentRound    int                   `json:"current_round"`
	AnsweredPlayers []int                 `json:"-"`
	DateCreated     time.Time             `json:"date_created"`
	Status          constants.GAME_STATUS `json:"status"`
	OwnerId         string                `json:"owner_id"`
	Shortcode       string                `json:"short_code"`
}

func toJSON(data any) string {
	d, _ := json.Marshal(data)
	return string(d)
}

func (g *Game) toPlayersPayload() []PlayerDetails {
	players := g.GetSnapshotPlayers()
	playerArr := make([]PlayerDetails, 0, len(players))
	for _, playerDetail := range players {
		playerArr = append(playerArr, PlayerDetails{
			Id:   playerDetail.Id,
			Name: playerDetail.Name,
		})
	}
	return playerArr
}

func (g *Game) toGameJSON() string {
	players := g.toPlayersPayload()
	gameData, err := json.Marshal(GamePayload{
		Players:      players,
		Id:           g.Id,
		Status:       g.Status,
		OwnerId:      g.Id,
		DateCreated:  g.DateCreated,
		CurrentRound: g.CurrentRound,
		Name:         g.Name,
		Shortcode:    g.Shortcode,
	})

	if err != nil {
		log.Println("[Game.toGameJSON]: failed to marshal game")
		return ""
	}

	return string(gameData)
}

func (g *Game) ChangeGameState(status constants.GAME_STATUS) {
	g.Status = status
	g.Out <- gameMessage{
		Content:     toJSON(g),
		Dest:        DestNotSender,
		MessageType: MessageTypeGameStateChange,
	}
}

func (g *Game) Stop(ctx context.Context) error {
	s := g.GetSnapshotPlayers()
	for _, p := range s {
		select {
		case <-ctx.Done():
			log.Printf("Failed to stop err=%s", ctx.Err())
		default:
			p.RemoveConnection()
		}
	}
	return nil
}

func (g *Game) GetSnapshotPlayers() map[string]*Player {
	g.sw.RLock()
	defer g.sw.RUnlock()
	snap := make(map[string]*Player, len(g.Players))
	maps.Copy(snap, g.Players)
	return snap
}

func (g *Game) GetPlayers(filterIds ...string) []*Player {
	out := make([]*Player, 0, len(g.Players))

	snap := g.GetSnapshotPlayers()
	filtersMap := map[string]any{}

	if len(filterIds) > 0 {
		for _, id := range filterIds {
			filtersMap[id] = nil
		}
	}

	fMapLen := len(filtersMap)

	for k, v := range snap {
		if fMapLen > 0 {
			if _, found := filtersMap[k]; found {
				continue
			}
		}
		out = append(out, v)
	}
	return out
}

func (g *Game) CreatePlayer(input BaseCreatePlayerInput) *Player {
	// create player
	p := NewPlayer(CreatePlayerInput{
		Out: g.Out,
		BaseCreatePlayerInput: BaseCreatePlayerInput{
			Connection: input.Connection,
			PlayerDetails: PlayerDetails{
				Name: input.Name,
				Id:   input.Id,
			},
		},
	})
	// assign new player
	g.sw.Lock()
	g.Players[p.Id] = p
	g.sw.Unlock()
	// alert users on new user joined
	g.Out <- gameMessage{
		MessageType: MessageTypeUserJoined,
		Content:     toJSON(p),
		SentFrom:    p.Id,
		Dest:        DestNotSender,
	}
	// decide message type for after creating game
	messageType := MessageGameCreatedSuccess

	if p.Id != g.OwnerId {
		messageType = MessageGameConnectSuccess
	}
	p.MsgIn <- gameMessage{
		MessageType: messageType,
		Content:     g.toGameJSON(),
		Dest:        p.Id,
		SentFrom:    SentFromGame,
	}
	return p
}

func (g *Game) RemovePlayer(pId string) {
	g.sw.Lock()
	p, found := g.Players[pId]
	if !found {
		log.Printf("Player pid=%s coud not be found", pId)
		g.sw.Unlock()
	}
	close(p.MsgIn)
	delete(g.Players, pId)
	p.RemoveConnection()
	g.sw.Unlock()
}

func (g *Game) read() {
	for msg := range g.Out {
		if msg.MessageType == MessageTypeUserLeft {
			g.RemovePlayer(msg.SentFrom)
		}
		snap := g.GetSnapshotPlayers()
		for _, p := range snap {
			if msg.Dest == DestNotSender && p.Id == msg.SentFrom {
				continue
			}
			select {
			case p.MsgIn <- msg:
			default:
				log.Printf("drop: MsgIn full for participantId=%s", p.Id)

			}
		}
	}
}

func generateShortCode() (string, error) {
	maxLen := 4
	var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	l := len(alphabet)
	var builder strings.Builder
	builder.Grow(maxLen)
	for range maxLen {
		i, err := rand.Int(rand.Reader, big.NewInt(int64(l)))
		if err != nil {
			return "", err
		}
		builder.WriteRune(alphabet[i.Int64()])
	}
	return builder.String(), nil
}

func NewGame(name string, ownerId string) *Game {
	code, err := generateShortCode()
	if err != nil {
		log.Println("[Game.NewGame]: Failed to generate shortcode")
	}
	g := &Game{
		Id:              uuid.NewString(),
		Players:         make(map[string]*Player),
		AnsweredPlayers: []int{},
		DateCreated:     time.Now(),
		CurrentRound:    0,
		Name:            name,
		Status:          constants.GAMESTATE_AWAITING_PLAYERS,
		sw:              sync.RWMutex{},
		Out:             make(chan gameMessage, 1025),
		OwnerId:         ownerId,
		Shortcode:       code,
	}
	go g.read()
	return g
}
