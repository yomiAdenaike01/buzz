package game

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yomiAdenaike01/buzz-core/internals/constants"
)

type PlayerDetails struct {
	Name              string `json:"name"`
	Id                string `json:"id"`
	NotificationToken string `json:"-"`
}
type BaseCreatePlayerInput struct {
	Connection *websocket.Conn `json:"-"`
	PlayerDetails
}
type CreatePlayerInput struct {
	Out chan<- gameMessage `json:"-"`
	BaseCreatePlayerInput
}
type Player struct {
	Id         string             `json:"id"`
	connection *websocket.Conn    `json:"-"`
	MsgIn      chan gameMessage   `json:"-"`
	MsgOut     chan<- gameMessage `json:"-"`
	once       sync.Once          `json:"-"`
	sw         sync.Mutex         `json:"-"`
	Name       string             `json:"name"`
	ticker     *time.Ticker       `json:"-"`
}

const (
	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second // keep under typical 60s idle timeouts
	pongWait   = 3 * pingPeriod   // tolerate a couple missed pongs (≈90s)
)

func (p *Player) Consume(msg gameMessage) bool {
	timer := time.NewTimer(constants.MAX_EMIT_TIMEOUT)
	defer timer.Stop()
	select {
	case <-timer.C:
		// might be slow consider dropping
		log.Println("[Player.Consume]: Failed to send message in duration, dropping player")
		p.RemoveConnection()
		return false
	case p.MsgIn <- msg:
		return true
	}
}

func (p *Player) listen() {
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
			log.Println("[listen]: Failed to send message")
		}
	}
}

func (p *Player) RemoveConnection() {
	p.once.Do(func() {
		log.Printf("Attempting to remove connection for participantId=%s", p.Id)
		if p.ticker != nil {
			p.ticker.Stop()
			p.ticker = nil
		}
		p.sw.Lock()
		c := p.connection
		p.connection = nil
		p.sw.Unlock()
		if c == nil {
			return
		}
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Normal closure")
		c.WriteControl(
			websocket.CloseMessage,
			msg,
			time.Now().Add(writeWait),
		)
		c.Close()
	})

}

func (p *Player) write() {
	t := time.NewTicker(pingPeriod)
	p.ticker = t
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
				t.Stop()
			}
			log.Printf("Sending ping to participant pid=%s", p.Id)
		}
	}

}

func NewPlayer(input CreatePlayerInput) *Player {
	if input.Id == "" {
		input.Id = uuid.NewString()
	}
	p := &Player{
		Id:         input.Id,
		MsgIn:      make(chan gameMessage, 1025),
		MsgOut:     input.Out,
		connection: input.Connection,
		once:       sync.Once{},
		sw:         sync.Mutex{},
		Name:       input.Name,
	}
	go p.listen()
	go p.write()
	return p
}
