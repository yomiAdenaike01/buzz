package config

import (
	"embed"
	"encoding/json"
	"os"
)

//go:embed questions_by_category.json
var files embed.FS

type Question struct {
	Id       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
	Metadata string `json:"metadata"`
}

type AppConfig struct {
	CLOUD_FLARE_ACCOUNT_ID string
	CLOUD_FLARE_AI_MODEL   string
	CLOUD_FLARE_API_TOKEN  string
}

func NewConfig() AppConfig {
	return AppConfig{
		CLOUD_FLARE_ACCOUNT_ID: os.Getenv("CT_ACCOUNT_ID"),
		CLOUD_FLARE_AI_MODEL:   os.Getenv("CT_AI_MODEL"),
		CLOUD_FLARE_API_TOKEN:  os.Getenv("CT_API_TOKEN"),
	}
}

func LoadDeck() (*map[string][]Question, error) {
	data, err := files.ReadFile("questions_by_category.json")
	if err != nil {
		return nil, err
	}
	var questionsByCat map[string][]Question
	if err := json.Unmarshal(data, &questionsByCat); err != nil {
		return nil, err
	}
	return &questionsByCat, nil
}
