package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Dashboard.SortBy != "idle" {
		t.Errorf("expected sort_by=idle, got %s", cfg.Dashboard.SortBy)
	}
	if cfg.Dashboard.RefreshInterval != 2*time.Second {
		t.Errorf("expected refresh_interval=2s, got %s", cfg.Dashboard.RefreshInterval)
	}
	if cfg.Dashboard.Preview.Width != 45 {
		t.Errorf("expected preview width=45, got %d", cfg.Dashboard.Preview.Width)
	}
	if len(cfg.Theme.IdleColors) != 8 {
		t.Errorf("expected 8 idle colors, got %d", len(cfg.Theme.IdleColors))
	}
}

func TestLoadMinimal(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "testdata", "config", "minimal.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dashboard.SortBy != "name" {
		t.Errorf("expected sort_by=name, got %s", cfg.Dashboard.SortBy)
	}
	if cfg.Dashboard.RefreshInterval != 5*time.Second {
		t.Errorf("expected refresh_interval=5s, got %s", cfg.Dashboard.RefreshInterval)
	}
	// Should keep defaults for unspecified fields
	if cfg.Dashboard.Preview.Width != 45 {
		t.Errorf("expected default preview width=45, got %d", cfg.Dashboard.Preview.Width)
	}
}

func TestLoadFull(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "testdata", "config", "full.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StateDir != "/tmp/test-tasks" {
		t.Errorf("expected state_dir=/tmp/test-tasks, got %s", cfg.StateDir)
	}
	if cfg.Dashboard.Preview.Width != 50 {
		t.Errorf("expected preview width=50, got %d", cfg.Dashboard.Preview.Width)
	}
	if len(cfg.Theme.IdleColors) != 3 {
		t.Errorf("expected 3 idle colors, got %d", len(cfg.Theme.IdleColors))
	}
	if cfg.Tokens.InputCost != 5.00 {
		t.Errorf("expected input_cost=5.00, got %f", cfg.Tokens.InputCost)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yml")
	if err != nil {
		t.Fatal("expected no error for missing config file")
	}
	if cfg.Dashboard.SortBy != "idle" {
		t.Error("expected defaults when config file missing")
	}
}

func TestExpandPaths(t *testing.T) {
	cfg := &Config{StateDir: "~/test"}
	cfg.expandPaths()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "test")
	if cfg.StateDir != expected {
		t.Errorf("expected %s, got %s", expected, cfg.StateDir)
	}
}
