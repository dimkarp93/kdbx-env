package domain

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"secrets/internal/config"
)

func TestResolveMerge(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "~/store.kdbx", Secrets: map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "SHARED": "SHARED_ENV"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM_TOKEN", "SHARED": "OVERRIDE"}},
	}}

	r := Resolve(cfg, "install", "", map[string]string{})
	if r.KeyStore != "~/store.kdbx" {
		t.Errorf("keyStore: got %q", r.KeyStore)
	}
	want := map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "SHARED": "OVERRIDE", "NPM_TOKEN": "NPM_TOKEN"}
	if !reflect.DeepEqual(r.Secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.Secrets, want)
	}
}

func TestResolveFlagsOverride(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/from-cfg", Secrets: map[string]string{"A": "A1"}},
	}}
	r := Resolve(cfg, "unknown-tool", "/from-flag", map[string]string{"A": "A2", "B": "B1"})
	if r.KeyStore != "/from-flag" {
		t.Errorf("keyStore: got %q", r.KeyStore)
	}
	want := map[string]string{"A": "A2", "B": "B1"}
	if !reflect.DeepEqual(r.Secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.Secrets, want)
	}
}

func TestMappingsRoundTrip(t *testing.T) {
	m := map[string]string{"B": "2", "A": "1", "C": "3"}
	pairs := MappingsFromMap(m)
	if pairs[0].Name != "A" || pairs[1].Name != "B" || pairs[2].Name != "C" {
		t.Errorf("not sorted by name: %v", pairs)
	}
	if !reflect.DeepEqual(MappingsToMap(pairs), m) {
		t.Errorf("roundtrip mismatch: %v", MappingsToMap(pairs))
	}
}

func TestAggregateStores(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/a.kdbx", Secrets: map[string]string{"GITHUB_TOKEN": "GH"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM"}},
		"deploy":  {KeyStore: "/b.kdbx", Secrets: map[string]string{"AWS_KEY": "AWS"}},
	}}
	got := AggregateStores(cfg)

	wantA := []string{"GITHUB_TOKEN", "NPM_TOKEN"}
	if !reflect.DeepEqual(got["/a.kdbx"], wantA) {
		t.Errorf("/a.kdbx: got %v, want %v", got["/a.kdbx"], wantA)
	}
	wantB := []string{"AWS_KEY", "GITHUB_TOKEN"}
	if !reflect.DeepEqual(got["/b.kdbx"], wantB) {
		t.Errorf("/b.kdbx: got %v, want %v", got["/b.kdbx"], wantB)
	}
}

func TestBuildStoreViews(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "store.kdbx")
	if err := os.WriteFile(existing, nil, 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "absent.kdbx")

	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: existing, Secrets: map[string]string{"GITHUB_TOKEN": "GH"}},
		"deploy":  {KeyStore: missing, Secrets: map[string]string{"AWS_KEY": "AWS"}},
	}}
	views := BuildStoreViews(cfg)
	if len(views) != 2 {
		t.Fatalf("want 2 views, got %d: %+v", len(views), views)
	}
	byPath := map[string]StoreView{}
	for _, v := range views {
		byPath[v.Path] = v
	}
	if !byPath[existing].Exists {
		t.Errorf("existing store should be marked exists")
	}
	if byPath[missing].Exists {
		t.Errorf("missing store should be marked not exists")
	}
	if len(byPath[missing].Mappings) != 2 {
		t.Errorf("deploy store mappings: %+v", byPath[missing].Mappings)
	}
}
