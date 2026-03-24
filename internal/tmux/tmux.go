package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes a tmux command and returns stdout.
func Run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// RunSilent executes a tmux command ignoring output.
func RunSilent(args ...string) error {
	_, err := Run(args...)
	return err
}

// DisplayMessage runs tmux display-message -t target -p format.
func DisplayMessage(target, format string) (string, error) {
	return Run("display-message", "-t", target, "-p", format)
}

// SwitchClient switches the tmux client to the given target.
func SwitchClient(target string) error {
	// Try switch-client first (for sessions), fall back to select-window
	if err := RunSilent("switch-client", "-t", target); err != nil {
		return RunSilent("select-window", "-t", target)
	}
	return nil
}

// KillWindow kills the specified tmux window.
func KillWindow(target string) error {
	return RunSilent("kill-window", "-t", target)
}

// RenameWindow renames a tmux window.
func RenameWindow(target, name string) error {
	return RunSilent("rename-window", "-t", target, name)
}

// SetWindowOption sets a window option on a specific window.
func SetWindowOption(target, option, value string) error {
	return RunSilent("set-window-option", "-t", target, option, value)
}

// CurrentSession returns the current tmux session name.
func CurrentSession() (string, error) {
	return DisplayMessage("", "#{session_name}")
}

// CurrentWindowID returns the current window ID.
func CurrentWindowID() (string, error) {
	return DisplayMessage("", "#{window_id}")
}

// PipePane sets or clears pipe-pane on a given pane.
func PipePane(target string, command string) error {
	if command == "" {
		return RunSilent("pipe-pane", "-t", target)
	}
	return RunSilent("pipe-pane", "-t", target, command)
}

// ListPanes returns pane IDs for a window.
func ListPanes(target string) ([]string, error) {
	out, err := Run("list-panes", "-t", target, "-F", "#{pane_id}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}
