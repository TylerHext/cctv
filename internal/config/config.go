package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type NewSession struct {
	Commands []string `toml:"commands"`
	Layout   string   `toml:"layout"`
}

type Config struct {
	NewSession    NewSession `toml:"new_session"`
	WorkspacePath string     `toml:"workspace_path"`
	ReposPath     string     `toml:"repos_path"`
	NotesPath     string     `toml:"notes_path"`
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cctv")
}

func SessionsDir() string {
	return filepath.Join(Dir(), "sessions")
}

func Load() (Config, error) {
	cfg := Config{}

	path := filepath.Join(Dir(), "config.toml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		applyDefaults(&cfg)
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	applyDefaults(&cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.WorkspacePath == "" {
		home, _ := os.UserHomeDir()
		cfg.WorkspacePath = filepath.Join(home, "workspace")
	}
	if cfg.ReposPath == "" {
		home, _ := os.UserHomeDir()
		cfg.ReposPath = filepath.Join(home, "repositories")
	}
	if cfg.NotesPath == "" {
		home, _ := os.UserHomeDir()
		cfg.NotesPath = filepath.Join(home, "notes")
	}
}
