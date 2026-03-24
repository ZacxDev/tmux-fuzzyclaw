package config

// KeyConfig holds keybinding configuration.
type KeyConfig struct {
	Quit      []string `yaml:"quit"`
	Switch    []string `yaml:"switch"`
	Kill      []string `yaml:"kill"`
	Select    []string `yaml:"select"`
	SelectAll []string `yaml:"select_all"`
	Export    []string `yaml:"export"`
	Timeline  []string `yaml:"timeline"`
}

// DefaultKeys returns keybindings matching the plan.
func DefaultKeys() KeyConfig {
	return KeyConfig{
		Quit:      []string{"q", "ctrl+c", "esc"},
		Switch:    []string{"enter"},
		Kill:      []string{"ctrl+x"},
		Select:    []string{"tab"},
		SelectAll: []string{"ctrl+a"},
		Export:    []string{"ctrl+e"},
		Timeline:  []string{"ctrl+t"},
	}
}
