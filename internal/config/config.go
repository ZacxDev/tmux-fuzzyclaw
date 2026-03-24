package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration.
type Config struct {
	StateDir         string          `yaml:"state_dir"`
	ActivityDir      string          `yaml:"activity_dir"`
	ClaudeProjectDir string          `yaml:"claude_projects_dir"`
	Dashboard        DashboardConfig `yaml:"dashboard"`
	Theme            ThemeConfig     `yaml:"theme"`
	Keys             KeyConfig       `yaml:"keys"`
	Tokens           TokenConfig     `yaml:"tokens"`
}

// DashboardConfig controls the dashboard TUI.
type DashboardConfig struct {
	SortBy          string        `yaml:"sort_by"`
	SortAscending   bool          `yaml:"sort_ascending"`
	ShowSections    bool          `yaml:"show_sections"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	Columns         []string      `yaml:"columns"`
	Preview         PreviewConfig `yaml:"preview"`
}

// PreviewConfig controls the right preview panel.
type PreviewConfig struct {
	Position        string `yaml:"position"`
	Width           int    `yaml:"width"`
	SyntaxHighlight bool   `yaml:"syntax_highlight"`
}

// TokenConfig holds cost estimation parameters.
type TokenConfig struct {
	InputCost  float64 `yaml:"input_cost"`
	OutputCost float64 `yaml:"output_cost"`
}

// Default returns a Config with sensible defaults matching the existing bash behavior.
func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		StateDir:         filepath.Join(home, ".tmux", "tasks"),
		ActivityDir:      filepath.Join(home, ".tmux", "activity"),
		ClaudeProjectDir: filepath.Join(home, ".claude", "projects"),
		Dashboard: DashboardConfig{
			SortBy:          "idle",
			SortAscending:   true,
			ShowSections:    true,
			RefreshInterval: 2 * time.Second,
			Columns:         []string{"status", "name", "dir", "idle", "summary"},
			Preview: PreviewConfig{
				Position:        "right",
				Width:           45,
				SyntaxHighlight: true,
			},
		},
		Theme: DefaultTheme(),
		Keys:  DefaultKeys(),
		Tokens: TokenConfig{
			InputCost:  3.00,
			OutputCost: 15.00,
		},
	}
}

// Load reads a config file and merges it over defaults.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		path = defaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.expandPaths()
	return cfg, nil
}

func defaultConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "fuzzyclaw", "config.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "fuzzyclaw", "config.yml")
}

func (c *Config) expandPaths() {
	home, _ := os.UserHomeDir()
	expand := func(p string) string {
		if len(p) > 0 && p[0] == '~' {
			return filepath.Join(home, p[1:])
		}
		return p
	}
	c.StateDir = expand(c.StateDir)
	c.ActivityDir = expand(c.ActivityDir)
	c.ClaudeProjectDir = expand(c.ClaudeProjectDir)
}
