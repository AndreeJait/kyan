package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type KyanConfig struct {
	AI struct {
		Model string `yaml:"model"`
		Host  string `yaml:"host"`
		Key   string `yaml:"key"`
	} `yaml:"ai"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".kyan"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func DefaultConfig() *KyanConfig {
	cfg := &KyanConfig{}
	cfg.AI.Model = "llama3"
	cfg.AI.Host = "http://localhost:11434"
	return cfg
}

func Load() (*KyanConfig, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

func Save(cfg *KyanConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir, _ := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

var validKeys = map[string]string{
	"ai.model": "AI.Model",
	"ai.host":  "AI.Host",
	"ai.key":   "AI.Key",
}

func Set(key, value string) error {
	if _, ok := validKeys[key]; !ok {
		return fmt.Errorf("invalid config key: %s (valid: ai.model, ai.host, ai.key)", key)
	}

	cfg, err := Load()
	if err != nil {
		return err
	}

	switch key {
	case "ai.model":
		cfg.AI.Model = value
	case "ai.host":
		cfg.AI.Host = value
	case "ai.key":
		cfg.AI.Key = value
	}

	return Save(cfg)
}