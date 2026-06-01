package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

type Section struct {
	KeyStore string            `json:"key-store,omitempty"`
	Secrets  map[string]string `json:"secrets,omitempty"`
}

type Config map[string]Section

type resolved struct {
	keyStore string
	secrets  map[string]string
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return c, nil
}

func saveConfig(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func resolve(cfg Config, tool string, flags runFlags) resolved {
	r := resolved{secrets: map[string]string{}}
	apply := func(s Section) {
		if s.KeyStore != "" {
			r.keyStore = s.KeyStore
		}
		maps.Copy(r.secrets, s.Secrets)
	}
	if s, ok := cfg["default"]; ok {
		apply(s)
	}
	if tool != "" && tool != "default" {
		if s, ok := cfg[tool]; ok {
			apply(s)
		}
	}
	if flags.keyStore != "" {
		r.keyStore = flags.keyStore
	}
	maps.Copy(r.secrets, flags.secrets)
	return r
}
