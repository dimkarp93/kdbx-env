package keyring

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dimkarp93/kdbx-env/internal/config"
)

func TestNewCache(t *testing.T) {
	if New(nil).enabled {
		t.Error("nil cache config should be disabled")
	}
	if New(&config.CacheConfig{Enabled: false}).enabled {
		t.Error("enabled=false should be disabled")
	}
	c := New(&config.CacheConfig{Enabled: true})
	if !c.enabled || c.ttl != defaultCacheTTL {
		t.Errorf("default ttl: %+v", c)
	}
	c = New(&config.CacheConfig{Enabled: true, TTL: "30s"})
	if c.ttl != 30*time.Second {
		t.Errorf("parsed ttl: %v", c.ttl)
	}
	c = New(&config.CacheConfig{Enabled: true, TTL: "garbage"})
	if c.ttl != defaultCacheTTL {
		t.Errorf("invalid ttl should fall back to default: %v", c.ttl)
	}
}

func TestCacheHitExpireForget(t *testing.T) {
	store := map[string]string{}
	origSet, origGet, origDel := keyringSet, keyringGet, keyringDelete
	keyringSet = func(_, user, secret string) error { store[user] = secret; return nil }
	keyringGet = func(_, user string) (string, error) {
		v, ok := store[user]
		if !ok {
			return "", fmt.Errorf("not found")
		}
		return v, nil
	}
	keyringDelete = func(_, user string) error { delete(store, user); return nil }
	defer func() { keyringSet, keyringGet, keyringDelete = origSet, origGet, origDel }()

	c := Cache{enabled: true, ttl: time.Minute}

	if _, ok := c.Get("/k.kdbx"); ok {
		t.Error("empty cache should miss")
	}
	c.Remember("/k.kdbx", "pw")
	if pw, ok := c.Get("/k.kdbx"); !ok || pw != "pw" {
		t.Errorf("hit: ok=%v pw=%q", ok, pw)
	}

	data, _ := json.Marshal(cachedSecret{Password: "old", StoredAt: time.Now().Add(-time.Hour).Unix()})
	store["/k.kdbx"] = string(data)
	if _, ok := c.Get("/k.kdbx"); ok {
		t.Error("expired entry should miss")
	}
	if _, present := store["/k.kdbx"]; present {
		t.Error("expired entry should be deleted on read")
	}

	disabled := Cache{}
	disabled.Remember("/x", "y")
	if _, ok := disabled.Get("/x"); ok {
		t.Error("disabled cache should never hit")
	}

	c.Remember("/k.kdbx", "pw2")
	c.Forget("/k.kdbx")
	if _, ok := c.Get("/k.kdbx"); ok {
		t.Error("forget should remove the entry")
	}
}
