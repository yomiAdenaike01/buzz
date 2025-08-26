package config

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
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
	CONFIG_DIR             string
}

func (a AppConfig) GetConfigFilePath(filename string) string {
	configDir := a.GetConfigDir()
	if configDir == nil {
		return ""
	}
	return filepath.Clean(filepath.Join(*configDir, filename))
}

func (a AppConfig) GetConfigDir() *string {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	path := filepath.Clean(filepath.Join(cwd, "../", a.CONFIG_DIR))
	return &path

}
func NewConfig() AppConfig {
	return AppConfig{
		CLOUD_FLARE_ACCOUNT_ID: os.Getenv("CT_ACCOUNT_ID"),
		CLOUD_FLARE_AI_MODEL:   os.Getenv("CT_AI_MODEL"),
		CLOUD_FLARE_API_TOKEN:  os.Getenv("CT_API_TOKEN"),
		CONFIG_DIR:             os.Getenv("CONFIG_DIR"),
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
