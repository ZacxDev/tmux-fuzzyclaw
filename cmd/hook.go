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
	Use:   "hook <stop|resume|session-start|session-end|notification>",
	Short: "Handle Claude Code hook events",
	Args:  cobra.ExactArgs(1),
	RunE:  runHook,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

// hookInput represents the JSON sent by Claude Code on hook events.
type hookInput struct {
	SessionID        string `json:"session_id"`
	TranscriptPath   string `json:"transcript_path"`
	Cwd              string `json:"cwd"`
	PermissionMode   string `json:"permission_mode"`
	HookEventName    string `json:"hook_event_name"`
	StopHookActive   bool   `json:"stop_hook_active"`
	LastAssistantMsg string `json:"last_assistant_message"`
	NotificationType string `json:"notification_type,omitempty"`
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
	case "session-start":
		return hookSessionStart(tmuxPane)
	case "session-end":
		return hookSessionEnd(tmuxPane)
	case "notification":
		return hookNotification(tmuxPane)
	default:
		return fmt.Errorf("unknown hook action: %s (expected stop|resume|session-start|session-end|notification)", action)
	}
}

// readHookInput reads and parses JSON from stdin.
func readHookInput() (*hookInput, error) {
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	var input hookInput
	if err := json.Unmarshal(inputData, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// resolvedCwd returns the cwd from hook input, falling back to tmux query.
func resolvedCwd(input *hookInput, tmuxPane string) string {
	if input.Cwd != "" {
		return input.Cwd
	}
	cwd, err := tmux.DisplayMessage(tmuxPane, "#{pane_current_path}")
	if err != nil {
		return ""
	}
	return cwd
}

func hookStop(tmuxPane string) error {
	input, err := readHookInput()
	if err != nil {
		return nil // silent failure like bash version
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
	cwdDir := resolvedCwd(input, tmuxPane)

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
	transcriptPath := input.TranscriptPath
	if existing, err := state.ReadTask(cfg.StateDir, winID); err == nil {
		if existing.Task != "" && existing.Task != "●" {
			origTask = existing.Task
		}
		if existing.Started != "" {
			started = existing.Started
		}
		// Preserve transcript path if not provided in this event
		if transcriptPath == "" && existing.TranscriptPath != "" {
			transcriptPath = existing.TranscriptPath
		}
	}

	// Status lives in the JSON task state below, not the window name —
	// the window name is left to tmux automatic-rename so the tab tracks cwd.

	// Parse window index
	var windowIndex int
	fmt.Sscanf(winIdx, "%d", &windowIndex)

	// Write task state
	task := &state.TaskState{
		Task:           origTask,
		WindowID:       winID,
		TmuxSession:    sessionName,
		WindowIndex:    windowIndex,
		Status:         "paused",
		Cwd:            cwdDir,
		ClaudeSession:  input.SessionID,
		Started:        started,
		LastActivity:   now,
		Summary:        summary,
		TranscriptPath: transcriptPath,
	}
	return state.WriteTask(cfg.StateDir, task)
}

func hookResume(tmuxPane string) error {
	winID, err := tmux.DisplayMessage(tmuxPane, "#{window_id}")
	if err != nil {
		return nil
	}

	// Mark the task running again (idempotent). Status is the single source of
	// truth read by the dashboard, counters, and scratch indicator — the window
	// name is left to tmux automatic-rename so the tab keeps tracking cwd.
	existing, err := state.ReadTask(cfg.StateDir, winID)
	if err != nil {
		return nil // no task yet; Stop/SessionStart will create it
	}
	if existing.Status == "running" {
		return nil
	}
	existing.Status = "running"
	existing.LastActivity = time.Now().Format(time.RFC3339)
	return state.WriteTask(cfg.StateDir, existing)
}

func hookSessionStart(tmuxPane string) error {
	input, err := readHookInput()
	if err != nil {
		return nil
	}

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
	cwdDir := resolvedCwd(input, tmuxPane)

	now := time.Now().Format(time.RFC3339)

	taskName := stripEmojiPrefix(winName)
	taskName = strings.TrimSuffix(taskName, " ●")
	if taskName == "●" || taskName == "" {
		taskName = filepath.Base(cwdDir)
	}

	// Status tracked via JSON state below; window name left to automatic-rename.

	var windowIndex int
	fmt.Sscanf(winIdx, "%d", &windowIndex)

	task := &state.TaskState{
		Task:           taskName,
		WindowID:       winID,
		TmuxSession:    sessionName,
		WindowIndex:    windowIndex,
		Status:         "running",
		Cwd:            cwdDir,
		ClaudeSession:  input.SessionID,
		Started:        now,
		LastActivity:   now,
		TranscriptPath: input.TranscriptPath,
	}
	return state.WriteTask(cfg.StateDir, task)
}

func hookSessionEnd(tmuxPane string) error {
	input, err := readHookInput()
	if err != nil {
		return nil
	}

	winID, err := tmux.DisplayMessage(tmuxPane, "#{window_id}")
	if err != nil {
		return nil
	}

	now := time.Now().Format(time.RFC3339)

	// Status tracked via JSON state below; window name left to automatic-rename.

	// Update existing task state to "done"
	if existing, err := state.ReadTask(cfg.StateDir, winID); err == nil {
		existing.Status = "done"
		existing.LastActivity = now
		if input.TranscriptPath != "" {
			existing.TranscriptPath = input.TranscriptPath
		}
		return state.WriteTask(cfg.StateDir, existing)
	}

	return nil
}

func hookNotification(tmuxPane string) error {
	input, err := readHookInput()
	if err != nil {
		return nil
	}

	// Only handle permission prompts as "waiting" state
	if input.NotificationType != "permission_prompt" {
		return nil
	}

	winID, err := tmux.DisplayMessage(tmuxPane, "#{window_id}")
	if err != nil {
		return nil
	}

	// Update task state to waiting
	if existing, err := state.ReadTask(cfg.StateDir, winID); err == nil {
		existing.Status = "waiting"
		existing.LastActivity = time.Now().Format(time.RFC3339)
		return state.WriteTask(cfg.StateDir, existing)
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
