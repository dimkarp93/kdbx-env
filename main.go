package main

import (
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "version":
		fmt.Println(version)
		return
	case "help", "--help", "-h":
		usage()
		return
	case "config":
		cmdConfig(configFlagOnly(args[1:]))
		return
	}

	left, child, hasSep := splitArgs(args)

	for _, a := range left {
		if a == "--version" || a == "-v" {
			fmt.Println(version)
			return
		}
	}

	if !hasSep {
		fmt.Fprintln(os.Stderr, "missing -- separator before the command to run")
		usage()
		os.Exit(2)
	}

	flags, err := parseRunFlags(left)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	os.Exit(cmdRun(flags, child))
}

func configFlagOnly(args []string) string {
	path := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--config="):
			path = strings.TrimPrefix(a, "--config=")
		default:
			fmt.Fprintln(os.Stderr, "unknown argument for config:", a)
			os.Exit(2)
		}
	}
	return path
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  secrets [--config <path>] [--key-store <path>] [--secrets=name:env,...] -- <cmd> [args...]
  secrets config [--config <path>]
  secrets version | --version | -v

Runs <cmd> with secrets from a .kdbx key-store injected as environment variables.
Secrets never touch your shell history or disk.

Flags:
  --config <path>          Config file (default: ~/.config/secrets/default)
  --key-store <path>       Path to the .kdbx file (overrides config)
  --secrets=name:env,...   Secret-to-env mapping (merged over config)

The first word of <cmd> selects the config section (falling back to "default").
Requires keepassxc-cli in PATH.
`)
}
