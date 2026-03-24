package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/config"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/state"
)

// App is the root Bubble Tea model that routes between views.
type App struct {
	cfg     *config.Config
	watcher *state.Watcher
}

// NewApp creates the root application model.
func NewApp(cfg *config.Config) *App {
	return &App{cfg: cfg}
}

// StartWatcher creates fsnotify watchers for state directories.
func (a *App) StartWatcher() tea.Cmd {
	return func() tea.Msg {
		w, err := state.NewWatcher(a.cfg.StateDir, a.cfg.ActivityDir)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		a.watcher = w
		return watcherStartedMsg{}
	}
}

// WaitForFileChange waits for the next file change notification.
func (a *App) WaitForFileChange() tea.Cmd {
	if a.watcher == nil {
		return nil
	}
	w := a.watcher
	return func() tea.Msg {
		path, ok := <-w.Events
		if !ok {
			return nil
		}
		return FileChangedMsg{Path: path}
	}
}

// Close cleans up resources.
func (a *App) Close() {
	if a.watcher != nil {
		a.watcher.Close()
	}
}

type watcherStartedMsg struct{}
