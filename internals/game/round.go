package game

import (
	"log"
	"sync"
	"time"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
)

type Round struct {
	questionsSet      []config.Question
	answeredQuestions map[string]struct{}
	answeredPlayers   map[string]struct{}
	order             []string
	turnIndex         int
	roundCount        int
	mu                sync.RWMutex
}

func (r *Round) getTurnIndex() int {
	return r.turnIndex
}

func (r *Round) nextAsker() string {
	if len(r.answeredPlayers) == len(r.order) {
		r.answeredPlayers = make(map[string]struct{}, len(r.order))
	}
	for _, id := range r.order {
		if _, ok := r.answeredPlayers[id]; !ok {
			r.answeredPlayers[id] = struct{}{}
			r.turnIndex++
			return id
		}
		continue
	}
	return ""

}

func (r *Round) nextQuestion() config.Question {
	for _, question := range r.questionsSet {
		if _, ok := r.answeredQuestions[question.Id]; !ok {
			r.mu.RLock()
			playersCount := len(r.order)
			r.mu.RUnlock()
			if question.CanVote() && playersCount < 3 {
				log.Println("[round.nextQuestion] skipping question")
				r.answeredQuestions[question.Id] = struct{}{}
				continue
			}
			r.answeredQuestions[question.Id] = struct{}{}
			return question
		}
		continue
	}
	return config.Question{}
}

func (r *Round) AddPlayer(playerId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, playerId)
}

func NewRound(questionSet []config.Question, order []string, duration time.Duration, ownerId string) *Round {
	return &Round{
		turnIndex:         0,
		questionsSet:      questionSet,
		order:             order,
		answeredPlayers:   make(map[string]struct{}, len(order)),
		answeredQuestions: make(map[string]struct{}, len(questionSet)),
		roundCount:        1,
		mu:                sync.RWMutex{},
	}
}
