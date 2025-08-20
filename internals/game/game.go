package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/constants"
)

type Game struct {
	Name              string             `json:"name"`
	Id                string             `json:"id"`
	Players           map[string]*Player `json:"players"`
	CurrentRound      int                `json:"current_round"`
	CurrentQuestionId string             `json:"current_question_id"`
	RoundQuestionSet  []config.Question  `json:"questions"`
	AnsweredQuestions []string
	AnsweredPlayers   []string              `json:"-"`
	DateCreated       time.Time             `json:"date_created"`
	Status            constants.GAME_STATUS `json:"status"`
	Out               chan gameMessage      `json:"-"`
	sw                sync.RWMutex          `json:"-"`
	OwnerId           string                `json:"owner_id"`
	Shortcode         string                `json:"shortcode"`
	timer             *time.Timer           `json:"-"`
	QuestionEndsAt    time.Time             `json:"-"`
	loadOnce          sync.Once             `json:"-"`
	In                chan gameMessage      `json:"-"`
	round             *Round                `json:"-"`
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

func mustJSON(data any) string {
	d, _ := json.Marshal(data)
	return string(d)
}

func (g *Game) toPlayersPayload() []PlayerDetails {
	players := g.GetPlayers()
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
		OwnerId:      g.OwnerId,
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

// func (g *Game) nextAskerId() string {
// 	g.sw.Lock()
// 	defer g.sw.Unlock()
// 	p := g.Players
// 	existingAnsweredPlayers := make(map[string]bool, len(g.AnsweredPlayers))
// 	for _, pid := range g.AnsweredPlayers {
// 		existingAnsweredPlayers[pid] = true
// 	}

// 	l := len(existingAnsweredPlayers)
// 	for pid := range p {
// 		if l == 0 {
// 			if pid == g.OwnerId {
// 				continue
// 			}
// 			g.AnsweredPlayers = append(g.AnsweredPlayers, pid)
// 			return pid
// 		}
// 		if _, seen := existingAnsweredPlayers[pid]; seen {
// 			continue
// 		}
// 		g.AnsweredPlayers = append(g.AnsweredPlayers, pid)
// 		return pid

// 	}
// 	return ""
// }

//	func (g *Game) nextQuestionId() config.Question {
//		g.sw.Lock()
//		defer g.sw.Unlock()
//		usedQuestionsRef := make(map[string]bool, len(g.AnsweredQuestions))
//		for _, q := range g.AnsweredQuestions {
//			usedQuestionsRef[q] = true
//		}
//		l := len(g.AnsweredQuestions)
//		for _, q := range g.RoundQuestionSet {
//			if l == 0 {
//				g.AnsweredQuestions = append(g.AnsweredQuestions, q.Id)
//				return q
//			}
//			if _, seen := usedQuestionsRef[q.Id]; seen {
//				continue
//			}
//			g.AnsweredQuestions = append(g.AnsweredQuestions, q.Id)
//			return q
//		}
//		return config.Question{}
//	}
func (g *Game) getPlayerOrder() []string {
	x := make([]string, 0, len(g.Players))
	for k := range g.Players {
		x = append(x, k)
	}
	return x
}
func (g *Game) askQuestion() {

	log.Println("[askQuestion]: Processing next question..")
	askerId := g.round.nextAsker()
	log.Printf("[askQuestion]: Found askerId=%s..", askerId)
	if askerId == "" {
		// round complete
		return
	}
	question := g.round.nextQuestion()
	log.Printf("[askQuestion]: Preparing to ask questionId=%s", question.Text)
	if question.Id == "" {
		// round complete
		return
	}

	g.emit(gameMessage{
		SentFrom:    SentFromGame,
		Dest:        DestAll,
		MessageType: MessageTypeGameAskQuestion,
		Content: mustJSON(map[string]any{
			"player_id":      askerId,
			"question":       question,
			"ends_at_ms":     g.QuestionEndsAt.UnixMilli(),
			"game_now_ms":    time.Now().UnixMilli(),
			"question_index": g.round.getTurnIndex(),
		})})

}

func (g *Game) schedule() {
	g.sw.Lock()
	g.QuestionEndsAt = time.Now().Add(30 * time.Second)
	if g.timer != nil {
		g.timer.Stop()
	}
	g.timer = time.AfterFunc(30*time.Second, func() {
		g.schedule()
	})
	g.sw.Unlock()
	g.askQuestion()
}

func (g *Game) newRound() {
	order := g.getPlayerOrder()
	g.sw.Lock()
	g.round = NewRound(g.RoundQuestionSet, order, time.Second*30, g.OwnerId)
	g.sw.Unlock()
}

func (g *Game) start() {
	g.newRound()
	g.changeAndBroadcastGameState(constants.GAMESTATUS_STARTED)
	g.schedule()
}

func (g *Game) changeAndBroadcastGameState(status constants.GAME_STATUS) {
	g.sw.Lock()
	g.Status = status
	g.sw.Unlock()
	g.emit(gameMessage{
		SentFrom: g.OwnerId,
		Content: mustJSON(map[string]string{
			"status": status,
		}),
		Dest:        DestAll,
		MessageType: MessageTypeGameStateChange,
	})

}

func (g *Game) Stop(ctx context.Context) error {
	players := g.GetPlayers()
	for _, p := range players {
		select {
		case <-ctx.Done():
			log.Printf("Failed to stop err=%s", ctx.Err())
		default:
			p.RemoveConnection()
		}
	}
	if g.timer != nil {
		g.timer.Stop()
	}
	g.emit(gameMessage{
		MessageType: MessageTypeGameEnd,
		Dest:        DestAll,
		Content: mustJSON(map[string]any{
			"game_now": time.Now().UnixMilli(),
		}),
	})
	close(g.Out)
	close(g.In)
	return nil
}

func (g *Game) GetPlayers() map[string]*Player {
	g.sw.RLock()
	defer g.sw.RUnlock()
	snap := make(map[string]*Player, len(g.Players))
	maps.Copy(snap, g.Players)
	return snap
}

func (g *Game) CreatePlayer(input BaseCreatePlayerInput) *Player {
	// create player
	p := NewPlayer(CreatePlayerInput{
		Out: g.In,
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
	g.emit(gameMessage{
		MessageType: MessageTypeUserJoined,
		Content: mustJSON(PlayerDetails{
			Id:   p.Id,
			Name: p.Name,
		}),
		SentFrom: p.Id,
		Dest:     DestNotSender,
	})
	// decide message type for after creating game
	messageType := MessageGameCreatedSuccess

	if p.Id != g.OwnerId {
		messageType = MessageGameConnectSuccess
	}
	p.Consume(gameMessage{
		MessageType: messageType,
		Content:     g.toGameJSON(),
		Dest:        p.Id,
		SentFrom:    SentFromGame,
	})

	return p
}

func (g *Game) RemovePlayer(pId string) {
	g.sw.Lock()
	p, found := g.Players[pId]
	if !found {
		log.Printf("Player pid=%s coud not be found", pId)
		g.sw.Unlock()
		return
	}
	delete(g.Players, pId)
	g.sw.Unlock()
	close(p.MsgIn)
	p.RemoveConnection()
}

func (g *Game) emit(msg gameMessage) {
	select {
	case g.Out <- msg:

	default:
		log.Println("Failed to send message to game out")
	}

}

func (g *Game) answerQuestion(msg gameMessage) {
	var m map[string]any
	if err := json.Unmarshal([]byte(msg.Content), &m); err != nil {
		return
	}
	g.sw.RLock()
	timeDiff := time.Until(g.QuestionEndsAt)
	g.sw.RUnlock()

	if timeDiff <= 0 {
		p := g.GetPlayers()
		player, ok := p[msg.SentFrom]
		if !ok {
			return
		}
		g.sw.Lock()
		g.AnsweredPlayers = append(g.AnsweredPlayers, player.Id)
		g.sw.Unlock()

		g.emit(gameMessage{
			Dest: DestAll,
			Content: mustJSON(map[string]any{
				"message": fmt.Sprintf("%s drinks", player.Name),
			}),
		})

		time.AfterFunc(20*time.Second, func() {
			g.emit(gameMessage{
				Dest:        DestAll,
				SentFrom:    "GAME",
				MessageType: MessageTypeTillNextRound,
				Content: mustJSON(map[string]any{
					"game_now": time.Now().UnixMilli(),
					"end":      time.Now().Add(time.Second * 20).UnixMilli(),
				}),
			})
			g.schedule()
		})

	}
}
func (g *Game) loop() {
	for msg := range g.In {
		switch msg.MessageType {
		case MessageTypeAnswerQuestion:
			g.answerQuestion(msg)
		case MessageTypeGameStart:
			g.start()
		case MessageTypeUserLeft:
			g.RemovePlayer(msg.SentFrom)
		case MessageTypeQuestionComplete:
			g.schedule()
		}
	}
}

func (g *Game) read() {
	for msg := range g.Out {
		snap := g.GetPlayers()
		for _, p := range snap {
			if msg.Dest == DestNotSender && p.Id == msg.SentFrom {
				continue
			}
			p.Consume(msg)
		}
	}
}

type NewGameInput struct {
	Name      string
	OwnerId   string
	Questions []config.Question
}

func NewGame(input NewGameInput) *Game {
	code, err := generateShortCode()
	if err != nil {
		log.Println("[Game.NewGame]: Failed to generate shortcode")
	}
	g := &Game{
		Id:               uuid.NewString(),
		Players:          make(map[string]*Player),
		AnsweredPlayers:  make([]string, 0, 10),
		DateCreated:      time.Now(),
		CurrentRound:     0,
		Name:             input.Name,
		Status:           constants.GAMESTATE_AWAITING_PLAYERS,
		sw:               sync.RWMutex{},
		Out:              make(chan gameMessage, 1025),
		OwnerId:          input.OwnerId,
		Shortcode:        code,
		timer:            nil,
		loadOnce:         sync.Once{},
		In:               make(chan gameMessage, 256),
		RoundQuestionSet: input.Questions,
	}
	go g.read()
	go g.loop()
	return g
}
