package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dimkarp93/kdbx-env/internal/config"
	"github.com/dimkarp93/kdbx-env/internal/domain"
	"github.com/dimkarp93/kdbx-env/internal/keyring"
	"github.com/dimkarp93/kdbx-env/internal/term"
)

func cmdConfig(path string, assumeYes bool) {
	if path == "" {
		path = config.DefaultPath()
	}

	cfg := config.Config{}
	if c, err := config.Load(path); err == nil {
		cfg = c
	}
	if cfg.Sections == nil {
		cfg.Sections = map[string]config.Section{}
	}
	def := cfg.Sections["default"]
	pairs := domain.MappingsFromMap(def.Secrets)

	var keyStore string
	var canceled bool
	var newCache *config.CacheConfig
	if term.IsInteractive() {
		ks, ps, cc, c, err := runConfigTUI(def.KeyStore, pairs, cfg.Cache)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tui error:", err)
			os.Exit(1)
		}
		keyStore, pairs, newCache, canceled = ks, ps, cc, c
	} else {
		keyStore, pairs = configFallback(def.KeyStore, pairs)
		newCache = cfg.Cache
	}

	if canceled {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		os.Exit(1)
	}

	def.KeyStore = keyStore
	def.Secrets = domain.MappingsToMap(pairs)
	cfg.Sections["default"] = def
	cfg.Cache = newCache

	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "cannot write config:", err)
		os.Exit(1)
	}
	fmt.Println("Config saved to", path)

	if keyStore != "" && len(pairs) > 0 {
		fmt.Println()
		domain.ReconcileStores(map[string][]string{config.ExpandHome(keyStore): titlesOf(pairs)}, assumeYes, keyring.New(cfg.Cache))
	}
}

func cmdCheck(path string, assumeYes bool) {
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load config", path+":", err)
		os.Exit(1)
	}
	byStore := domain.AggregateStores(cfg)
	if len(byStore) == 0 {
		fmt.Println("No secrets configured in", path)
		return
	}
	domain.ReconcileStores(byStore, assumeYes, keyring.New(cfg.Cache))
}

func cmdForget(path string) {
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load config", path+":", err)
		os.Exit(1)
	}
	byStore := domain.AggregateStores(cfg)
	if len(byStore) == 0 {
		fmt.Println("No key-stores configured in", path)
		return
	}
	cache := keyring.New(cfg.Cache)
	stores := make([]string, 0, len(byStore))
	for ks := range byStore {
		stores = append(stores, ks)
	}
	sort.Strings(stores)
	for _, ks := range stores {
		cache.Forget(ks)
		fmt.Println("Forgot cached password for", ks)
	}
}

func titlesOf(pairs []domain.Mapping) []string {
	titles := make([]string, 0, len(pairs))
	for _, p := range pairs {
		titles = append(titles, p.Name)
	}
	sort.Strings(titles)
	return titles
}

func configFallback(keyStore string, pairs []domain.Mapping) (string, []domain.Mapping) {
	fmt.Println("Configure the default section (Enter to keep the current value).")
	fmt.Println()

	fmt.Println("  # full path to the .kdbx key-store")
	keyStore = term.ReadWithPrefill("  key-store", keyStore)
	fmt.Println()

	fmt.Println("  # secrets mapping: name:env,name:env (name = entry Title in the .kdbx)")
	spec := term.ReadWithPrefill("  secrets", formatSecrets(domain.MappingsToMap(pairs)))
	m := map[string]string{}
	if err := mergeSecretsFlag(m, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return keyStore, domain.MappingsFromMap(m)
}

func formatSecrets(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+m[k])
	}
	return strings.Join(parts, ",")
}
