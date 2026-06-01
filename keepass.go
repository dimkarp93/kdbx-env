package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type KPFile struct {
	XMLName xml.Name `xml:"KeePassFile"`
	Root    KPRoot   `xml:"Root"`
}

type KPRoot struct {
	Group KPGroup `xml:"Group"`
}

type KPGroup struct {
	Name    string    `xml:"Name"`
	Groups  []KPGroup `xml:"Group"`
	Entries []KPEntry `xml:"Entry"`
}

type KPEntry struct {
	Strings []KPString `xml:"String"`
}

type KPString struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type kpEntry struct {
	path  string
	title string
	value string
}

func checkEngine() {
	if _, err := exec.LookPath("keepassxc-cli"); err != nil {
		fmt.Fprintln(os.Stderr, "keepassxc-cli not found in PATH.")
		fmt.Fprintln(os.Stderr, "Install it:")
		fmt.Fprintln(os.Stderr, "  Debian/Ubuntu:  sudo apt install keepassxc")
		fmt.Fprintln(os.Stderr, "  Arch Linux:     sudo pacman -S keepassxc")
		fmt.Fprintln(os.Stderr, "  macOS:          brew install keepassxc")
		os.Exit(1)
	}
}

func runKP(stdinData string, args ...string) (string, error) {
	cmd := exec.Command("keepassxc-cli", args...)
	cmd.Stdin = strings.NewReader(stdinData)
	out, err := cmd.Output()
	return string(out), err
}

func parseSecrets(xmlData string) ([]kpEntry, error) {
	var kpf KPFile
	if err := xml.Unmarshal([]byte(xmlData), &kpf); err != nil {
		return nil, err
	}

	var out []kpEntry
	var walk func(g KPGroup, prefix []string, depth int)
	walk = func(g KPGroup, prefix []string, depth int) {
		cur := prefix
		if depth > 0 {
			cur = append(append([]string{}, prefix...), g.Name)
		}
		for _, e := range g.Entries {
			var title, password string
			for _, s := range e.Strings {
				switch s.Key {
				case "Title":
					title = s.Value
				case "Password":
					password = s.Value
				}
			}
			if title == "" {
				continue
			}
			path := title
			if len(cur) > 0 {
				path = strings.Join(append(append([]string{}, cur...), title), "/")
			}
			out = append(out, kpEntry{path: path, title: title, value: password})
		}
		for _, child := range g.Groups {
			walk(child, cur, depth+1)
		}
	}
	walk(kpf.Root.Group, nil, 0)
	return out, nil
}

func lookupSecret(entries []kpEntry, name string) (string, error) {
	byPath := strings.Contains(name, "/")
	var matches []kpEntry
	for _, e := range entries {
		if (byPath && e.path == name) || (!byPath && e.title == name) {
			matches = append(matches, e)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("secret %q not found in key-store", name)
	case 1:
		return matches[0].value, nil
	default:
		var paths []string
		for _, m := range matches {
			paths = append(paths, m.path)
		}
		return "", fmt.Errorf("secret %q is ambiguous, matches: %s (use full Group/Title path)", name, strings.Join(paths, ", "))
	}
}
