package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Section struct {
	KeyStore string            `json:"key-store,omitempty"`
	Secrets  map[string]string `json:"secrets,omitempty"`
}

type CacheConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	TTL     string `json:"ttl,omitempty"`
}

type Config struct {
	Sections map[string]Section `json:"sections"`
	Cache    *CacheConfig       `json:"cached,omitempty"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	if c.Sections == nil {
		c.Sections = map[string]Section{}
	}
	return c, nil
}

func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if c.Sections == nil {
		c.Sections = map[string]Section{}
	}
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, append(data, '\n'), 0600)
}
