package domain

import (
	"fmt"

	"secrets/internal/keepass"
	"secrets/internal/keyring"
	"secrets/internal/term"
)

func UnlockExport(keyStore string, cache keyring.Cache, prompt string) (string, string, error) {
	if pw, ok := cache.Get(keyStore); ok {
		out, err := keepass.Run(pw+"\n", "export", "-q", "-f", "xml", keyStore)
		if err == nil {
			return out, pw, nil
		}
		cache.Forget(keyStore)
	}

	pw := term.ReadPassword(prompt)
	if pw == "" {
		return "", "", fmt.Errorf("password cannot be empty")
	}
	out, err := keepass.Run(pw+"\n", "export", "-q", "-f", "xml", keyStore)
	if err != nil {
		return "", "", fmt.Errorf("keepassxc-cli export: %w: %s", err, out)
	}
	cache.Remember(keyStore, pw)
	return out, pw, nil
}
