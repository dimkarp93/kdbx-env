package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func cmdConfig(path string) {
	if path == "" {
		path = defaultConfigPath()
	}

	cfg := Config{}
	if c, err := loadConfig(path); err == nil {
		cfg = c
	}
	def := cfg["default"]
	if def.Secrets == nil {
		def.Secrets = map[string]string{}
	}

	fmt.Println("Configure the default section (Enter to keep the current value).")
	fmt.Println()

	fmt.Println("  # full path to the .kdbx key-store")
	def.KeyStore = readWithPrefill("  key-store", def.KeyStore)
	fmt.Println()

	fmt.Println("  # secrets mapping: name:env,name:env (name = entry Title in the .kdbx)")
	mapped, err := promptSecrets(formatSecrets(def.Secrets))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	def.Secrets = mapped

	cfg["default"] = def
	if err := saveConfig(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "cannot write config:", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("Config saved to", path)
}

func promptSecrets(current string) (map[string]string, error) {
	spec := readWithPrefill("  secrets", current)
	m := map[string]string{}
	if err := mergeSecretsFlag(m, spec); err != nil {
		return nil, err
	}
	return m, nil
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
