package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/claude"
)

var statuslineCmd = &cobra.Command{
	Use:   "statusline",
	Short: "Receive Claude Code statusline JSON and write telemetry file",
	Long:  "Reads statusline JSON from stdin (sent by Claude Code) and writes telemetry to ~/.tmux/activity/<session_id>.statusline.json",
	RunE:  runStatusline,
}

func init() {
	rootCmd.AddCommand(statuslineCmd)
}

// TelemetryFile is the subset of statusline data written to disk for dashboard consumption.
type TelemetryFile struct {
	SessionID  string  `json:"session_id"`
	Model      string  `json:"model"`
	CostUSD    float64 `json:"cost_usd"`
	ContextPct *int    `json:"context_pct"`
	LinesAdded int     `json:"lines_added"`
	LinesRemoved int   `json:"lines_removed"`
	Cwd        string  `json:"cwd"`
}

func runStatusline(cmd *cobra.Command, args []string) error {
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil
	}

	sl, err := claude.ParseStatusline(inputData)
	if err != nil {
		return nil
	}

	if sl.SessionID == "" {
		return nil
	}

	tf := TelemetryFile{
		SessionID:    sl.SessionID,
		Model:        sl.Model.DisplayName,
		CostUSD:      sl.Cost.TotalCostUSD,
		ContextPct:   sl.ContextWindow.UsedPercentage,
		LinesAdded:   sl.Cost.TotalLinesAdded,
		LinesRemoved: sl.Cost.TotalLinesRemoved,
		Cwd:          sl.Cwd,
	}

	// Write telemetry file for dashboard consumption
	if err := os.MkdirAll(cfg.ActivityDir, 0755); err == nil {
		cleanID := strings.ReplaceAll(sl.SessionID, "/", "-")
		path := filepath.Join(cfg.ActivityDir, cleanID+".statusline.json")
		if data, err := json.Marshal(tf); err == nil {
			_ = os.WriteFile(path, data, 0644)
		}
	}

	// Output formatted status line for Claude Code display
	var parts []string
	parts = append(parts, sl.Model.DisplayName)
	if sl.ContextWindow.UsedPercentage != nil {
		pct := *sl.ContextWindow.UsedPercentage
		bar := contextBarASCII(pct, 10)
		parts = append(parts, fmt.Sprintf("ctx:%s %d%%", bar, pct))
	}
	if sl.Cost.TotalLinesAdded > 0 || sl.Cost.TotalLinesRemoved > 0 {
		parts = append(parts, fmt.Sprintf("+%d/-%d", sl.Cost.TotalLinesAdded, sl.Cost.TotalLinesRemoved))
	}
	fmt.Println(strings.Join(parts, " | "))
	return nil
}

func contextBarASCII(pct, width int) string {
	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
