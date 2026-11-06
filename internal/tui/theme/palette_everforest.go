package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var Everforest = Palette{
	ID:        "everforest",
	Name:      "Everforest",
	IsLight:   false,
	BG:        lipgloss.Color("#2d353b"),
	Surface:   lipgloss.Color("#343f44"),
	Text:      lipgloss.Color("#d3c6aa"),
	Muted:     lipgloss.Color("#9da9a0"),
	Accent:    lipgloss.Color("#a7c080"),
	Success:   lipgloss.Color("#a7c080"),
	Warning:   lipgloss.Color("#dbbc7f"),
	Error:     lipgloss.Color("#e67e80"),
	Border:    lipgloss.Color("#475258"),
	Selection: lipgloss.Color("#3d484d"),
}
