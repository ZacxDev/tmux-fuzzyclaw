package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tui/styles"
)

func (m Model) renderPreview(width, height int) string {
	if m.viewMode == "projects" {
		return m.renderProjectPreview(width, height)
	}

	entry := m.currentEntry()
	if entry == nil {
		return styles.PreviewBorder.Width(width - 2).Height(height).Render(
			styles.DimStyle.Render("No session selected"),
		)
	}

	var sections []string

	// Task state section
	if entry.Task != nil {
		t := entry.Task
		sections = append(sections, styles.PreviewTitle.Render("Task State"))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Task:"), t.Task))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Status:"), statusWithColor(t.Status)))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Dir:"), t.Cwd))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Started:"), t.Started))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Activity:"), t.LastActivity))
		sections = append(sections, "")
		if t.Summary != "" {
			sections = append(sections, styles.PreviewTitle.Render("Last Claude Output"))
			sections = append(sections, wrapText(t.Summary, width-4))
		}
	} else {
		sections = append(sections, styles.PreviewTitle.Render("Window Info"))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Dir:"), entry.Window.FullCwd))
		sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Command:"), entry.Window.Command))
	}

	// Telemetry from statusline (when available)
	if entry.ContextPct != nil || entry.Model != "" {
		sections = append(sections, "")
		sections = append(sections, styles.PreviewTitle.Render("Telemetry"))
		if entry.Model != "" {
			sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Model:"), entry.Model))
		}
		if entry.ContextPct != nil {
			pct := *entry.ContextPct
			bar := contextBar(pct, width-18)
			sections = append(sections, fmt.Sprintf("  %s %s %d%%", styles.PreviewLabel.Render("Context:"), bar, pct))
		}
	}

	// Session metadata from index (when available)
	if entry.FirstPrompt != "" || entry.GitBranch != "" || entry.MessageCount > 0 {
		sections = append(sections, "")
		sections = append(sections, styles.PreviewTitle.Render("Session"))
		if entry.FirstPrompt != "" {
			prompt := entry.FirstPrompt
			if len(prompt) > 120 {
				prompt = prompt[:120]
			}
			sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Prompt:"), prompt))
		}
		if entry.GitBranch != "" {
			sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Branch:"), entry.GitBranch))
		}
		if entry.MessageCount > 0 {
			sections = append(sections, fmt.Sprintf("  %s %d", styles.PreviewLabel.Render("Messages:"), entry.MessageCount))
		}
	}

	// Search results or recent prompts
	if m.searchQuery != "" && len(m.searchResults) > 0 {
		sections = append(sections, "")
		sections = append(sections,
			styles.PreviewTitle.Render("Matches for ")+
				lipgloss.NewStyle().Foreground(styles.ColorYellow).Render(m.searchQuery))
		sections = append(sections, "")
		for i, r := range m.searchResults {
			if i >= 20 {
				sections = append(sections, styles.DimStyle.Render(fmt.Sprintf("  ... and %d more", len(m.searchResults)-20)))
				break
			}
			prefix := "  "
			if r.Type == "user" {
				prefix = "  > "
			}
			sections = append(sections, prefix+wrapText(r.Context, width-6))
		}
	} else if len(m.previewPrompts) > 0 {
		sections = append(sections, "")
		sections = append(sections, styles.PreviewTitle.Render("Recent Prompts"))
		for _, p := range m.previewPrompts {
			line := p
			if len(line) > 120 {
				line = line[:120]
			}
			sections = append(sections, "  > "+line)
		}
	} else if m.previewSummary != "" && entry.Task == nil {
		sections = append(sections, "")
		sections = append(sections, styles.PreviewTitle.Render("Latest Summary"))
		sections = append(sections, wrapText(m.previewSummary, width-4))
	}

	content := strings.Join(sections, "\n")

	// Truncate to height — lipgloss Height() only pads, it doesn't clip overflow
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	content = strings.Join(lines, "\n")

	return styles.PreviewBorder.Width(width - 2).Height(height).Render(content)
}

func (m Model) renderProjectPreview(width, height int) string {
	if m.cursor < 0 || m.cursor >= len(m.projects) {
		return styles.PreviewBorder.Width(width - 2).Height(height).Render(
			styles.DimStyle.Render("No project selected"),
		)
	}

	p := &m.projects[m.cursor]
	var sections []string

	sections = append(sections, styles.PreviewTitle.Render("Project"))
	sections = append(sections, fmt.Sprintf("  %s %s", styles.PreviewLabel.Render("Path:"), p.cwd))
	sections = append(sections, fmt.Sprintf("  %s %d", styles.PreviewLabel.Render("Sessions:"), p.sessionCount))
	if p.linesAdded > 0 {
		sections = append(sections, fmt.Sprintf("  %s +%d", styles.PreviewLabel.Render("Lines:"), p.linesAdded))
	}

	sections = append(sections, "")
	sections = append(sections, styles.PreviewTitle.Render("Status"))
	if p.running > 0 {
		sections = append(sections, fmt.Sprintf("  %s %d", styles.PreviewLabel.Render("Running:"), p.running))
	}
	if p.waiting > 0 {
		sections = append(sections, fmt.Sprintf("  %s %d", styles.PreviewLabel.Render("Waiting:"), p.waiting))
	}
	if p.paused > 0 {
		sections = append(sections, fmt.Sprintf("  %s %d", styles.PreviewLabel.Render("Paused:"), p.paused))
	}
	if p.done > 0 {
		sections = append(sections, fmt.Sprintf("  %s %d", styles.PreviewLabel.Render("Done:"), p.done))
	}

	// List session prompts for this project
	sections = append(sections, "")
	sections = append(sections, styles.PreviewTitle.Render("Sessions"))
	for i := range m.entries {
		e := &m.entries[i]
		cwd := e.Window.FullCwd
		if cwd == "" {
			cwd = e.Window.Dir
		}
		if cwd != p.cwd {
			continue
		}
		st := e.StatusIndicator()
		prompt := e.FirstPrompt
		if prompt == "" {
			prompt = e.CleanName()
		}
		if prompt == "" {
			prompt = e.Summary
		}
		if len(prompt) > width-8 {
			prompt = prompt[:width-8]
		}
		sections = append(sections, fmt.Sprintf("  %s %s", st, prompt))
	}

	content := strings.Join(sections, "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	content = strings.Join(lines, "\n")

	return styles.PreviewBorder.Width(width - 2).Height(height).Render(content)
}

func statusWithColor(status string) string {
	color, ok := styles.StatusColors[status]
	if !ok {
		color = styles.ColorFgDim
	}
	return lipgloss.NewStyle().Foreground(color).Render(status)
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) > width {
			lines = append(lines, current)
			current = w
		} else {
			current += " " + w
		}
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n  ")
}

func contextBar(pct, width int) string {
	if width < 4 {
		width = 4
	}
	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	color := styles.ColorGreen
	if pct >= 80 {
		color = styles.ColorBrightRed
	} else if pct >= 60 {
		color = styles.ColorOrange
	} else if pct >= 40 {
		color = styles.ColorYellow
	}
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

func (m Model) renderStatusBar() string {
	// Fleet status counts
	var running, waiting, paused, done, totalLines int
	for i := range m.entries {
		e := &m.entries[i]
		switch m.entryStatus(e) {
		case "running":
			running++
		case "waiting":
			waiting++
		case "paused":
			paused++
		case "done":
			done++
		}
		totalLines += e.LinesAdded
	}

	var parts []string
	if running > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorGreen).Render(fmt.Sprintf("%d running", running)))
	}
	if waiting > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorOrange).Render(fmt.Sprintf("%d waiting", waiting)))
	}
	if paused > 0 {
		parts = append(parts, fmt.Sprintf("%d paused", paused))
	}
	if done > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorFgDim).Render(fmt.Sprintf("%d done", done)))
	}
	if totalLines > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorGreen).Render(fmt.Sprintf("+%d lines", totalLines)))
	}

	selectedCount := len(m.selected)
	if selectedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d selected", selectedCount))
	}
	if m.statusFilter != "" {
		parts = append(parts, fmt.Sprintf("filter:%s", m.statusFilter))
	}
	left := strings.Join(parts, " · ")

	var help string
	if m.viewMode == "projects" {
		help = "j/k:nav  enter:drill in  p:all sessions  q:quit"
	} else if m.selectedProject != "" {
		help = "j/k:nav  enter:switch  esc:back  /:search  1-4:filter  q:quit"
	} else {
		help = "j/k:nav  enter:switch  p:projects  /:search  1-4:filter  q:quit"
	}

	bar := left + strings.Repeat(" ", max(0, m.width-len(left)-len(help)-2)) + help

	return styles.StatusBarStyle.Width(m.width).Render(bar)
}
