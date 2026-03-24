package tui

import (
	"time"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/claude"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
)

// WindowEntry is the merged view of a window for dashboard display.
type WindowEntry struct {
	Window   tmux.Window
	Task     *TaskSnapshot
	Activity time.Time
	Summary  string
	Keywords string
}

// TaskSnapshot holds task state data for display.
type TaskSnapshot struct {
	Task          string
	Status        string
	Cwd           string
	ClaudeSession string
	Started       string
	LastActivity  string
	Summary       string
}

// StatusIndicator returns the display indicator for this entry.
func (e *WindowEntry) StatusIndicator() string {
	name := e.Window.WindowName
	switch {
	case len(name) > 2 && name[:len("🔄")] == "🔄":
		return "🔄"
	case len(name) > 2 && name[:len("⏸")] == "⏸":
		return "⏸"
	case len(name) > 2 && name[:len("✅")] == "✅":
		return "✅"
	case isClaudeCommand(e.Window.Command):
		return "●"
	default:
		return " "
	}
}

// CleanName returns the window name with status emoji prefix stripped.
func (e *WindowEntry) CleanName() string {
	name := e.Window.WindowName
	prefixes := []string{"🔄 ", "⏸ ", "✅ "}
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			name = name[len(p):]
			break
		}
	}
	// Strip trailing " ●" (● is multi-byte: 3 bytes)
	suffix := " ●"
	if len(name) >= len(suffix) && name[len(name)-len(suffix):] == suffix {
		name = name[:len(name)-len(suffix)]
	}
	if name == "●" {
		name = ""
	}
	return name
}

// IdleSeconds returns how many seconds since last activity.
func (e *WindowEntry) IdleSeconds(now time.Time) int {
	if e.Activity.IsZero() {
		if e.Window.Activity > 0 {
			return int(now.Unix() - e.Window.Activity)
		}
		return 99999
	}
	return int(now.Sub(e.Activity).Seconds())
}

// IdleString returns a human-readable idle time string.
func (e *WindowEntry) IdleString(now time.Time) string {
	secs := e.IdleSeconds(now)
	switch {
	case secs < 60:
		return itoa(secs) + "s"
	case secs < 3600:
		return itoa(secs/60) + "m"
	case secs < 86400:
		return itoa(secs/3600) + "h"
	default:
		return itoa(secs/86400) + "d"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 5)
	for n > 0 {
		b = append(b, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func isClaudeCommand(cmd string) bool {
	return len(cmd) >= 6 && cmd[:6] == "claude"
}

// --- Bubble Tea messages ---

// WindowsRefreshedMsg carries refreshed window list data.
type WindowsRefreshedMsg struct {
	Entries []WindowEntry
}

// SearchResultsMsg carries search results.
type SearchResultsMsg struct {
	Results []claude.SearchResult
	Query   string
}

// ConversationLoadedMsg carries loaded conversation data for preview.
type ConversationLoadedMsg struct {
	WindowID string
	Prompts  []string
	Summary  string
}

// FileChangedMsg indicates a state file changed (from fsnotify).
type FileChangedMsg struct {
	Path string
}

// RefreshTickMsg triggers idle time display updates.
type RefreshTickMsg struct{}

// DataPollMsg triggers data re-reads.
type DataPollMsg struct{}

// SwitchWindowMsg requests switching to a window.
type SwitchWindowMsg struct {
	Target string
}

// KillWindowsMsg requests killing selected windows.
type KillWindowsMsg struct {
	Targets []string
}

// ErrorMsg carries an error to display.
type ErrorMsg struct {
	Err error
}
