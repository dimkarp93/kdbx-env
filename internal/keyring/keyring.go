package keyring

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/zalando/go-keyring"

	"secrets/internal/config"
)

const (
	keyringService  = "secrets"
	defaultCacheTTL = 10 * time.Minute
)

var (
	keyringSet    = keyring.Set
	keyringGet    = keyring.Get
	keyringDelete = keyring.Delete
)

type cachedSecret struct {
	Password string `json:"password"`
	StoredAt int64  `json:"stored_at"`
}

type Cache struct {
	enabled bool
	ttl     time.Duration
}

func New(c *config.CacheConfig) Cache {
	if c == nil || !c.Enabled {
		return Cache{}
	}
	ttl := defaultCacheTTL
	if c.TTL != "" {
		if d, err := time.ParseDuration(c.TTL); err == nil && d > 0 {
			ttl = d
		}
	}
	return Cache{enabled: true, ttl: ttl}
}

func (c Cache) Get(keyStore string) (string, bool) {
	if !c.enabled {
		return "", false
	}
	raw, err := keyringGet(keyringService, keyStore)
	if err != nil {
		return "", false
	}
	var cs cachedSecret
	if json.Unmarshal([]byte(raw), &cs) != nil {
		return "", false
	}
	if time.Since(time.Unix(cs.StoredAt, 0)) > c.ttl {
		_ = keyringDelete(keyringService, keyStore)
		return "", false
	}
	return cs.Password, true
}

func (c Cache) Remember(keyStore, password string) {
	if !c.enabled {
		return
	}
	data, _ := json.Marshal(cachedSecret{Password: password, StoredAt: time.Now().Unix()})
	if err := keyringSet(keyringService, keyStore, string(data)); err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot cache password:", err)
	}
}

func (c Cache) Forget(keyStore string) {
	_ = keyringDelete(keyringService, keyStore)
}

type CacheStatus struct {
	Cached    bool
	ExpiresIn time.Duration
}

func (c Cache) Status(keyStore string) CacheStatus {
	if !c.enabled {
		return CacheStatus{}
	}
	raw, err := keyringGet(keyringService, keyStore)
	if err != nil {
		return CacheStatus{}
	}
	var cs cachedSecret
	if json.Unmarshal([]byte(raw), &cs) != nil {
		return CacheStatus{}
	}
	remaining := c.ttl - time.Since(time.Unix(cs.StoredAt, 0))
	if remaining <= 0 {
		return CacheStatus{}
	}
	return CacheStatus{Cached: true, ExpiresIn: remaining}
}
