package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var CatppuccinMocha = Palette{
	ID:        "catppuccin-mocha",
	Name:      "Catppuccin Mocha",
	IsLight:   false,
	BG:        lipgloss.Color("#1e1e2e"),
	Surface:   lipgloss.Color("#313244"),
	Text:      lipgloss.Color("#cdd6f4"),
	Muted:     lipgloss.Color("#a6adc8"),
	Accent:    lipgloss.Color("#cba6f7"),
	Success:   lipgloss.Color("#a6e3a1"),
	Warning:   lipgloss.Color("#f9e2af"),
	Error:     lipgloss.Color("#f38ba8"),
	Border:    lipgloss.Color("#45475a"),
	Selection: lipgloss.Color("#585b70"),
}
