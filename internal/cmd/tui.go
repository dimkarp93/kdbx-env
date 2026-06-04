package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"secrets/internal/config"
	"secrets/internal/domain"
)

type tuiMode int

const (
	modeKeyStore tuiMode = iota
	modeList
	modeEdit
	modeCache
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Bold(true)
	focusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	hintStyle  = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type configTUI struct {
	keyStore     textinput.Model
	name         textinput.Model
	env          textinput.Model
	pairs        []domain.Mapping
	cursor       int
	mode         tuiMode
	editIndex    int
	editName     string
	err          string
	canceled     bool
	cacheEnabled bool
	cacheTTL     textinput.Model
	cacheFocus   int
}

func newConfigTUI(keyStore string, pairs []domain.Mapping, cacheConfig *config.CacheConfig) configTUI {
	ks := textinput.New()
	ks.Prompt = ""
	ks.Placeholder = "/path/to/store.kdbx"
	ks.SetValue(keyStore)
	ks.CursorEnd()
	ks.ShowSuggestions = true
	ks.Focus()
	refreshPathSuggestions(&ks)

	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "SECRET_TITLE"
	env := textinput.New()
	env.Prompt = ""
	env.Placeholder = "ENV_VAR"

	ttlVal := "10m"
	cacheEnabled := false
	if cacheConfig != nil && cacheConfig.Enabled {
		cacheEnabled = true
		if cacheConfig.TTL != "" {
			ttlVal = cacheConfig.TTL
		}
	}
	cacheTTL := textinput.New()
	cacheTTL.Prompt = ""
	cacheTTL.Placeholder = "10m"
	cacheTTL.SetValue(ttlVal)

	return configTUI{
		keyStore:     ks,
		name:         name,
		env:          env,
		pairs:        pairs,
		mode:         modeKeyStore,
		editIndex:    -1,
		cacheEnabled: cacheEnabled,
		cacheTTL:     cacheTTL,
	}
}

func refreshPathSuggestions(ti *textinput.Model) {
	val := ti.Value()
	dir := filepath.Dir(val)
	if val == "" {
		dir = "."
	}
	if strings.HasSuffix(val, string(os.PathSeparator)) {
		dir = strings.TrimRight(val, string(os.PathSeparator))
		if dir == "" {
			dir = string(os.PathSeparator)
		}
	}
	entries, err := os.ReadDir(config.ExpandHome(dir))
	if err != nil {
		ti.SetSuggestions(nil)
		return
	}
	suggestions := make([]string, 0, len(entries))
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			full += string(os.PathSeparator)
		}
		suggestions = append(suggestions, full)
	}
	ti.SetSuggestions(suggestions)
}

func (m configTUI) Init() tea.Cmd {
	return textinput.Blink
}

func (m configTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if isKey && keyMsg.String() == "ctrl+c" {
		m.canceled = true
		return m, tea.Quit
	}

	switch m.mode {
	case modeKeyStore:
		return m.updateKeyStore(msg)
	case modeList:
		return m.updateList(msg)
	case modeEdit:
		return m.updateEdit(msg)
	case modeCache:
		return m.updateCache(msg)
	}
	return m, nil
}

func (m configTUI) updateKeyStore(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter", "ctrl+s":
			m.keyStore.Blur()
			m.mode = modeList
			return m, nil
		case "alt+backspace":
			deleteLastPathSegment(&m.keyStore)
			refreshPathSuggestions(&m.keyStore)
			return m, nil
		case "tab":
			if v := m.keyStore.Value(); strings.HasPrefix(v, "~") {
				m.keyStore.SetValue(config.ExpandHome(v))
				m.keyStore.CursorEnd()
				refreshPathSuggestions(&m.keyStore)
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.keyStore, cmd = m.keyStore.Update(msg)
	refreshPathSuggestions(&m.keyStore)
	return m, cmd
}

func deleteLastPathSegment(ti *textinput.Model) {
	v := strings.TrimRight(ti.Value(), string(os.PathSeparator))
	if i := strings.LastIndex(v, string(os.PathSeparator)); i >= 0 {
		v = v[:i+1]
	} else {
		v = ""
	}
	ti.SetValue(v)
	ti.CursorEnd()
}

func (m configTUI) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	m.err = ""
	switch k.String() {
	case "esc":
		m.canceled = true
		return m, tea.Quit
	case "ctrl+s":
		return m, tea.Quit
	case "tab":
		m.mode = modeKeyStore
		return m, m.keyStore.Focus()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.pairs)-1 {
			m.cursor++
		}
	case "a":
		return m.startEdit(-1)
	case "e", "enter":
		if len(m.pairs) > 0 {
			return m.startEdit(m.cursor)
		}
	case "d", "delete":
		if len(m.pairs) > 0 {
			m.pairs = append(m.pairs[:m.cursor], m.pairs[m.cursor+1:]...)
			if m.cursor >= len(m.pairs) && m.cursor > 0 {
				m.cursor--
			}
		}
	case "c":
		m.mode = modeCache
		m.cacheFocus = 0
		m.cacheTTL.Blur()
		return m, nil
	}
	return m, nil
}

func (m configTUI) updateCache(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if ok {
		switch k.String() {
		case "esc":
			m.mode = modeList
			m.cacheTTL.Blur()
			return m, nil
		case "ctrl+s":
			return m, tea.Quit
		case "space", "enter":
			if m.cacheFocus == 0 {
				m.cacheEnabled = !m.cacheEnabled
				if !m.cacheEnabled {
					m.cacheTTL.Blur()
				}
				return m, nil
			}
		case "tab":
			if m.cacheEnabled {
				if m.cacheFocus == 0 {
					m.cacheFocus = 1
					return m, m.cacheTTL.Focus()
				}
				m.cacheFocus = 0
				m.cacheTTL.Blur()
			}
			return m, nil
		}
	}
	if m.cacheFocus == 1 && m.cacheEnabled {
		var cmd tea.Cmd
		m.cacheTTL, cmd = m.cacheTTL.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m configTUI) startEdit(index int) (tea.Model, tea.Cmd) {
	m.mode = modeEdit
	m.editIndex = index
	if index >= 0 {
		m.name.SetValue(m.pairs[index].Name)
		m.env.SetValue(m.pairs[index].Env)
		m.editName = m.pairs[index].Name
	} else {
		m.name.SetValue("")
		m.env.SetValue("")
		m.editName = ""
	}
	m.name.CursorEnd()
	m.env.CursorEnd()
	m.env.Blur()
	return m, m.name.Focus()
}

func (m configTUI) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.mode = modeList
			m.name.Blur()
			m.env.Blur()
			return m, nil
		case "tab", "shift+tab":
			if m.name.Focused() {
				m.name.Blur()
				return m, m.env.Focus()
			}
			m.env.Blur()
			return m, m.name.Focus()
		case "enter":
			return m.commitEdit()
		}
	}
	var cmd tea.Cmd
	if m.name.Focused() {
		m.name, cmd = m.name.Update(msg)
	} else {
		m.env, cmd = m.env.Update(msg)
	}
	return m, cmd
}

func (m configTUI) commitEdit() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.name.Value())
	env := strings.TrimSpace(m.env.Value())
	if name == "" || env == "" {
		m.err = "both secret name and env are required"
		return m, nil
	}
	for i, p := range m.pairs {
		if p.Name == name && i != m.editIndex {
			m.err = fmt.Sprintf("secret %q is already mapped", name)
			return m, nil
		}
	}
	if m.editIndex >= 0 {
		m.pairs[m.editIndex] = domain.Mapping{Name: name, Env: env}
	} else {
		m.pairs = append(m.pairs, domain.Mapping{Name: name, Env: env})
	}
	sort.Slice(m.pairs, func(i, j int) bool { return m.pairs[i].Name < m.pairs[j].Name })
	for i, p := range m.pairs {
		if p.Name == name {
			m.cursor = i
			break
		}
	}
	m.mode = modeList
	m.name.Blur()
	m.env.Blur()
	m.err = ""
	return m, nil
}

func (m configTUI) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Configure secrets — default section"))
	b.WriteString("\n\n")

	ksLabel := "  key-store"
	if m.mode == modeKeyStore {
		ksLabel = focusStyle.Render("▸ key-store")
	}
	b.WriteString(labelStyle.Render(ksLabel) + "\n")
	b.WriteString("    " + m.keyStore.View() + "\n\n")

	listLabel := "  secrets (name → env)"
	if m.mode == modeList {
		listLabel = focusStyle.Render("▸ secrets (name → env)")
	}
	b.WriteString(labelStyle.Render(listLabel) + "\n")
	if len(m.pairs) == 0 {
		b.WriteString(hintStyle.Render("    (none — press 'a' to add)") + "\n")
	} else {
		for i, p := range m.pairs {
			line := fmt.Sprintf("%s → %s", p.Name, p.Env)
			if m.mode == modeList && i == m.cursor {
				b.WriteString(focusStyle.Render("  > "+line) + "\n")
			} else {
				b.WriteString("    " + line + "\n")
			}
		}
	}

	if m.mode == modeEdit {
		b.WriteString("\n")
		verb := "Edit"
		if m.editIndex < 0 {
			verb = "Add"
		}
		b.WriteString(labelStyle.Render("  "+verb+" mapping") + "\n")
		nameMark, envMark := "  ", "  "
		if m.name.Focused() {
			nameMark = focusStyle.Render("▸ ")
		}
		if m.env.Focused() {
			envMark = focusStyle.Render("▸ ")
		}
		b.WriteString(nameMark + "name (Title in .kdbx): " + m.name.View() + "\n")
		b.WriteString(envMark + "env variable:          " + m.env.View() + "\n")
	}

	cacheLabel := "  cache"
	if m.mode == modeCache {
		cacheLabel = focusStyle.Render("▸ cache")
	}
	b.WriteString("\n" + labelStyle.Render(cacheLabel) + "\n")
	toggle := "[ ] keyring disabled"
	if m.cacheEnabled {
		toggle = "[x] keyring enabled"
	}
	if m.mode == modeCache && m.cacheFocus == 0 {
		b.WriteString(focusStyle.Render("  > "+toggle) + "\n")
	} else {
		b.WriteString("    " + toggle + "\n")
	}
	if m.cacheEnabled {
		ttlLine := "TTL: " + m.cacheTTL.View()
		if m.mode == modeCache && m.cacheFocus == 1 {
			b.WriteString(focusStyle.Render("  > ") + ttlLine + "\n")
		} else {
			b.WriteString("    " + ttlLine + "\n")
		}
	}

	if m.err != "" {
		b.WriteString("\n" + errStyle.Render("  ! "+m.err) + "\n")
	}

	b.WriteString("\n" + hintStyle.Render("  "+m.legend()) + "\n")
	return b.String()
}

func (m configTUI) legend() string {
	switch m.mode {
	case modeKeyStore:
		return "tab complete (~ expands) · alt+⌫ delete dir · ↑/↓ suggestions · enter next · esc cancel"
	case modeList:
		return "↑/↓ move · a add · e edit · d delete · tab edit path · c cache · ctrl+s save · esc cancel"
	case modeEdit:
		return "tab switch field · enter ok · esc back"
	case modeCache:
		return "space toggle · tab TTL · ctrl+s save · esc back"
	}
	return ""
}

func runConfigTUI(keyStore string, pairs []domain.Mapping, cacheConfig *config.CacheConfig) (string, []domain.Mapping, *config.CacheConfig, bool, error) {
	res, err := tea.NewProgram(newConfigTUI(keyStore, pairs, cacheConfig)).Run()
	if err != nil {
		return "", nil, nil, false, err
	}
	final := res.(configTUI)
	if final.canceled {
		return "", nil, nil, true, nil
	}
	var outCache *config.CacheConfig
	if final.cacheEnabled {
		ttl := strings.TrimSpace(final.cacheTTL.Value())
		if ttl == "" {
			ttl = "10m"
		}
		outCache = &config.CacheConfig{Enabled: true, TTL: ttl}
	}
	return strings.TrimSpace(final.keyStore.Value()), final.pairs, outCache, false, nil
}
