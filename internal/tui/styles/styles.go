package styles

import "github.com/charmbracelet/lipgloss"

// Base styles for the dashboard TUI.
var (
	// App chrome
	AppStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Header bar
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorFg).
			Background(ColorBgLight).
			Padding(0, 1)

	// Section header (e.g., "WAITING FOR INPUT")
	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBrightRed)

	// Table row - normal
	RowStyle = lipgloss.NewStyle().
			Foreground(ColorFg)

	// Table row - selected/cursor
	SelectedRowStyle = lipgloss.NewStyle().
				Foreground(ColorFg).
				Background(ColorBgLight).
				Bold(true)

	// Table row - multi-selected
	MarkedRowStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	// Status indicators
	StatusActive  = lipgloss.NewStyle().Foreground(ColorGreen).SetString("●")
	StatusPaused  = lipgloss.NewStyle().Foreground(ColorYellow).SetString("⏸")
	StatusResumed = lipgloss.NewStyle().Foreground(ColorAqua).SetString("🔄")
	StatusDone    = lipgloss.NewStyle().Foreground(ColorGreen).SetString("✅")

	// Preview panel
	PreviewBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorGray).
			Padding(0, 1)

	PreviewTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorFg)

	PreviewLabel = lipgloss.NewStyle().
			Foreground(ColorFgDim)

	PreviewValue = lipgloss.NewStyle().
			Foreground(ColorFg)

	// Search bar
	SearchStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBlue).
			Padding(0, 1)

	SearchPrompt = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

	// Status bar (bottom)
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorFgDim).
			Background(ColorBgLight).
			Padding(0, 1)

	// Idle time with color
	IdleStyle = lipgloss.NewStyle()

	// Dim text
	DimStyle = lipgloss.NewStyle().
			Foreground(ColorFgDim)

	// Error/warning
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorBrightRed).
			Bold(true)
)
