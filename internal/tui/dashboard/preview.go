package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zachatrocern/tmux-fuzzyclaw/internal/tui/styles"
)

func (m Model) renderPreview(width, height int) string {
	entry := m.currentEntry()
	if entry == nil {
		return styles.PreviewBorder.Width(width - 2).Height(height).Render(
			styles.DimStyle.Render("No window selected"),
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

func (m Model) renderStatusBar() string {
	total := len(m.entries)
	visible := len(m.filtered)
	selectedCount := len(m.selected)

	parts := []string{
		fmt.Sprintf("%d windows", total),
	}
	if visible != total {
		parts = append(parts, fmt.Sprintf("%d shown", visible))
	}
	if selectedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d selected", selectedCount))
	}

	left := strings.Join(parts, " | ")

	help := "j/k:nav  enter:switch  /:search  tab:select  ctrl+x:kill  q:quit"

	bar := left + strings.Repeat(" ", max(0, m.width-len(left)-len(help)-2)) + help

	return styles.StatusBarStyle.Width(m.width).Render(bar)
}
