package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/yomiAdenaike01/buzz-core/internals/handlers"
	"github.com/yomiAdenaike01/buzz-core/internals/store"
)

func main() {
	m := http.NewServeMux()
	port := ":9080"

	appStore := store.NewStore()
	h := handlers.Init(appStore)

	m.HandleFunc("POST /create-game", h.HandleCreateGame)
	m.HandleFunc("GET /all", h.HandleGamesList)
	m.HandleFunc("GET /connect", h.HandleConnectToGame)

	s := &http.Server{
		Handler: m,
		Addr:    port,
	}
	ctx, cancel := signal.NotifyContext(context.TODO(), os.Interrupt)

	defer cancel()
	go func() {
		log.Printf("Starting server on port=%s", port)
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Failed to start server at port %s err=%s", port, err.Error())
		}
	}()
	<-ctx.Done()
	log.Println("Server shutting down")
	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	appStore.Shutdown(c)

	if err := s.Shutdown(c); err != nil {
		panic(err)
	}
}
