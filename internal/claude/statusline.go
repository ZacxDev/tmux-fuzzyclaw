package claude

import "encoding/json"

// StatuslineData represents the JSON sent by Claude Code to statusline commands.
type StatuslineData struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	Version        string          `json:"version"`
	Model          StatuslineModel `json:"model"`
	Cost           StatuslineCost  `json:"cost"`
	ContextWindow  StatuslineCtx   `json:"context_window"`
	RateLimits     *RateLimits     `json:"rate_limits,omitempty"`
}

// StatuslineModel identifies the Claude model in use.
type StatuslineModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// StatuslineCost tracks session cost and code changes.
type StatuslineCost struct {
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalDurationMs   int64   `json:"total_duration_ms"`
	TotalAPIDuration  int64   `json:"total_api_duration_ms"`
	TotalLinesAdded   int     `json:"total_lines_added"`
	TotalLinesRemoved int     `json:"total_lines_removed"`
}

// StatuslineCtx tracks context window usage.
type StatuslineCtx struct {
	TotalInputTokens  int          `json:"total_input_tokens"`
	TotalOutputTokens int          `json:"total_output_tokens"`
	ContextWindowSize int          `json:"context_window_size"`
	UsedPercentage    *int         `json:"used_percentage"`
	RemainingPct      *int         `json:"remaining_percentage"`
	CurrentUsage      *TokenUsage  `json:"current_usage"`
}

// TokenUsage details the current API call token counts.
type TokenUsage struct {
	InputTokens            int `json:"input_tokens"`
	OutputTokens           int `json:"output_tokens"`
	CacheCreationTokens    int `json:"cache_creation_input_tokens"`
	CacheReadTokens        int `json:"cache_read_input_tokens"`
}

// RateLimits tracks usage limits for Claude.ai Pro/Max.
type RateLimits struct {
	FiveHour *RateWindow `json:"five_hour,omitempty"`
	SevenDay *RateWindow `json:"seven_day,omitempty"`
}

// RateWindow represents a rolling rate limit window.
type RateWindow struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

// ParseStatusline parses JSON statusline data from Claude Code.
func ParseStatusline(data []byte) (*StatuslineData, error) {
	var s StatuslineData
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
