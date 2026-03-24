package tmux

import (
	"fmt"
	"os"
	"path/filepath"
)

// StartPipe sets up pipe-pane on the first pane of a window to track activity.
func StartPipe(windowID, receiverPath string) error {
	panes, err := ListPanes(windowID)
	if err != nil || len(panes) == 0 {
		return err
	}
	cmd := fmt.Sprintf("'%s' '%s'", receiverPath, windowID)
	return PipePane(panes[0], cmd)
}

// StopPipe removes pipe-pane from the first pane of a window.
func StopPipe(windowID string) error {
	panes, err := ListPanes(windowID)
	if err != nil || len(panes) == 0 {
		return err
	}
	return PipePane(panes[0], "")
}

// SwitchPipes handles the hot-path window switch: stop pipe on current, start on previous.
func SwitchPipes(activityDir, receiverPath string) error {
	session, err := CurrentSession()
	if err != nil {
		return err
	}
	currentWin, err := CurrentWindowID()
	if err != nil {
		return err
	}

	// Stop pipe on current (now focused)
	_ = StopPipe(currentWin)

	// Start pipe on previous (now in background)
	prevFile := filepath.Join(activityDir, ".prev_"+session)
	if data, err := os.ReadFile(prevFile); err == nil {
		prevWin := string(data)
		if prevWin != currentWin {
			_ = StartPipe(prevWin, receiverPath)
		}
	}

	// Track current for next switch
	return os.WriteFile(prevFile, []byte(currentWin), 0644)
}

// InitPipes starts pipes on all background windows in the current session.
func InitPipes(receiverPath string) error {
	format := "#{window_id}\t#{window_active}"
	out, err := Run("list-windows", "-F", format)
	if err != nil {
		return err
	}
	for _, line := range splitLines(out) {
		parts := splitTab(line)
		if len(parts) == 2 && parts[1] == "0" {
			_ = StartPipe(parts[0], receiverPath)
		}
	}
	return nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitTab(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
