package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/config"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tui"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tui/styles"
)

func (m Model) renderHeader(width int) string {
	cols := fmt.Sprintf(" %-2s %-24s  %-20s  %5s  %s", "ST", "TASK", "DIR", "IDLE", "SUMMARY")
	if len(cols) > width {
		cols = cols[:width]
	}
	return styles.HeaderStyle.Width(width).Render(cols)
}

func (m Model) renderTable(width, height int) string {
	if len(m.filtered) == 0 {
		msg := "No windows found"
		if m.searchQuery != "" {
			msg = "No matches for: " + m.searchQuery
		}
		return styles.DimStyle.Render(msg)
	}

	now := time.Now()
	var lines []string

	// Determine visible range (scroll around cursor)
	visibleStart, visibleEnd := m.visibleRange(height)

	// Check if we should show the "WAITING FOR INPUT" section
	if m.searchQuery == "" && m.cfg.Dashboard.ShowSections {
		var bellEntries []int
		for _, idx := range m.filtered {
			e := &m.entries[idx]
			if e.Window.BellFlag && isClaudeRunning(e) {
				bellEntries = append(bellEntries, idx)
			}
		}
		if len(bellEntries) > 0 {
			lines = append(lines, styles.SectionStyle.Render("── WAITING FOR INPUT ──"))
			for _, idx := range bellEntries {
				line := m.formatRow(idx, -1, now, width)
				lines = append(lines, line)
			}
			lines = append(lines, "")
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		idx := m.filtered[i]
		line := m.formatRow(idx, i, now, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m Model) formatRow(entryIdx, cursorPos int, now time.Time, width int) string {
	e := &m.entries[entryIdx]

	// Status indicator
	st := e.StatusIndicator()

	// Name (truncated to 24 chars)
	name := e.CleanName()
	if name == "" {
		name = e.Window.Dir
	}
	if len(name) > 24 {
		name = name[:24]
	}

	// Dir (truncated to 20 chars)
	dir := e.Window.Dir
	if len(dir) > 20 {
		dir = dir[:20]
	}

	// Idle time with color
	idleSecs := e.IdleSeconds(now)
	idleStr := e.IdleString(now)
	idleColor := lipgloss.Color(m.cfg.Theme.IdleColorFor(idleSecs))
	idleStyled := lipgloss.NewStyle().Foreground(idleColor).Render(fmt.Sprintf("%5s", idleStr))

	// Summary (truncated to 55 chars)
	summary := e.Summary
	if len(summary) > 55 {
		summary = summary[:55]
	}

	// Stale marker
	stale := ""
	if idleSecs > 86400 {
		stale = " 💀"
	}

	row := fmt.Sprintf(" %s %-24s  %-20s  %s  %s%s", st, name, dir, idleStyled, summary, stale)

	// Truncate to width
	// Note: this is approximate due to unicode widths, but good enough
	if len(row) > width && width > 0 {
		row = row[:width]
	}

	// Apply style based on cursor/selection state
	isCursor := cursorPos == m.cursor
	isSelected := m.selected[entryIdx]

	switch {
	case isCursor && isSelected:
		return lipgloss.NewStyle().Foreground(styles.ColorYellow).Background(styles.ColorBgLight).Bold(true).Width(width).Render(row)
	case isCursor:
		return styles.SelectedRowStyle.Width(width).Render(row)
	case isSelected:
		return styles.MarkedRowStyle.Render(row)
	default:
		return row
	}
}

func (m Model) visibleRange(height int) (int, int) {
	total := len(m.filtered)
	if total <= height {
		return 0, total
	}

	// Keep cursor in middle of visible range
	half := height / 2
	start := m.cursor - half
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
		start = end - height
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func isClaudeRunning(e *tui.WindowEntry) bool {
	return len(e.Window.Command) >= 6 && e.Window.Command[:6] == "claude"
}

// idleColorForSecs returns the theme-configured idle color.
func idleColorForSecs(cfg *config.Config, secs int) string {
	return cfg.Theme.IdleColorFor(secs)
}
