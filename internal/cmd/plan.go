package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dimkarp93/kdbx-env/internal/config"
	"github.com/dimkarp93/kdbx-env/internal/domain"
)

type envSecret struct {
	env  string
	name string
}

func sortedMappings(secrets map[string]string) []envSecret {
	out := make([]envSecret, 0, len(secrets))
	for name, env := range secrets {
		out = append(out, envSecret{env: env, name: name})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].env != out[j].env {
			return out[i].env < out[j].env
		}
		return out[i].name < out[j].name
	})
	return out
}

func describeSection(cfg config.Config, tool string) string {
	_, hasTool := cfg.Sections[tool]
	_, hasDefault := cfg.Sections["default"]
	switch {
	case tool == "default" && hasDefault:
		return `"default"`
	case hasTool && hasDefault:
		return fmt.Sprintf("%q (merged over \"default\")", tool)
	case hasTool:
		return fmt.Sprintf("%q", tool)
	case hasDefault:
		return fmt.Sprintf("\"default\" (no section for %q)", tool)
	default:
		return fmt.Sprintf("(no section for %q and no \"default\")", tool)
	}
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		safe := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./-:@%+=,", r)
		if !safe {
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}

func renderCommand(mappings []envSecret, child []string) string {
	parts := make([]string, 0, len(mappings)+len(child))
	for _, m := range mappings {
		parts = append(parts, fmt.Sprintf("%s=<secret from %s>", m.env, m.name))
	}
	for _, a := range child {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func printPlan(cfgPath, tool string, cfg config.Config, res domain.Resolved, flags runFlags, child []string) {
	out := os.Stdout
	fmt.Fprintln(out, "Dry run — the command will NOT be executed.")
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Config file:  %s\n", cfgPath)
	if _, err := os.Stat(cfgPath); err != nil {
		fmt.Fprintln(out, "              (not found — using flags only)")
	}

	fmt.Fprintf(out, "Tool:         %s\n", tool)
	fmt.Fprintf(out, "Section:      %s\n", describeSection(cfg, tool))

	keyStore := res.KeyStore
	if keyStore == "" {
		keyStore = "(not configured)"
	}
	fmt.Fprintf(out, "Key-store:    %s\n", keyStore)
	if flags.keyStore != "" {
		fmt.Fprintln(out, "              (from --key-store)")
	}

	mappings := sortedMappings(res.Secrets)
	fmt.Fprintln(out)
	if len(mappings) == 0 {
		fmt.Fprintln(out, "Mappings:     (none configured)")
	} else {
		fmt.Fprintln(out, "Mappings (env ← secret):")
		w := 0
		for _, m := range mappings {
			if len(m.env) > w {
				w = len(m.env)
			}
		}
		for _, m := range mappings {
			fmt.Fprintf(out, "  %-*s ← %s\n", w, m.env, m.name)
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Command:")
	fmt.Fprintf(out, "  %s\n", renderCommand(mappings, child))
}
