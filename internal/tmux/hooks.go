package tmux

import "fmt"

// InstallHooks sets up tmux hooks for activity tracking.
func InstallHooks(scriptDir string) error {
	hooks := []struct {
		event, action string
	}{
		{"after-select-window", fmt.Sprintf("run-shell -b '%s/pipe-activity.sh switch'", scriptDir)},
		{"window-linked", fmt.Sprintf("run-shell -b '%s/pipe-activity.sh linked'", scriptDir)},
		{"session-created", fmt.Sprintf("run-shell -b '%s/pipe-activity.sh init'", scriptDir)},
	}
	for _, h := range hooks {
		if err := RunSilent("set-hook", "-g", h.event, h.action); err != nil {
			return fmt.Errorf("install hook %s: %w", h.event, err)
		}
	}
	return nil
}
