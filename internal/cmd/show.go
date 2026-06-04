package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"secrets/internal/config"
	"secrets/internal/domain"
	"secrets/internal/keyring"
	"secrets/internal/term"
)

var openInKeePassXC = func(path string) error {
	bin, err := exec.LookPath("keepassxc")
	if err != nil {
		return fmt.Errorf("keepassxc GUI not found in PATH")
	}
	cmd := exec.Command(bin, path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func cmdShow(path string) {
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load config", path+":", err)
		os.Exit(1)
	}
	views := domain.BuildStoreViews(cfg)
	cache := keyring.New(cfg.Cache)

	if !term.IsInteractive() {
		printStoreViews(path, cfg.Cache, views, cache)
		return
	}
	if _, err := tea.NewProgram(newShowTUI(path, cfg.Cache, views, cache)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}

func printStoreViews(configPath string, cacheConfig *config.CacheConfig, views []domain.StoreView, cache keyring.Cache) {
	fmt.Println("Config:", configPath)
	fmt.Println()
	if cacheConfig == nil || !cacheConfig.Enabled {
		fmt.Println("Caching: disabled")
	} else {
		ttl := cacheConfig.TTL
		if ttl == "" {
			ttl = "10m"
		}
		fmt.Printf("Caching: enabled (TTL: %s)\n", ttl)
	}
	fmt.Println()
	if len(views) == 0 {
		fmt.Println("No key-stores configured.")
		return
	}
	fmt.Println("Key-stores:")
	for _, v := range views {
		suffix := ""
		if !v.Exists {
			suffix = "  (missing)"
		}
		fmt.Printf("  %s%s\n", v.Path, suffix)
		for _, mp := range v.Mappings {
			fmt.Printf("      %s → %s\n", mp.Name, mp.Env)
		}
		if cacheConfig != nil && cacheConfig.Enabled {
			st := cache.Status(v.Path)
			if st.Cached {
				fmt.Printf("      [cached, expires in %s]\n", formatDuration(st.ExpiresIn))
			} else {
				fmt.Printf("      [not cached]\n")
			}
		}
	}
}

func formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

type showTUI struct {
	configPath  string
	cacheConfig *config.CacheConfig
	views       []domain.StoreView
	cache       keyring.Cache
	cursor      int
	status      string
}

func newShowTUI(configPath string, cacheConfig *config.CacheConfig, views []domain.StoreView, cache keyring.Cache) showTUI {
	return showTUI{configPath: configPath, cacheConfig: cacheConfig, views: views, cache: cache}
}

func (m showTUI) Init() tea.Cmd { return nil }

func (m showTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.status = ""
		}
	case "down", "j":
		if m.cursor < len(m.views)-1 {
			m.cursor++
			m.status = ""
		}
	case "enter":
		if len(m.views) == 0 {
			return m, nil
		}
		v := m.views[m.cursor]
		if !v.Exists {
			m.status = "✗ file does not exist: " + v.Path
			return m, nil
		}
		if err := openInKeePassXC(v.Path); err != nil {
			m.status = "✗ " + err.Error()
		} else {
			m.status = "→ opened in KeePassXC: " + v.Path
		}
	}
	return m, nil
}

func (m showTUI) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("secrets — config view"))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("  config: ") + m.configPath + "\n")

	if m.cacheConfig == nil || !m.cacheConfig.Enabled {
		b.WriteString(hintStyle.Render("  caching: disabled") + "\n")
	} else {
		ttl := m.cacheConfig.TTL
		if ttl == "" {
			ttl = "10m"
		}
		b.WriteString(hintStyle.Render(fmt.Sprintf("  caching: enabled (TTL: %s)", ttl)) + "\n")
	}
	b.WriteString("\n")

	if len(m.views) == 0 {
		b.WriteString(hintStyle.Render("  No key-stores configured.") + "\n")
	} else {
		b.WriteString(labelStyle.Render("  key-stores") + "\n")
		for i, v := range m.views {
			missing := ""
			if !v.Exists {
				missing = hintStyle.Render("  (missing)")
			}
			if i == m.cursor {
				b.WriteString(focusStyle.Render("  > "+v.Path) + missing + "\n")
			} else {
				b.WriteString("    " + v.Path + missing + "\n")
			}
			for _, mp := range v.Mappings {
				b.WriteString(hintStyle.Render(fmt.Sprintf("        %s → %s", mp.Name, mp.Env)) + "\n")
			}
			if m.cacheConfig != nil && m.cacheConfig.Enabled {
				st := m.cache.Status(v.Path)
				if st.Cached {
					b.WriteString(hintStyle.Render(fmt.Sprintf("        [cached, expires in %s]", formatDuration(st.ExpiresIn))) + "\n")
				} else {
					b.WriteString(hintStyle.Render("        [not cached]") + "\n")
				}
			}
		}
	}

	if m.status != "" {
		b.WriteString("\n  " + m.status + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("  ↑/↓ select key-store · enter open in KeePassXC · q quit") + "\n")
	return b.String()
}
