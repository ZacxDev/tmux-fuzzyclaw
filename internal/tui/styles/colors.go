package styles

import "github.com/charmbracelet/lipgloss"

// Gruvbox palette colors.
var (
	ColorBg        = lipgloss.Color("#282828")
	ColorFg        = lipgloss.Color("#ebdbb2")
	ColorFgDim     = lipgloss.Color("#928374")
	ColorRed       = lipgloss.Color("#cc241d")
	ColorGreen     = lipgloss.Color("#b8bb26")
	ColorYellow    = lipgloss.Color("#d79921")
	ColorBlue      = lipgloss.Color("#458588")
	ColorPurple    = lipgloss.Color("#b16286")
	ColorAqua      = lipgloss.Color("#689d6a")
	ColorOrange    = lipgloss.Color("#d65d0e")
	ColorGray      = lipgloss.Color("#665c54")
	ColorBgLight   = lipgloss.Color("#504945")
	ColorBrightRed = lipgloss.Color("#fb4934")
)

// StatusColors maps task status indicators to colors.
var StatusColors = map[string]lipgloss.Color{
	"active":  ColorGreen,
	"paused":  ColorYellow,
	"resumed": ColorAqua,
	"stale":   ColorGray,
	"bell":    ColorBrightRed,
}
