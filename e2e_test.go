//go:build e2e

package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"secrets/internal/config"
	"secrets/internal/keepass"
)

const testPassword = "test-pass-123"

var binaryPath string

func TestMain(m *testing.M) {
	root := os.Getenv("SECRETS_E2E_ROOT")
	if root == "" {
		fmt.Fprintln(os.Stderr, "SECRETS_E2E_ROOT is required for e2e tests")
		os.Exit(2)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "cannot create SECRETS_E2E_ROOT:", err)
		os.Exit(2)
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "getwd:", err)
		os.Exit(2)
	}
	binaryPath = filepath.Join(wd, "secrets")
	if _, err := os.Stat(binaryPath); err != nil {
		fmt.Fprintln(os.Stderr, "binary not found at", binaryPath, "— run `just build` first")
		os.Exit(2)
	}
	if _, err := exec.LookPath("keepassxc-cli"); err != nil {
		fmt.Fprintln(os.Stderr, "keepassxc-cli is required for e2e tests")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

type sandbox struct {
	t    *testing.T
	dir  string
	home string
}

func newSandbox(t *testing.T) *sandbox {
	t.Helper()
	root := os.Getenv("SECRETS_E2E_ROOT")
	keep := os.Getenv("SECRETS_E2E_KEEP") == "1"

	dir := filepath.Join(root, t.Name())
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(home, ".config", "secrets"), 0755); err != nil {
		t.Fatal(err)
	}
	sb := &sandbox{t: t, dir: dir, home: home}
	if !keep {
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
	}
	return sb
}

func xmlEsc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func buildTestXML(entries map[string]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString("<KeePassFile><Root><Group><Name>Root</Name>\n")
	for title, pass := range entries {
		b.WriteString("<Entry>\n")
		b.WriteString("  <String><Key>Title</Key><Value>" + xmlEsc(title) + "</Value></String>\n")
		b.WriteString("  <String><Key>Password</Key><Value>" + xmlEsc(pass) + "</Value></String>\n")
		b.WriteString("</Entry>\n")
	}
	b.WriteString("</Group></Root></KeePassFile>\n")
	return b.String()
}

func (s *sandbox) makeStore(rel string, entries map[string]string) string {
	s.t.Helper()
	dbPath := filepath.Join(s.home, rel)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		s.t.Fatal(err)
	}
	xmlPath := filepath.Join(s.dir, strings.ReplaceAll(rel, "/", "_")+".import.xml")
	if err := os.WriteFile(xmlPath, []byte(buildTestXML(entries)), 0600); err != nil {
		s.t.Fatal(err)
	}
	out, err := keepass.Run(testPassword+"\n"+testPassword+"\n", "import", "-q", "-p", xmlPath, dbPath)
	if err != nil {
		s.t.Fatalf("keepassxc-cli import failed: %v\n%s", err, out)
	}
	return dbPath
}

func (s *sandbox) writeConfig(sections map[string]config.Section) {
	s.t.Helper()
	data, _ := json.MarshalIndent(config.Config{Sections: sections}, "", "  ")
	path := filepath.Join(s.home, ".config", "secrets", "default")
	if err := os.WriteFile(path, data, 0600); err != nil {
		s.t.Fatal(err)
	}
}

type cmdResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func (s *sandbox) run(args ...string) cmdResult {
	s.t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(),
		"HOME="+s.home,
		"SECRETS_PASSWORD="+testPassword,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			s.t.Fatalf("run %v: %v", args, err)
		}
	}
	return cmdResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: code}
}

func TestE2E_InjectSecret(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{"GITHUB_TOKEN": "ghp_secret"})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{"GITHUB_TOKEN": "GH_TOKEN"}},
	})

	r := sb.run("--", "sh", "-c", `printf %s "$GH_TOKEN"`)
	if r.exitCode != 0 {
		t.Fatalf("exit %d\nstderr:\n%s", r.exitCode, r.stderr)
	}
	if r.stdout != "ghp_secret" {
		t.Errorf("stdout: got %q, want %q", r.stdout, "ghp_secret")
	}
}

func TestE2E_MergeDefaultAndToolSection(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{
		"GITHUB_TOKEN": "ghp_secret",
		"API_KEY":      "api_secret",
	})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{"GITHUB_TOKEN": "GH_TOKEN"}},
		"sh":      {Secrets: map[string]string{"API_KEY": "API_KEY"}},
	})

	r := sb.run("--", "sh", "-c", `printf "%s|%s" "$GH_TOKEN" "$API_KEY"`)
	if r.exitCode != 0 {
		t.Fatalf("exit %d\nstderr:\n%s", r.exitCode, r.stderr)
	}
	if r.stdout != "ghp_secret|api_secret" {
		t.Errorf("stdout: got %q, want %q", r.stdout, "ghp_secret|api_secret")
	}
}

func TestE2E_FlagsOverrideConfig(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("flag.kdbx", map[string]string{"TOKEN": "flag_secret"})

	r := sb.run("--key-store", store, "--secrets=TOKEN:MY_TOKEN", "--", "sh", "-c", `printf %s "$MY_TOKEN"`)
	if r.exitCode != 0 {
		t.Fatalf("exit %d\nstderr:\n%s", r.exitCode, r.stderr)
	}
	if r.stdout != "flag_secret" {
		t.Errorf("stdout: got %q, want %q", r.stdout, "flag_secret")
	}
}

func TestE2E_MissingSecret(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{"GITHUB_TOKEN": "ghp_secret"})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{"NOPE": "NOPE_ENV"}},
	})

	r := sb.run("--", "sh", "-c", "echo should-not-run")
	if r.exitCode == 0 {
		t.Fatalf("expected non-zero exit\nstdout:\n%s", r.stdout)
	}
	if !strings.Contains(r.stderr, "not found") {
		t.Errorf("expected 'not found' in stderr:\n%s", r.stderr)
	}
	if strings.Contains(r.stdout, "should-not-run") {
		t.Errorf("child command ran despite missing secret:\n%s", r.stdout)
	}
}

func TestE2E_ExitCodePropagation(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{"TOKEN": "secret"})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{"TOKEN": "TOKEN"}},
	})

	r := sb.run("--", "sh", "-c", "exit 7")
	if r.exitCode != 7 {
		t.Errorf("exit code: got %d, want 7\nstderr:\n%s", r.exitCode, r.stderr)
	}
}

func (s *sandbox) runNoPassword(args ...string) cmdResult {
	s.t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(), "HOME="+s.home)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	}
	return cmdResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: code}
}

func TestE2E_DryRun(t *testing.T) {
	sb := newSandbox(t)
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: "~/missing-store.kdbx", Secrets: map[string]string{"GITHUB_TOKEN": "GH_TOKEN"}},
		"sh":      {Secrets: map[string]string{"API_KEY": "API_KEY"}},
	})

	r := sb.runNoPassword("--dry-run", "--", "sh", "-c", "echo MARKER_EXECUTED")
	if r.exitCode != 0 {
		t.Fatalf("dry-run exit %d\nstderr:\n%s", r.exitCode, r.stderr)
	}
	if n := strings.Count(r.stdout, "MARKER_EXECUTED"); n != 1 {
		t.Errorf("marker appears %d times (want 1 — only in the plan); command was executed:\n%s", n, r.stdout)
	}
	for _, want := range []string{
		"Dry run",
		`"sh" (merged over "default")`,
		"← GITHUB_TOKEN",
		"← API_KEY",
		"GH_TOKEN=<secret from GITHUB_TOKEN>",
		"API_KEY=<secret from API_KEY>",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, r.stdout)
		}
	}
}

func (s *sandbox) storeTitles(dbPath string) map[string]bool {
	s.t.Helper()
	out, err := keepass.Run(testPassword+"\n", "export", "-q", "-f", "xml", dbPath)
	if err != nil {
		s.t.Fatalf("export %s failed: %v\n%s", dbPath, err, out)
	}
	entries, err := keepass.ParseSecrets(out)
	if err != nil {
		s.t.Fatal(err)
	}
	titles := map[string]bool{}
	for _, e := range entries {
		titles[e.Title] = true
	}
	return titles
}

func TestE2E_CheckAddsMissingSecrets(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{"GITHUB_TOKEN": "ghp_secret"})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{
			"GITHUB_TOKEN": "GH_TOKEN",
			"NPM_TOKEN":    "NPM_TOKEN",
		}},
	})

	r := sb.run("check", "-y")
	if r.exitCode != 0 {
		t.Fatalf("check exit %d\nstderr:\n%s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "NPM_TOKEN") {
		t.Errorf("check did not report missing NPM_TOKEN:\n%s", r.stdout)
	}

	titles := sb.storeTitles(store)
	if !titles["NPM_TOKEN"] || !titles["GITHUB_TOKEN"] {
		t.Errorf("NPM_TOKEN not added to store, titles=%v", titles)
	}
}

func TestE2E_CheckAllPresent(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{"TOKEN": "v"})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{"TOKEN": "TOKEN"}},
	})

	r := sb.run("check", "-y")
	if r.exitCode != 0 {
		t.Fatalf("check exit %d\nstderr:\n%s", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "All secrets are present") {
		t.Errorf("expected all-present message:\n%s", r.stdout)
	}
}

func TestE2E_ConfigCreatesStoreAndSecrets(t *testing.T) {
	sb := newSandbox(t)
	store := filepath.Join(sb.home, "new", "store.kdbx")
	cfgPath := filepath.Join(sb.home, ".config", "secrets", "default")

	stdin := store + "\n" + "GITHUB_TOKEN:GH_TOKEN,API_KEY:API_KEY\n"
	r := sb.runStdin(stdin, "config", "-y", "--config", cfgPath)
	if r.exitCode != 0 {
		t.Fatalf("config exit %d\nstdout:\n%s\nstderr:\n%s", r.exitCode, r.stdout, r.stderr)
	}

	if _, err := os.Stat(store); err != nil {
		t.Fatalf("store not created: %v", err)
	}
	titles := sb.storeTitles(store)
	if !titles["GITHUB_TOKEN"] || !titles["API_KEY"] {
		t.Errorf("created store missing secrets, titles=%v", titles)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Sections["default"].KeyStore != store {
		t.Errorf("config key-store: got %q, want %q", cfg.Sections["default"].KeyStore, store)
	}
}

func (s *sandbox) runStdin(stdin string, args ...string) cmdResult {
	s.t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = s.dir
	cmd.Env = append(os.Environ(), "HOME="+s.home, "SECRETS_PASSWORD="+testPassword)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	}
	return cmdResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: code}
}

func TestE2E_WrongPassword(t *testing.T) {
	sb := newSandbox(t)
	store := sb.makeStore("store.kdbx", map[string]string{"TOKEN": "secret"})
	sb.writeConfig(map[string]config.Section{
		"default": {KeyStore: store, Secrets: map[string]string{"TOKEN": "TOKEN"}},
	})

	cmd := exec.Command(binaryPath, "--", "sh", "-c", "echo nope")
	cmd.Dir = sb.dir
	cmd.Env = append(os.Environ(), "HOME="+sb.home, "SECRETS_PASSWORD=wrong-password")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if stdout.String() == "nope\n" {
		t.Errorf("child ran with wrong password")
	}
	if !strings.Contains(stderr.String(), "keepassxc-cli") {
		t.Errorf("expected keepassxc-cli error in stderr:\n%s", stderr.String())
	}
}
