package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
)

var pipeCmd = &cobra.Command{
	Use:   "pipe <start|stop|switch|linked|init> [window_id]",
	Short: "Manage pipe-pane for background activity tracking",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPipe,
}

func init() {
	rootCmd.AddCommand(pipeCmd)
}

func runPipe(cmd *cobra.Command, args []string) error {
	action := args[0]
	windowID := ""
	if len(args) > 1 {
		windowID = args[1]
	}

	receiverPath := findReceiverScript()

	switch action {
	case "start":
		if windowID == "" {
			return fmt.Errorf("window_id required for start")
		}
		return tmux.StartPipe(windowID, receiverPath)

	case "stop":
		if windowID == "" {
			return fmt.Errorf("window_id required for stop")
		}
		return tmux.StopPipe(windowID)

	case "switch":
		return tmux.SwitchPipes(cfg.ActivityDir, receiverPath)

	case "linked", "init":
		return tmux.InitPipes(receiverPath)

	default:
		return fmt.Errorf("unknown action: %s (expected start|stop|switch|linked|init)", action)
	}
}

// findReceiverScript locates the activity-receiver.sh script.
func findReceiverScript() string {
	// Check next to the binary first
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "activity-receiver.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Fall back to FUZZYCLAW_DIR or default
	if dir := os.Getenv("FUZZYCLAW_DIR"); dir != "" {
		return filepath.Join(dir, "activity-receiver.sh")
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tmux", "activity-receiver.sh")
}
