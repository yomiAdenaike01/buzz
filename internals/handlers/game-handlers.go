package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
	"github.com/yomiAdenaike01/buzz-core/internals/responses"
	serviceClients "github.com/yomiAdenaike01/buzz-core/internals/service-clients"
	"github.com/yomiAdenaike01/buzz-core/internals/services"
)

type ConnectState struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Token    string `json:"token"`
	PlayerId string `json:"playerId"`
	Id       string `json:"id"`
}

type Handlers struct {
	gameService         *services.GameService
	notificationClient  *serviceClients.NotificationClient
	notificationChannel chan serviceClients.NotificationContent
}

var u = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewHandlers(gameService *services.GameService) *Handlers {
	return &Handlers{
		gameService: gameService,
	}
}

func (h *Handlers) updgradeConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := u.Upgrade(w, r, r.Header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func getQueryStateParam(urlValues url.Values) (ConnectState, error) {
	state := urlValues.Get("state")
	var connectPayload ConnectState
	decoded, err := base64.StdEncoding.DecodeString(state)
	if err != nil {
		return connectPayload, nil
	}

	if err := json.Unmarshal(decoded, &connectPayload); err != nil {
		return connectPayload, nil
	}
	return connectPayload, nil
}

func (h *Handlers) HandleConnectToGame(w http.ResponseWriter, r *http.Request) {
	stateQuery := r.URL.Query().Get("state")
	state, err := getQueryStateParam(r.URL.Query())
	if err != nil {
		responses.WriteErrResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to parse state=%s", stateQuery))
		return
	}

	if state.Id == "" && state.Code == "" {
		responses.WriteErrResponse(w, http.StatusBadRequest, "required params, code or game id are required")
		return
	}

	log.Printf("[handlers.HandleConnectToGame] Attempting to connect to game=%s", state.Id)
	var gameInstance *game.Game

	if state.Id != "" {
		gameInstance = h.gameService.FindGameById(state.Id)
	}
	if state.Code != "" {
		gameInstance = h.gameService.FindGameByShortCode(state.Code)
	}

	log.Printf("[handlers.HandleConnectToGame] Found game by id=%s", state.Id)

	if gameInstance == nil {
		responses.WriteErrResponse(w, http.StatusNotFound, "failed to find game")
		return
	}

	conn, err := h.updgradeConnection(w, r)
	if err != nil {
		responses.WriteErrResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to establish a connection reason=%s", err.Error()))
		return
	}

	gameInstance.CreatePlayer(game.BaseCreatePlayerInput{
		Connection: conn,
		PlayerDetails: game.PlayerDetails{
			Id:                state.PlayerId,
			Name:              state.Name,
			NotificationToken: state.Token,
		},
	})
}

func (h *Handlers) HandleCreateGame(w http.ResponseWriter, r *http.Request) {
	log.Println("[handlers.HandleCreateGame] Attempting to create game...")
	// posts, err := h.gameService.GetRedditPosts()
	// if err != nil {
	// 	log.Printf("Failed to get reddit posts err=%s", err.Error())
	// }
	// log.Printf("Posts =%v", posts)

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
	playerId := uuid.NewString()
	gameInstance := h.gameService.CreateGame(game.NewGameInput{
		Name:    createGameInput.Name,
		OwnerId: playerId,
	})
	responses.WriteJSON(w, map[string]any{
		"id":  gameInstance.Id,
		"pid": playerId,
	})

}

func (h *Handlers) HandleGetPrompts(w http.ResponseWriter, r *http.Request) {
	questions := h.gameService.GetQuestions()
	rawQuestions, err := json.Marshal(questions)
	if err != nil {
		responses.WriteErrResponse(w, http.StatusInternalServerError, "failed to marshal questions")
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rawQuestions)
}

func (h *Handlers) HandleHealtchCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}
