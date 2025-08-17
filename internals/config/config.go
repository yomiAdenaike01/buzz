package config

import (
	"embed"
	"encoding/json"
)

//go:embed questions_by_category.json
var files embed.FS

type Question struct {
	Id   string `json:"id"`
	Text string `json:"text"`
}

func LoadDeck() (map[string][]Question, error) {
	data, err := files.ReadFile("questions_by_category.json")
	if err != nil {
		return nil, err
	}
	var questionsByCat map[string][]Question
	if err := json.Unmarshal(data, &questionsByCat); err != nil {
		return nil, err
	}
	return questionsByCat, nil
}
