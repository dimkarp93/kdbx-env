package domain

import (
	"maps"

	"github.com/dimkarp93/kdbx-env/internal/config"
)

type Resolved struct {
	KeyStore string
	Secrets  map[string]string
}

func Resolve(cfg config.Config, tool, keyStoreOverride string, secretsOverride map[string]string) Resolved {
	r := Resolved{Secrets: map[string]string{}}
	apply := func(s config.Section) {
		if s.KeyStore != "" {
			r.KeyStore = s.KeyStore
		}
		maps.Copy(r.Secrets, s.Secrets)
	}
	if s, ok := cfg.Sections["default"]; ok {
		apply(s)
	}
	if tool != "" && tool != "default" {
		if s, ok := cfg.Sections[tool]; ok {
			apply(s)
		}
	}
	if keyStoreOverride != "" {
		r.KeyStore = keyStoreOverride
	}
	maps.Copy(r.Secrets, secretsOverride)
	return r
}
