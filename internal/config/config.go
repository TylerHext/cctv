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
	NewSession NewSession `toml:"new_session"`
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
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
