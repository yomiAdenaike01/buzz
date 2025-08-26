package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
)

type Store struct {
	ongoingGames map[string]*game.Game
	questionsSet []config.Question
	config       config.AppConfig
}

func (s *Store) GetGameQuestions() []config.Question {
	return s.questionsSet
}
func (s *Store) GetOnGoingGames() map[string]*game.Game {
	return s.ongoingGames
}
func (s *Store) LoadRemoteChallengesFromFile() bool {
	configFilePath := s.config.GetConfigFilePath("posts.json")
	log.Printf("[LoadRemoteChallengesFromFile] Loading content from %s", configFilePath)
	_, err := os.Stat(configFilePath)
	if err != nil && errors.Is(err, os.ErrExist) {
		log.Println("[LoadRemoteChallengesFromFile]: File doesn't exist")
		return false
	}
	f, err := os.ReadFile(configFilePath)
	if err != nil {
		return false
	}
	var configQuestions []config.Question
	if err := json.NewDecoder(bytes.NewBuffer(f)).Decode(&configQuestions); err != nil {
		return false
	}
	s.AppendQuestions(configQuestions)
	return true
}

func (s *Store) SaveFetchedChallenges(questions []config.Question) error {
	configDir := s.config.GetConfigDir()
	savePath := filepath.Clean(filepath.Join(*configDir, "posts.json"))
	saveData, err := json.Marshal(questions)
	if err != nil {
		log.Printf("Failed to save data, err=%s", err.Error())
	}
	err = os.WriteFile(savePath, saveData, 0o600)
	if err != nil {
		log.Printf("Failed to save file err=%s", err.Error())
		return err
	}
	return err
}

func (s *Store) AppendQuestions(newQuestions []config.Question) {
	s.questionsSet = append(s.questionsSet, newQuestions...)
}

func (s *Store) Shutdown(ctx context.Context) error {
	wg := sync.WaitGroup{}
	done := make(chan struct{})
	errs := make(chan error, len(s.ongoingGames))
	wg.Add(len(s.ongoingGames))

	for _, ongoingGame := range s.ongoingGames {
		wg.Add(1)
		go func(og *game.Game) {
			defer wg.Done()
			err := og.Stop()
			errs <- err
		}(ongoingGame)
	}

	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return errors.New("failed to complete in time")
	}
	close(errs)

	return nil
}

func (s *Store) AddNewGame(g *game.Game) {
	s.ongoingGames[g.Id] = g
}
func (s *Store) RemoveGame(id string) {
	delete(s.ongoingGames, id)
	log.Printf("Removed game id=%s ongoing games=%d", id, len(s.ongoingGames))
}
func (s *Store) FindGameByShortCode(code string) *game.Game {
	for _, g := range s.ongoingGames {
		if g.Shortcode != code {
			continue
		}
		return g
	}
	return nil
}
func (s *Store) FindGameById(id string) *game.Game {
	if g, ok := s.ongoingGames[id]; ok {
		return g
	}
	return nil
}

func NewStore(config config.AppConfig) *Store {
	questions, err := loadChallenges()
	if err != nil {
		log.Printf("Failed to load challenges err=%v", err)
		return nil
	}
	return &Store{
		config:       config,
		questionsSet: *questions,
		ongoingGames: map[string]*game.Game{},
	}
}
