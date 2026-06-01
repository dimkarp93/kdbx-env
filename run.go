package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func cmdRun(flags runFlags, child []string) int {
	if len(child) == 0 {
		fmt.Fprintln(os.Stderr, "no command specified after --")
		return 2
	}
	checkEngine()

	cfgPath := flags.configPath
	if cfgPath == "" {
		cfgPath = defaultConfigPath()
	}
	cfg := Config{}
	if c, err := loadConfig(cfgPath); err == nil {
		cfg = c
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	tool := filepath.Base(child[0])
	res := resolve(cfg, tool, flags)

	if res.keyStore == "" {
		fmt.Fprintln(os.Stderr, "no key-store configured: set it in", cfgPath, "or pass --key-store")
		return 1
	}
	if len(res.secrets) == 0 {
		fmt.Fprintf(os.Stderr, "no secrets configured for %q in %s (or via --secrets)\n", tool, cfgPath)
		return 1
	}

	keyStore := expandHome(res.keyStore)
	if _, err := os.Stat(keyStore); err != nil {
		fmt.Fprintln(os.Stderr, "key-store not found:", keyStore)
		return 1
	}

	pass := readPassword(fmt.Sprintf("Enter password for %s to run %s: ", keyStore, tool))
	if pass == "" {
		fmt.Fprintln(os.Stderr, "password cannot be empty")
		return 1
	}

	out, err := runKP(pass+"\n", "export", "-q", "-f", "xml", keyStore)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keepassxc-cli export error:", err)
		if msg := strings.TrimSpace(out); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
		return 1
	}

	entries, err := parseSecrets(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot parse key-store:", err)
		return 1
	}

	names := make([]string, 0, len(res.secrets))
	for n := range res.secrets {
		names = append(names, n)
	}
	sort.Strings(names)

	env := os.Environ()
	var failures []string
	for _, name := range names {
		val, err := lookupSecret(entries, name)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		env = append(env, res.secrets[name]+"="+val)
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
