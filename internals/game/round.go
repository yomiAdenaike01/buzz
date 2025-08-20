package game

import (
	"time"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
)

type Round struct {
	questionsSet      []config.Question
	answeredQuestions map[string]struct{}
	answeredPlayers   map[string]struct{}
	order             []string
	turnIndex         int
	ownerId           string
	roundCount        int
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
	for _, existingQuestion := range r.questionsSet {
		if _, ok := r.answeredQuestions[existingQuestion.Id]; !ok {
			r.answeredQuestions[existingQuestion.Id] = struct{}{}
			return existingQuestion
		}
		continue
	}
	return config.Question{}
}

func NewRound(questionSet []config.Question, order []string, duration time.Duration, ownerId string) *Round {
	return &Round{
		turnIndex:         0,
		questionsSet:      questionSet,
		order:             order,
		answeredPlayers:   make(map[string]struct{}, len(order)),
		answeredQuestions: make(map[string]struct{}, len(questionSet)),
		roundCount:        1,
	}
}
