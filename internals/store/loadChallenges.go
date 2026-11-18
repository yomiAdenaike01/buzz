package store

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
)

func loadChallenges() (*[]config.Question, error) {
	questionsByCat, loadErr := config.LoadDeck()
	maxQuestions := 80
	questionSet := make([]config.Question, 0, maxQuestions)
	if loadErr != nil {
		return nil, loadErr
	}
	seenQuestions := make(map[string]struct{}, maxQuestions)
	for category, questionsList := range *questionsByCat {
		for range maxQuestions / 2 {
			r, err := rand.Int(rand.Reader, big.NewInt(int64(len(questionsList))))
			if err != nil {
				return nil, err
			}

			currentQuestion := questionsList[r.Int64()]
			if _, ok := seenQuestions[currentQuestion.Id]; ok {
				continue
			}
			currentQuestion.Category = category
			questionSet = append(questionSet, currentQuestion)
			seenQuestions[currentQuestion.Id] = struct{}{}
		}
	}
	if len(questionSet) == 0 {
		return nil, errors.New("no questions loaded")
	}
	return &questionSet, nil
}
