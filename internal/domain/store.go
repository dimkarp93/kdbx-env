package domain

import (
	"fmt"
	"maps"
	"os"
	"sort"

	"github.com/dimkarp93/kdbx-env/internal/config"
	"github.com/dimkarp93/kdbx-env/internal/keepass"
	"github.com/dimkarp93/kdbx-env/internal/keyring"
	"github.com/dimkarp93/kdbx-env/internal/term"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func AggregateStores(cfg config.Config) map[string][]string {
	sets := map[string]map[string]bool{}
	for section := range cfg.Sections {
		res := Resolve(cfg, section, "", map[string]string{})
		if res.KeyStore == "" || len(res.Secrets) == 0 {
			continue
		}
		ks := config.ExpandHome(res.KeyStore)
		if sets[ks] == nil {
			sets[ks] = map[string]bool{}
		}
		for name := range res.Secrets {
			sets[ks][name] = true
		}
	}
	out := make(map[string][]string, len(sets))
	for ks, set := range sets {
		titles := make([]string, 0, len(set))
		for t := range set {
			titles = append(titles, t)
		}
		sort.Strings(titles)
		out[ks] = titles
	}
	return out
}

func AggregateStoreMappings(cfg config.Config) map[string][]Mapping {
	sets := map[string]map[string]string{}
	for section := range cfg.Sections {
		res := Resolve(cfg, section, "", map[string]string{})
		if res.KeyStore == "" || len(res.Secrets) == 0 {
			continue
		}
		ks := config.ExpandHome(res.KeyStore)
		if sets[ks] == nil {
			sets[ks] = map[string]string{}
		}
		maps.Copy(sets[ks], res.Secrets)
	}
	out := make(map[string][]Mapping, len(sets))
	for ks, m := range sets {
		out[ks] = MappingsFromMap(m)
	}
	return out
}

type storeState struct {
	path     string
	exists   bool
	password string
	missing  []string
}

func gatherMissing(byStore map[string][]string, cache keyring.Cache) []storeState {
	stores := make([]string, 0, len(byStore))
	for ks := range byStore {
		stores = append(stores, ks)
	}
	sort.Strings(stores)

	var states []storeState
	for _, ks := range stores {
		titles := byStore[ks]
		if len(titles) == 0 {
			continue
		}
		s := storeState{path: ks}
		if !fileExists(ks) {
			s.missing = append([]string{}, titles...)
			states = append(states, s)
			continue
		}
		s.exists = true
		out, pw, err := UnlockExport(ks, cache, fmt.Sprintf("Enter password for %s: ", ks))
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", ks, err)
			continue
		}
		s.password = pw
		entries, _ := keepass.ParseSecrets(out)
		for _, t := range titles {
			if _, e := keepass.LookupSecret(entries, t); e != nil {
				s.missing = append(s.missing, t)
			}
		}
		states = append(states, s)
	}
	return states
}

func ReconcileStores(byStore map[string][]string, assumeYes bool, cache keyring.Cache) {
	keepass.CheckEngine()
	states := gatherMissing(byStore, cache)

	total := 0
	for _, s := range states {
		total += len(s.missing)
	}
	if total == 0 {
		fmt.Println("All secrets are present in the key-store(s).")
		return
	}

	fmt.Println("Missing secrets:")
	for _, s := range states {
		if len(s.missing) == 0 {
			continue
		}
		suffix := ""
		if !s.exists {
			suffix = "  (key-store does not exist)"
		}
		fmt.Printf("  %s%s\n", s.path, suffix)
		for _, t := range s.missing {
			fmt.Printf("    - %s\n", t)
		}
	}

	for i := range states {
		s := &states[i]
		if len(s.missing) == 0 {
			continue
		}
		if !s.exists {
			if !term.Confirm(fmt.Sprintf("Create key-store %s and add %d empty secret(s)?", s.path, len(s.missing)), assumeYes) {
				fmt.Println("Skipped", s.path)
				continue
			}
			pw := term.ReadPassword("Set a password for the new key-store " + s.path + ": ")
			if err := keepass.CreateStore(s.path, pw); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			s.password = pw
			cache.Remember(s.path, pw)
			fmt.Println("Created key-store", s.path)
		} else if !term.Confirm(fmt.Sprintf("Add %d empty secret(s) to %s?", len(s.missing), s.path), assumeYes) {
			fmt.Println("Skipped", s.path)
			continue
		}
		for _, t := range s.missing {
			if err := keepass.AddEmptySecret(s.path, s.password, t); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			fmt.Printf("  + %s → %s\n", t, s.path)
		}
	}
}

type StoreView struct {
	Path     string
	Exists   bool
	Mappings []Mapping
}

func BuildStoreViews(cfg config.Config) []StoreView {
	byStore := AggregateStoreMappings(cfg)
	paths := make([]string, 0, len(byStore))
	for p := range byStore {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	views := make([]StoreView, 0, len(paths))
	for _, p := range paths {
		views = append(views, StoreView{
			Path:     p,
			Exists:   fileExists(p),
			Mappings: byStore[p],
		})
	}
	return views
}
