package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/state"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tmux"
)

var idleCmd = &cobra.Command{
	Use:   "idle-update",
	Short: "Batch update window tab colors based on idle time",
	RunE:  runIdleUpdate,
}

func init() {
	rootCmd.AddCommand(idleCmd)
}

func runIdleUpdate(cmd *cobra.Command, args []string) error {
	session, err := tmux.CurrentSession()
	if err != nil {
		return err
	}
	activeWin, err := tmux.CurrentWindowID()
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	windows, err := tmux.ListSessionWindows(session)
	if err != nil {
		return err
	}

	for _, w := range windows {
		if w.WindowID == activeWin {
			continue
		}

		idleSecs := int64(99999)
		if act, err := state.ReadActivity(cfg.ActivityDir, w.WindowID); err == nil {
			idleSecs = now - act.Unix()
		}

		color := cfg.Theme.IdleColorFor(int(idleSecs))

		style := fmt.Sprintf("fg=%s,bg=default", color)
		if w.BellFlag {
			style += ",bold"
		}

		flags := strings.ReplaceAll(w.WindowFlags, "#", "")

		idx := ""
		if parts := strings.SplitN(w.Target, ":", 2); len(parts) == 2 {
			idx = parts[1]
		}

		format := fmt.Sprintf("#[%s] %s:%s%s ", style, idx, w.WindowName, flags)
		_ = tmux.SetWindowOption(w.WindowID, "window-status-format", format)
	}

	cleanupThrottled(now)
	return nil
}

func cleanupThrottled(now int64) {
	markerPath := cfg.ActivityDir + "/.last_cleanup"
	data, err := os.ReadFile(markerPath)
	if err == nil {
		if ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			if now-ts < 60 {
				return
			}
		}
	}

	os.WriteFile(markerPath, []byte(strconv.FormatInt(now, 10)), 0644)

	allWindows, err := tmux.ListAllWindows()
	if err != nil {
		return
	}
	valid := make(map[string]bool)
	for _, w := range allWindows {
		valid[w.CleanID()] = true
	}

	_ = state.CleanOrphans(cfg.ActivityDir, valid)
}
