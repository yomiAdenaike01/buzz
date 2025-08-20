package handlers

import (
	"encoding/json"
	"io"
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

	gameId := string(r.URL.Query().Get("id"))
	playerId := string(r.URL.Query().Get("player_id"))
	name := string(r.URL.Query().Get("name"))
	code := string(r.URL.Query().Get("code"))

	if gameId == "" && code == "" {
		responses.WriteErrResponse(w, http.StatusBadRequest, "required params, code or game id are required")
		return
	}

	log.Printf("[handlers.HandleConnectToGame] Attempting to connect to game=%s", gameId)
	var g *game.Game
	// find game by identifier
	if gameId != "" {
		g = h.store.FindGameById(gameId)
	}
	if code != "" {
		g = h.store.FindGameByShortCode(code)
	}

	log.Printf("[handlers.HandleConnectToGame] Found game by id=%s", gameId)

	if g == nil {
		responses.WriteErrResponse(w, http.StatusNotFound, "failed to find game")
		return
	}
	// connect to socket
	conn := h.updgradeConnection(w, r)
	if conn == nil {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "failed to establish a connection")
		return
	}
	// create participant
	g.CreatePlayer(game.BaseCreatePlayerInput{
		Connection: conn,
		PlayerDetails: game.PlayerDetails{
			Id:   playerId,
			Name: name,
		},
	})
}

func (h *Handlers) HandleCreateGame(w http.ResponseWriter, r *http.Request) {
	log.Println("[handlers.HandleCreateGame] Attempting to create game...")

	uid := r.Context().Value("uid")
	d, err := io.ReadAll(r.Body)
	if err != nil {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "request failed")
	}
	var createGameInput struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(d, &createGameInput); err != nil {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "failed to parse input")
	}
	log.Printf("[handlers.HandleCreateGame] Creating game name=%s", createGameInput.Name)
	if uid == nil || uid == "" {
		uid = uuid.NewString()
	}
	uidString, ok := uid.(string)
	if !ok {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "failed to find user")
	}
	g := game.NewGame(game.NewGameInput{
		Name:      createGameInput.Name,
		OwnerId:   uidString,
		Questions: h.store.GetGameQuestions(),
	})
	h.store.AddNewGame(g)
	responses.WriteJSON(w, map[string]any{
		"id":  g.Id,
		"pid": uidString,
	})

}

func (h *Handlers) HandleGamesList(w http.ResponseWriter, r *http.Request) {
	log.Println("[handlers.HandleGamesList]: Fetching current games")
	ongoingGames := h.store.GetAllGames()
	responses.WriteJSON(w, map[string]any{
		"games": ongoingGames,
	})
}

func (h *Handlers) HandleGetPrompts(w http.ResponseWriter, r *http.Request) {
	questions := h.store.GetGameQuestions()
	rawQuestions, err := json.Marshal(questions)
	if err != nil {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "failed to marshal questions")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rawQuestions)
}
