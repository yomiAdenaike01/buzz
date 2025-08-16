package game

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yomiAdenaike01/buzz-core/internals/constants"
)

type Game struct {
	Name                 string                  `json:"name"`
	Id                   string                  `json:"id"`
	Participants         map[string]*Participant `json:"-"`
	CurrentRound         int                     `json:"current_round"`
	AnsweredParticipants []int                   `json:"-"`
	DateCreated          time.Time               `json:"date_created"`
	Status               constants.GAME_STATUS   `json:"status"`
	Out                  chan gameMessage        `json:"-"`
	sw                   sync.RWMutex            `json:"-"`
	OwnerId              string                  `json:"owner_id"`
}

func (g *Game) Stop(ctx context.Context) error {
	s := g.GetSnapshotParticipants()
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

func toJSON(data any) string {
	d, _ := json.Marshal(data)
	return string(d)
}
func (g *Game) GetSnapshotParticipants() map[string]*Participant {
	g.sw.RLock()
	defer g.sw.RUnlock()
	snap := make(map[string]*Participant, len(g.Participants))
	maps.Copy(snap, g.Participants)
	return snap
}
func (g *Game) GetParticipants(filterIds ...string) []*Participant {
	out := make([]*Participant, 0, len(g.Participants))
	snap := g.GetSnapshotParticipants()
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
func (g *Game) CreateParticipant(conn *websocket.Conn, uid *string) *Participant {
	p := NewParticipant(g.Out, conn)
	if uid != nil && *uid != "" {
		p.Id = *uid
	}
	g.sw.Lock()
	g.Participants[p.Id] = p
	g.sw.Unlock()
	g.Out <- gameMessage{
		MessageType: MessageTypeUserJoined,
		Content: toJSON(map[string]any{
			"participant": p.Id,
		}),
		SentFrom: p.Id,
		Dest:     DestNotSender,
	}
	messageType := MessageGameCreatedSuccess
	if p.Id != g.OwnerId {
		messageType = MessageGameConnectSuccess
	}
	p.MsgIn <- gameMessage{
		MessageType: messageType,
		Content:     toJSON(g),
		Dest:        p.Id,
		SentFrom:    SentFromGame,
	}
	return p
}

func (g *Game) RemoveParticipant(pId string) {
	g.sw.Lock()
	p, found := g.Participants[pId]
	if !found {
		log.Printf("Participant pid=%s coud not be found", pId)
		g.sw.Unlock()
	}
	close(p.MsgIn)
	delete(g.Participants, pId)
	p.RemoveConnection()
	g.sw.Unlock()
}

func (g *Game) read() {
	for msg := range g.Out {
		if msg.MessageType == MessageTypeUserLeft {
			g.RemoveParticipant(msg.SentFrom)
		}
		snap := g.GetSnapshotParticipants()
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

func NewGame(name string, ownerId string) *Game {
	g := &Game{
		Id:                   uuid.NewString(),
		Participants:         make(map[string]*Participant),
		AnsweredParticipants: []int{},
		DateCreated:          time.Now(),
		CurrentRound:         0,
		Name:                 name,
		Status:               constants.GAMESTATUS_PENDING,
		sw:                   sync.RWMutex{},
		Out:                  make(chan gameMessage, 1025),
		OwnerId:              ownerId,
	}
	go g.read()
	return g
}
