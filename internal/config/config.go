package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Gemini   GeminiConfig   `yaml:"gemini"`
	Notion   NotionConfig   `yaml:"notion"`
	Database DatabaseConfig `yaml:"database"`
}

type TelegramConfig struct {
	Token string `yaml:"token"`
}

type GeminiConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type NotionConfig struct {
	APIKey           string `yaml:"api_key"`
	KanbanDatabaseID string `yaml:"kanban_database_id"`
	DocsDatabaseID   string `yaml:"docs_database_id"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

const defaultConfigPath = "config.yaml"

func Load(path string) (*Config, error) {
	if path == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
