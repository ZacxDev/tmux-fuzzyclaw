package config

import "testing"

func TestIdleColorFor(t *testing.T) {
	theme := DefaultTheme()

	tests := []struct {
		idleSecs int
		expected string
	}{
		{0, "#b8bb26"},       // <10 min = bright green
		{300, "#b8bb26"},     // 5 min = bright green
		{900, "#98971a"},     // 15 min = green
		{2400, "#689d6a"},    // 40 min = aqua
		{5400, "#d79921"},    // 90 min = yellow
		{10800, "#d65d0e"},   // 3 hr = orange
		{21600, "#cc241d"},   // 6 hr = red
		{57600, "#b16286"},   // 16 hr = purple
		{172800, "#665c54"},  // 48 hr = gray
	}

	for _, tt := range tests {
		got := theme.IdleColorFor(tt.idleSecs)
		if got != tt.expected {
			t.Errorf("IdleColorFor(%d) = %s, want %s", tt.idleSecs, got, tt.expected)
		}
	}
}
