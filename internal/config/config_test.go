package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg")
	if err := os.WriteFile(path, []byte(`{"sections":{"default":{"key-store":"/k.kdbx"}},"cached":{"enabled":true,"ttl":"5m"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sections["default"].KeyStore != "/k.kdbx" {
		t.Errorf("sections: %+v", c.Sections)
	}
	if c.Cache == nil || !c.Cache.Enabled || c.Cache.TTL != "5m" {
		t.Errorf("cache: %+v", c.Cache)
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg")
	c := Config{Sections: map[string]Section{"default": {KeyStore: "/k.kdbx"}}}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"sections"`) {
		t.Errorf("saved file should contain sections:\n%s", data)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Sections["default"].KeyStore != "/k.kdbx" {
		t.Errorf("reload after save: %+v", c2.Sections)
	}
}
