package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/state"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "One-line status for tmux status-right",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	windows, err := tmux.ListAllWindows()
	if err != nil {
		return err
	}

	now := time.Now()
	active := 0
	paused := 0
	stale := 0

	for _, w := range windows {
		name := w.WindowName
		switch {
		case strings.HasPrefix(name, "🔄 ") || isClaudeCmd(w.Command):
			active++
		case strings.HasPrefix(name, "⏸ "):
			// Check if stale
			if act, err := state.ReadActivity(cfg.ActivityDir, w.WindowID); err == nil {
				if now.Sub(act).Hours() > 24 {
					stale++
					continue
				}
			}
			paused++
		}
	}

	parts := []string{}
	if active > 0 {
		parts = append(parts, fmt.Sprintf("%d active", active))
	}
	if paused > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting", paused))
	}
	if stale > 0 {
		parts = append(parts, fmt.Sprintf("%d stale", stale))
	}

	if len(parts) == 0 {
		fmt.Println("no sessions")
	} else {
		fmt.Println(strings.Join(parts, " | "))
	}
	return nil
}

func isClaudeCmd(cmd string) bool {
	return len(cmd) >= 6 && cmd[:6] == "claude"
}
