package cmd

import (
	"fmt"
	"strings"
)

type runFlags struct {
	configPath string
	keyStore   string
	secrets    map[string]string
	dryRun     bool
}

func splitArgs(args []string) (left, child []string, hasSep bool) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:], true
		}
	}
	return args, nil, false
}

func parseRunFlags(args []string) (runFlags, error) {
	f := runFlags{secrets: map[string]string{}}
	needValue := func(i int) (string, error) {
		if i+1 >= len(args) {
			return "", fmt.Errorf("%s requires a value", args[i])
		}
		return args[i+1], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.configPath = v
			i++
		case strings.HasPrefix(a, "--config="):
			f.configPath = strings.TrimPrefix(a, "--config=")
		case a == "--key-store":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.keyStore = v
			i++
		case strings.HasPrefix(a, "--key-store="):
			f.keyStore = strings.TrimPrefix(a, "--key-store=")
		case a == "--secrets":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			if err := mergeSecretsFlag(f.secrets, v); err != nil {
				return f, err
			}
			i++
		case strings.HasPrefix(a, "--secrets="):
			if err := mergeSecretsFlag(f.secrets, strings.TrimPrefix(a, "--secrets=")); err != nil {
				return f, err
			}
		case a == "--dry-run":
			f.dryRun = true
		default:
			return f, fmt.Errorf("unknown flag: %s", a)
		}
	}
	return f, nil
}

func mergeSecretsFlag(dst map[string]string, spec string) error {
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.LastIndex(part, ":")
		if i < 0 {
			return fmt.Errorf("invalid secrets entry %q (expected name:env)", part)
		}
		name := strings.TrimSpace(part[:i])
		env := strings.TrimSpace(part[i+1:])
		if name == "" || env == "" {
			return fmt.Errorf("invalid secrets entry %q (expected name:env)", part)
		}
		dst[name] = env
	}
	return nil
}
