package tui

import (
	"testing"
	"time"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
)

func TestStatusIndicator(t *testing.T) {
	tests := []struct {
		name    string
		winName string
		command string
		expect  string
	}{
		{"resumed", "🔄 auth", "", "🔄"},
		{"paused", "⏸ auth", "", "⏸"},
		{"done", "✅ auth", "", "✅"},
		{"claude running", "project", "claude", "●"},
		{"no status", "project", "zsh", " "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &WindowEntry{
				Window: tmux.Window{WindowName: tt.winName, Command: tt.command},
			}
			if got := e.StatusIndicator(); got != tt.expect {
				t.Errorf("StatusIndicator() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestCleanName(t *testing.T) {
	tests := []struct {
		winName string
		expect  string
	}{
		{"🔄 auth-impl", "auth-impl"},
		{"⏸ auth-impl", "auth-impl"},
		{"✅ auth-impl", "auth-impl"},
		{"project ●", "project"},
		{"●", ""},
		{"plain-name", "plain-name"},
	}
	for _, tt := range tests {
		e := &WindowEntry{Window: tmux.Window{WindowName: tt.winName}}
		if got := e.CleanName(); got != tt.expect {
			t.Errorf("CleanName(%q) = %q, want %q", tt.winName, got, tt.expect)
		}
	}
}

func TestIdleString(t *testing.T) {
	now := time.Now()
	tests := []struct {
		activity time.Time
		expect   string
	}{
		{now.Add(-30 * time.Second), "30s"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-2 * 24 * time.Hour), "2d"},
	}
	for _, tt := range tests {
		e := &WindowEntry{Activity: tt.activity}
		if got := e.IdleString(now); got != tt.expect {
			t.Errorf("IdleString() = %q, want %q (activity offset: %v)", got, tt.expect, now.Sub(tt.activity))
		}
	}
}

func TestIdleSecondsFromWindowActivity(t *testing.T) {
	now := time.Now()
	e := &WindowEntry{
		Window: tmux.Window{Activity: now.Unix() - 120},
	}
	secs := e.IdleSeconds(now)
	if secs < 119 || secs > 121 {
		t.Errorf("expected ~120 seconds, got %d", secs)
	}
}
