package handlers

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
	"github.com/yomiAdenaike01/buzz-core/internals/responses"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
)

type Handlers struct {
	store *store.Store
}

var u = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Init(store *store.Store) *Handlers {
	return &Handlers{
		store: store,
	}
}

func (h *Handlers) updgradeConnection(w http.ResponseWriter, r *http.Request) *websocket.Conn {
	conn, err := u.Upgrade(w, r, r.Header)
	if err != nil {
		return nil
	}
	return conn
}

func (h *Handlers) HandleConnectToGame(w http.ResponseWriter, r *http.Request) {
	// handle participation

	// find game by id
	gameId := string(r.URL.Query().Get("id"))
	uid := string(r.URL.Query().Get("user_id"))
	log.Printf("[handlers.HandleConnectToGame] Attempting to connect to game=%s", gameId)
	if gameId == "" {
		responses.WriteErrResponse(w, http.StatusBadRequest, "game doesn't exist")
	}
	g := h.store.FindGameById(gameId)
	log.Printf("[handlers.HandleConnectToGame] Found game by id=%s", gameId)

	if g == nil {
		responses.WriteErrResponse(w, http.StatusNotFound, "failed to find game")
	}
	// connect to socket
	conn := h.updgradeConnection(w, r)
	// create participant
	g.CreateParticipant(conn, &uid)
}

func (h *Handlers) HandleCreateGame(w http.ResponseWriter, r *http.Request) {
	log.Println("[handlers.HandleCreateGame] Attempting to create game...")

	uid := r.Context().Value("uid")
	gameName := string(r.URL.Query().Get("name"))
	log.Printf("[handlers.HandleCreateGame] Creating game name=%s", gameName)
	if uid == nil || uid == "" {
		uid = uuid.NewString()
	}
	uidString, ok := uid.(string)
	if !ok {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "failed to find user")
	}
	g := game.NewGame(gameName, uidString)
	h.store.AddNewGame(g)
	responses.WriteJSON(w, map[string]any{
		"id":       g.Id,
		"owner_id": uidString,
	})

}

func (h *Handlers) HandleGamesList(w http.ResponseWriter, r *http.Request) {
	log.Println("[handlers.HandleGamesList]: Fetching current games")
	ongoingGames := h.store.GetAllGames()
	responses.WriteJSON(w, map[string]any{
		"games": ongoingGames,
	})
}
