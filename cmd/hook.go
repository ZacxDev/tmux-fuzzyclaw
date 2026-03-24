package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/state"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
)

var hookCmd = &cobra.Command{
	Use:   "hook <stop|resume>",
	Short: "Handle Claude Code hook events",
	Args:  cobra.ExactArgs(1),
	RunE:  runHook,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

// hookInput represents the JSON sent by Claude Code on Stop events.
type hookInput struct {
	SessionID         string `json:"session_id"`
	StopHookActive    bool   `json:"stop_hook_active"`
	LastAssistantMsg  string `json:"last_assistant_message"`
}

func runHook(cmd *cobra.Command, args []string) error {
	action := args[0]

	tmuxPane := os.Getenv("TMUX_PANE")
	if tmuxPane == "" {
		return nil // not in tmux, silently exit
	}

	switch action {
	case "stop":
		return hookStop(tmuxPane)
	case "resume":
		return hookResume(tmuxPane)
	default:
		return fmt.Errorf("unknown hook action: %s (expected stop|resume)", action)
	}
}

func hookStop(tmuxPane string) error {
	// Read stdin JSON
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil // silent failure like bash version
	}

	var input hookInput
	if err := json.Unmarshal(inputData, &input); err != nil {
		return nil
	}

	// Skip if hook is re-triggering
	if input.StopHookActive {
		return nil
	}

	// Get tmux window info
	winID, err := tmux.DisplayMessage(tmuxPane, "#{window_id}")
	if err != nil {
		return nil
	}
	winName, err := tmux.DisplayMessage(tmuxPane, "#{window_name}")
	if err != nil {
		return nil
	}
	sessionName, err := tmux.DisplayMessage(tmuxPane, "#{session_name}")
	if err != nil {
		return nil
	}
	winIdx, err := tmux.DisplayMessage(tmuxPane, "#{window_index}")
	if err != nil {
		return nil
	}
	cwdDir, err := tmux.DisplayMessage(tmuxPane, "#{pane_current_path}")
	if err != nil {
		return nil
	}

	now := time.Now().Format(time.RFC3339)

	// Strip emoji prefixes to get task name
	taskName := stripEmojiPrefix(winName)
	// Strip trailing Claude indicator
	taskName = strings.TrimSuffix(taskName, " ●")
	if taskName == "●" {
		taskName = ""
	}
	if taskName == "" {
		taskName = filepath.Base(cwdDir)
	}

	// Truncate last assistant message
	summary := input.LastAssistantMsg
	if len(summary) > 200 {
		summary = summary[:200]
	}
	summary = strings.ReplaceAll(summary, "\n", " ")

	// Read existing task to preserve original name and start time
	origTask := taskName
	started := now
	if existing, err := state.ReadTask(cfg.StateDir, winID); err == nil {
		if existing.Task != "" && existing.Task != "●" {
			origTask = existing.Task
		}
		if existing.Started != "" {
			started = existing.Started
		}
	}

	// Rename window with paused prefix
	_ = tmux.RenameWindow(winID, "⏸ "+origTask)

	// Parse window index
	var windowIndex int
	fmt.Sscanf(winIdx, "%d", &windowIndex)

	// Write task state
	task := &state.TaskState{
		Task:          origTask,
		WindowID:      winID,
		TmuxSession:   sessionName,
		WindowIndex:   windowIndex,
		Status:        "paused",
		Cwd:           cwdDir,
		ClaudeSession: input.SessionID,
		Started:       started,
		LastActivity:  now,
		Summary:       summary,
	}
	return state.WriteTask(cfg.StateDir, task)
}

func hookResume(tmuxPane string) error {
	winName, err := tmux.DisplayMessage(tmuxPane, "#{window_name}")
	if err != nil {
		return nil
	}

	// Flip ⏸ → 🔄 (idempotent)
	pausePrefix := "⏸ "
	if strings.HasPrefix(winName, pausePrefix) {
		newName := "🔄 " + winName[len(pausePrefix):]
		_ = tmux.RenameWindow(tmuxPane, newName)
	}
	return nil
}

func stripEmojiPrefix(name string) string {
	prefixes := []string{"🔄 ", "⏸ ", "✅ "}
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return name[len(p):]
		}
	}
	return name
}
