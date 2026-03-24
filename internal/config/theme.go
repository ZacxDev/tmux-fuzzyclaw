package config

// ThemeConfig holds the color theme configuration.
type ThemeConfig struct {
	Name       string      `yaml:"name"`
	IdleColors []IdleColor `yaml:"idle_colors"`
}

// IdleColor maps an idle time threshold to a color.
type IdleColor struct {
	MaxMinutes int    `yaml:"max_minutes"`
	Color      string `yaml:"color"`
}

// DefaultTheme returns the Gruvbox idle color scale matching the bash implementation.
func DefaultTheme() ThemeConfig {
	return ThemeConfig{
		Name: "gruvbox",
		IdleColors: []IdleColor{
			{MaxMinutes: 10, Color: "#b8bb26"},
			{MaxMinutes: 30, Color: "#98971a"},
			{MaxMinutes: 60, Color: "#689d6a"},
			{MaxMinutes: 120, Color: "#d79921"},
			{MaxMinutes: 240, Color: "#d65d0e"},
			{MaxMinutes: 480, Color: "#cc241d"},
			{MaxMinutes: 1440, Color: "#b16286"},
			{MaxMinutes: 99999, Color: "#665c54"},
		},
	}
}

// IdleColor returns the color for a given idle duration in seconds.
func (t *ThemeConfig) IdleColorFor(idleSecs int) string {
	idleMin := idleSecs / 60
	for _, ic := range t.IdleColors {
		if idleMin < ic.MaxMinutes {
			return ic.Color
		}
	}
	if len(t.IdleColors) > 0 {
		return t.IdleColors[len(t.IdleColors)-1].Color
	}
	return "#665c54"
}
