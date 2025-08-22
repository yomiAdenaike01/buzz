package store

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
)

type Store struct {
	ongoingGames map[string]*game.Game
	questionsSet []config.Question
}

func (s *Store) GetGameQuestions() []config.Question {
	return s.questionsSet
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

func NewStore() *Store {
	questions, err := loadChallenges()
	if err != nil {
		log.Printf("Failed to load challenges err=%v", err)
		return nil
	}
	return &Store{
		questionsSet: *questions,
		ongoingGames: map[string]*game.Game{},
	}
}
