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
	if m.viewMode == "projects" {
		cols := fmt.Sprintf(" %-2s %-24s  %8s  %9s  %s", "ST", "PROJECT", "SESSIONS", "CHANGES", "STATUS")
		if len(cols) > width {
			cols = cols[:width]
		}
		return styles.HeaderStyle.Width(width).Render(cols)
	}
	cols := fmt.Sprintf(" %-2s %-24s  %-20s  %5s  %9s  %s", "ST", "TASK", "DIR", "MSG", "CHANGES", "PROMPT")
	if len(cols) > width {
		cols = cols[:width]
	}
	return styles.HeaderStyle.Width(width).Render(cols)
}

func (m Model) renderTable(width, height int) string {
	if m.viewMode == "projects" {
		return m.renderProjectTable(width, height)
	}
	return m.renderSessionTable(width, height)
}

func (m Model) renderProjectTable(width, height int) string {
	if len(m.projects) == 0 {
		return styles.DimStyle.Render("No projects found")
	}

	var lines []string
	total := len(m.projects)
	start := m.scrollOffset
	if start > total-height {
		start = total - height
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
	}

	for i := start; i < end; i++ {
		p := &m.projects[i]
		line := m.formatProjectRow(p, i, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m Model) formatProjectRow(p *projectEntry, cursorPos int, width int) string {
	// Status indicator — most urgent status in the project
	var st string
	switch {
	case p.waiting > 0:
		st = "⚠"
	case p.paused > 0:
		st = "⏸"
	case p.running > 0:
		st = "🔄"
	case p.done > 0:
		st = "✅"
	default:
		st = " "
	}

	name := p.name
	if len(name) > 24 {
		name = name[:24]
	}

	sessions := fmt.Sprintf("%8d", p.sessionCount)

	chgStr := fmt.Sprintf("%9s", "-")
	if p.linesAdded > 0 {
		chgStr = fmt.Sprintf("+%d", p.linesAdded)
		if len(chgStr) < 9 {
			chgStr = fmt.Sprintf("%9s", chgStr)
		}
		chgColor := styles.ColorGreen
		if p.linesAdded > 500 {
			chgColor = styles.ColorYellow
		}
		chgStr = lipgloss.NewStyle().Foreground(chgColor).Render(chgStr)
	}

	// Status breakdown
	var statusParts []string
	if p.running > 0 {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(styles.ColorGreen).Render(fmt.Sprintf("%d running", p.running)))
	}
	if p.waiting > 0 {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(styles.ColorOrange).Render(fmt.Sprintf("%d waiting", p.waiting)))
	}
	if p.paused > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d paused", p.paused))
	}
	if p.done > 0 {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(styles.ColorFgDim).Render(fmt.Sprintf("%d done", p.done)))
	}
	status := strings.Join(statusParts, ", ")

	row := fmt.Sprintf(" %s %-24s  %s  %s  %s", st, name, sessions, chgStr, status)

	if len(row) > width && width > 0 {
		row = row[:width]
	}

	if cursorPos == m.cursor {
		return styles.SelectedRowStyle.Width(width).Render(row)
	}
	return row
}

func (m Model) renderSessionTable(width, height int) string {
	if len(m.filtered) == 0 {
		msg := "No sessions found"
		if m.searchQuery != "" {
			msg = "No matches for: " + m.searchQuery
		}
		return styles.DimStyle.Render(msg)
	}

	now := time.Now()
	var lines []string

	visibleStart, visibleEnd := m.visibleRange(height)

	for i := visibleStart; i < visibleEnd; i++ {
		idx := m.filtered[i]
		line := m.formatRow(idx, i, now, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m Model) formatRow(entryIdx, cursorPos int, now time.Time, width int) string {
	e := &m.entries[entryIdx]

	// Status indicator with context warning
	st := e.StatusIndicator()
	if e.ContextPct != nil && *e.ContextPct >= 80 {
		st = lipgloss.NewStyle().Foreground(styles.ColorBrightRed).Render("⚠")
	}

	// Name (truncated to 24 chars)
	name := e.CleanName()
	if name == "" && e.FirstPrompt != "" {
		name = e.FirstPrompt
	}
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

	// Message age with color
	msgSecs := e.MsgAgeSeconds(now)
	msgStr := e.MsgAgeString(now)
	msgColor := lipgloss.Color(m.cfg.Theme.IdleColorFor(msgSecs))
	msgStyled := lipgloss.NewStyle().Foreground(msgColor).Render(fmt.Sprintf("%5s", msgStr))

	// Lines changed (progress indicator)
	chgStr := fmt.Sprintf("%9s", "-")
	if e.LinesAdded > 0 || e.LinesRemoved > 0 {
		chgStr = fmt.Sprintf("+%d/-%d", e.LinesAdded, e.LinesRemoved)
		if len(chgStr) < 9 {
			chgStr = fmt.Sprintf("%9s", chgStr)
		}
		chgColor := styles.ColorGreen
		if e.LinesAdded+e.LinesRemoved > 500 {
			chgColor = styles.ColorYellow
		}
		chgStr = lipgloss.NewStyle().Foreground(chgColor).Render(chgStr)
	}

	// Prompt (firstPrompt is the most useful — tells you what the session is FOR)
	prompt := e.FirstPrompt
	if prompt == "" {
		prompt = e.Summary
	}
	if len(prompt) > 50 {
		prompt = prompt[:50]
	}

	// Stale marker
	stale := ""
	if msgSecs > 86400 {
		stale = " 💀"
	}

	row := fmt.Sprintf(" %s %-24s  %-20s  %s  %s  %s%s", st, name, dir, msgStyled, chgStr, prompt, stale)

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

// visibleRange returns the range of filtered indices to render.
// Uses the pre-computed scrollOffset for stable edge-based scrolling.
func (m Model) visibleRange(height int) (int, int) {
	total := len(m.filtered)
	if total <= height {
		return 0, total
	}
	start := m.scrollOffset
	if start > total-height {
		start = total - height
	}
	if start < 0 {
		start = 0
	}
	return start, start + height
}

func isClaudeRunning(e *tui.WindowEntry) bool {
	return len(e.Window.Command) >= 6 && e.Window.Command[:6] == "claude"
}

// idleColorForSecs returns the theme-configured idle color.
func idleColorForSecs(cfg *config.Config, secs int) string {
	return cfg.Theme.IdleColorFor(secs)
}
