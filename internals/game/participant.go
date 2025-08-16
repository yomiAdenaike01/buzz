package game

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Participant struct {
	Id         string
	connection *websocket.Conn
	MsgIn      chan gameMessage
	MsgOut     chan<- gameMessage
	once       sync.Once
	sw         sync.Mutex
}

const (
	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second // keep under typical 60s idle timeouts
	pongWait   = 3 * pingPeriod   // tolerate a couple missed pongs (≈90s)
)

func (p *Participant) listen() {
	c := p.connection
	c.SetReadDeadline(time.Now().Add(pongWait))
	c.SetPongHandler(func(appData string) error {
		p.sw.Lock()
		defer p.sw.Unlock()
		return c.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {

		_, content, err := c.ReadMessage()
		if err != nil {
			// remove connection
			p.RemoveConnection()
			return
		}
		if len(content) == 0 {
			continue
		}
		var g gameMessage
		if err := json.Unmarshal(content, &g); err != nil {
			log.Printf("Failed to handle game message err=%s", err.Error())
			continue
		}
		g.SentFrom = p.Id
		// send to other participants
		select {
		case p.MsgOut <- g:
		default:
		}
	}
}

func (p *Participant) RemoveConnection() {
	log.Printf("Attempting to remove connection for participantId=%s", p.Id)
	p.once.Do(func() {
		select {
		case p.MsgOut <- gameMessage{
			SentFrom:    p.Id,
			MessageType: MessageTypeUserLeft,
			Dest:        DestNotSender,
			Content:     fmt.Sprintf("User %s has left", p.Id),
		}:
		default:
		}
		if p == nil {
			return
		}
		c := p.connection
		if c == nil {
			return
		}
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Normal closure")
		p.sw.Lock()
		c.WriteControl(
			websocket.CloseMessage,
			msg,
			time.Now().Add(writeWait),
		)
		c.Close()
		p.sw.Unlock()
	})

}

func (p *Participant) write() {
	t := time.NewTicker(pingPeriod)
	defer func() {
		t.Stop()
		if p.connection != nil {
			p.connection.Close()
		}
	}()
	for {
		select {
		case msg, ok := <-p.MsgIn:
			c := p.connection
			if !ok {
				p.RemoveConnection()
				return
			}
			d, err := json.Marshal(msg)
			if err != nil {
				panic(err)
			}
			if c == nil {
				return
			}
			err = c.WriteMessage(websocket.TextMessage, d)
			if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				// remove connection
				p.RemoveConnection()
				return
			}
		case <-t.C:
			if p.connection == nil {
				return
			}
			if err := p.connection.WriteControl(
				websocket.PingMessage,
				nil,
				time.Now().Add(writeWait),
			); err != nil {
				p.RemoveConnection()
			}
			log.Printf("Sending ping to participant pid=%s", p.Id)
		}
	}

}

func NewParticipant(outChan chan<- gameMessage, conn *websocket.Conn) *Participant {
	p := &Participant{
		Id:         uuid.NewString(),
		MsgIn:      make(chan gameMessage, 1025),
		MsgOut:     outChan,
		connection: conn,
		once:       sync.Once{},
		sw:         sync.Mutex{},
	}
	go p.listen()
	go p.write()
	return p
}
