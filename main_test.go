package main

import (
	"reflect"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	cases := []struct {
		name      string
		in        []string
		wantLeft  []string
		wantChild []string
		wantSep   bool
	}{
		{"no sep", []string{"--config", "x"}, []string{"--config", "x"}, nil, false},
		{"sep at end", []string{"--config", "x", "--"}, []string{"--config", "x"}, []string{}, true},
		{"basic", []string{"--config", "x", "--", "install", "a/b"}, []string{"--config", "x"}, []string{"install", "a/b"}, true},
		{"sep first", []string{"--", "tool"}, []string{}, []string{"tool"}, true},
		{"child has flags", []string{"--", "tool", "-v", "--x"}, []string{}, []string{"tool", "-v", "--x"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			left, child, sep := splitArgs(c.in)
			if !reflect.DeepEqual(left, c.wantLeft) {
				t.Errorf("left: got %v, want %v", left, c.wantLeft)
			}
			if !reflect.DeepEqual(child, c.wantChild) {
				t.Errorf("child: got %v, want %v", child, c.wantChild)
			}
			if sep != c.wantSep {
				t.Errorf("sep: got %v, want %v", sep, c.wantSep)
			}
		})
	}
}

func TestParseRunFlags(t *testing.T) {
	f, err := parseRunFlags([]string{"--config", "/c", "--key-store=/k", "--secrets", "A:AA,B:BB", "--secrets=C:CC"})
	if err != nil {
		t.Fatal(err)
	}
	if f.configPath != "/c" {
		t.Errorf("configPath: got %q", f.configPath)
	}
	if f.keyStore != "/k" {
		t.Errorf("keyStore: got %q", f.keyStore)
	}
	want := map[string]string{"A": "AA", "B": "BB", "C": "CC"}
	if !reflect.DeepEqual(f.secrets, want) {
		t.Errorf("secrets: got %v, want %v", f.secrets, want)
	}
}

func TestParseRunFlagsErrors(t *testing.T) {
	cases := [][]string{
		{"--config"},
		{"--secrets", "noColon"},
		{"--secrets", ":env"},
		{"--secrets", "name:"},
		{"--unknown"},
	}
	for _, c := range cases {
		if _, err := parseRunFlags(c); err == nil {
			t.Errorf("expected error for %v", c)
		}
	}
}

func TestMergeSecretsFlag(t *testing.T) {
	m := map[string]string{}
	if err := mergeSecretsFlag(m, "Group/Sub/Title:ENV, X:Y ,"); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Group/Sub/Title": "ENV", "X": "Y"}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %v, want %v", m, want)
	}
}

func TestResolveMerge(t *testing.T) {
	cfg := Config{
		"default": {KeyStore: "~/store.kdbx", Secrets: map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "SHARED": "SHARED_ENV"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM_TOKEN", "SHARED": "OVERRIDE"}},
	}

	r := resolve(cfg, "install", runFlags{secrets: map[string]string{}})
	if r.keyStore != "~/store.kdbx" {
		t.Errorf("keyStore: got %q", r.keyStore)
	}
	want := map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "SHARED": "OVERRIDE", "NPM_TOKEN": "NPM_TOKEN"}
	if !reflect.DeepEqual(r.secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.secrets, want)
	}
}

func TestResolveFlagsOverride(t *testing.T) {
	cfg := Config{
		"default": {KeyStore: "/from-cfg", Secrets: map[string]string{"A": "A1"}},
	}
	flags := runFlags{keyStore: "/from-flag", secrets: map[string]string{"A": "A2", "B": "B1"}}
	r := resolve(cfg, "unknown-tool", flags)
	if r.keyStore != "/from-flag" {
		t.Errorf("keyStore: got %q", r.keyStore)
	}
	want := map[string]string{"A": "A2", "B": "B1"}
	if !reflect.DeepEqual(r.secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.secrets, want)
	}
}

const sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile><Root><Group><Name>Root</Name>
  <Entry><String><Key>Title</Key><Value>GITHUB_TOKEN</Value></String><String><Key>Password</Key><Value>ghp_top</Value></String></Entry>
  <Group><Name>web</Name>
    <Entry><String><Key>Title</Key><Value>API_KEY</Value></String><String><Key>Password</Key><Value>web_secret</Value></String></Entry>
    <Entry><String><Key>Title</Key><Value>DUP</Value></String><String><Key>Password</Key><Value>web_dup</Value></String></Entry>
  </Group>
  <Group><Name>db</Name>
    <Entry><String><Key>Title</Key><Value>DUP</Value></String><String><Key>Password</Key><Value>db_dup</Value></String></Entry>
  </Group>
</Group></Root></KeePassFile>`

func TestParseSecrets(t *testing.T) {
	entries, err := parseSecrets(sampleXML)
	if err != nil {
		t.Fatal(err)
	}

	v, err := lookupSecret(entries, "GITHUB_TOKEN")
	if err != nil || v != "ghp_top" {
		t.Errorf("GITHUB_TOKEN: got %q, err %v", v, err)
	}

	v, err = lookupSecret(entries, "API_KEY")
	if err != nil || v != "web_secret" {
		t.Errorf("API_KEY: got %q, err %v", v, err)
	}

	v, err = lookupSecret(entries, "web/API_KEY")
	if err != nil || v != "web_secret" {
		t.Errorf("web/API_KEY by path: got %q, err %v", v, err)
	}

	if _, err := lookupSecret(entries, "DUP"); err == nil {
		t.Error("expected ambiguity error for DUP")
	}

	v, err = lookupSecret(entries, "db/DUP")
	if err != nil || v != "db_dup" {
		t.Errorf("db/DUP by path: got %q, err %v", v, err)
	}

	if _, err := lookupSecret(entries, "MISSING"); err == nil {
		t.Error("expected not-found error for MISSING")
	}
}
