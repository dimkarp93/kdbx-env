package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"secrets/internal/config"
	"secrets/internal/domain"
	"secrets/internal/keepass"
	"secrets/internal/keyring"
)

func cmdRun(flags runFlags, child []string) int {
	if len(child) == 0 {
		fmt.Fprintln(os.Stderr, "no command specified after --")
		return 2
	}
	if !flags.dryRun {
		keepass.CheckEngine()
	}

	cfgPath := flags.configPath
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}
	cfg := config.Config{}
	if c, err := config.Load(cfgPath); err == nil {
		cfg = c
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	tool := filepath.Base(child[0])
	res := domain.Resolve(cfg, tool, flags.keyStore, flags.secrets)

	if flags.dryRun {
		printPlan(cfgPath, tool, cfg, res, flags, child)
		return 0
	}

	if res.KeyStore == "" {
		fmt.Fprintln(os.Stderr, "no key-store configured: set it in", cfgPath, "or pass --key-store")
		return 1
	}
	if len(res.Secrets) == 0 {
		fmt.Fprintf(os.Stderr, "no secrets configured for %q in %s (or via --secrets)\n", tool, cfgPath)
		return 1
	}

	keyStore := config.ExpandHome(res.KeyStore)
	if _, err := os.Stat(keyStore); err != nil {
		fmt.Fprintln(os.Stderr, "key-store not found:", keyStore)
		return 1
	}

	prompt := fmt.Sprintf("Enter password for %s to run %s: ", keyStore, tool)
	out, _, err := domain.UnlockExport(keyStore, keyring.New(cfg.Cache), prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	entries, err := keepass.ParseSecrets(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot parse key-store:", err)
		return 1
	}

	names := make([]string, 0, len(res.Secrets))
	for n := range res.Secrets {
		names = append(names, n)
	}
	sort.Strings(names)

	env := os.Environ()
	var failures []string
	for _, name := range names {
		val, err := keepass.LookupSecret(entries, name)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		env = append(env, res.Secrets[name]+"="+val)
	}
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "cannot resolve secrets:")
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, "  "+f)
		}
		return 1
	}

	bin, err := exec.LookPath(child[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "command not found:", child[0])
		return 127
	}
	cmd := exec.Command(bin, child[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "command failed:", err)
		return 1
	}
	return 0
}
