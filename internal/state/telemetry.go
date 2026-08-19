package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Telemetry holds statusline data written by the statusline command.
type Telemetry struct {
	SessionID    string  `json:"session_id"`
	Model        string  `json:"model"`
	CostUSD      float64 `json:"cost_usd"`
	ContextPct   *int    `json:"context_pct"`
	LinesAdded   int     `json:"lines_added"`
	LinesRemoved int     `json:"lines_removed"`
	Cwd          string  `json:"cwd"`
}

// ReadAllTelemetry reads all statusline telemetry files from the activity directory.
// Returns a map keyed by session ID.
func ReadAllTelemetry(activityDir string) map[string]*Telemetry {
	result := make(map[string]*Telemetry)
	entries, err := os.ReadDir(activityDir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".statusline.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(activityDir, e.Name()))
		if err != nil {
			continue
		}
		var t Telemetry
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		if t.SessionID != "" {
			result[t.SessionID] = &t
		}
	}
	return result
}

// ReadTelemetryByCwd looks up telemetry by cwd (most recent file wins).
func ReadTelemetryByCwd(activityDir string) map[string]*Telemetry {
	result := make(map[string]*Telemetry)
	all := ReadAllTelemetry(activityDir)
	for _, t := range all {
		if t.Cwd != "" {
			result[t.Cwd] = t
		}
	}
	return result
}
