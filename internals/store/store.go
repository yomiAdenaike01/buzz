package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yomiAdenaike01/buzz-core/internals/constants"
	"github.com/yomiAdenaike01/buzz-core/internals/game"
)

type Store struct {
	ongoingGames map[string]*game.Game
}

func (s *Store) Shutdown(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, time.Second*10)
	wg := sync.WaitGroup{}
	done := make(chan struct{})
	errs := make(chan error, len(s.ongoingGames))
	wg.Add(len(s.ongoingGames))
	defer cancel()

	for _, ongoingGame := range s.ongoingGames {
		wg.Add(1)
		go func(og *game.Game) {
			defer wg.Done()
			err := og.Stop(c)
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
func (s *Store) GetAllGames() []*game.Game {
	if len(s.ongoingGames) == 0 {
		return []*game.Game{}
	}
	arr := make([]*game.Game, 0, len(s.ongoingGames))
	for _, v := range s.ongoingGames {
		if v.Status == constants.GAMESTATUS_ENDED {
			continue
		}
		arr = append(arr, v)
	}
	return arr
}
func NewStore() *Store {
	return &Store{
		ongoingGames: map[string]*game.Game{},
	}
}
