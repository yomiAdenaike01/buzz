package store

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/yomiAdenaike01/buzz-core/internals/config"
)

func loadChallenges() ([]config.Question, error) {
	questionsByCat, loadErr := config.LoadDeck()
	questionSet := make([]config.Question, 0, 80)
	if loadErr != nil {
		return nil, loadErr
	}

	for category, questionsList := range questionsByCat {
		for range questionsList {
			l := len(questionsByCat)
			r, err := rand.Int(rand.Reader, big.NewInt(int64(l)))
			if err != nil {
				return nil, err
			}
			currentQuestion := questionsList[r.Int64()]
			currentQuestion.Category = category
			questionSet = append(questionSet, currentQuestion)
		}
	}
	if len(questionSet) == 0 {
		return nil, errors.New("no questions loaded")
	}
	return questionSet, nil
}
