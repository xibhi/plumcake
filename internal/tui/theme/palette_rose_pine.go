package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var RosePine = Palette{
	ID:        "rose-pine",
	Name:      "Rosé Pine",
	IsLight:   false,
	BG:        lipgloss.Color("#191724"),
	Surface:   lipgloss.Color("#1f1d2e"),
	Text:      lipgloss.Color("#e0def4"),
	Muted:     lipgloss.Color("#908caa"),
	Accent:    lipgloss.Color("#ebbcba"),
	Success:   lipgloss.Color("#9ccfd8"),
	Warning:   lipgloss.Color("#f6c177"),
	Error:     lipgloss.Color("#eb6f92"),
	Border:    lipgloss.Color("#26233a"),
	Selection: lipgloss.Color("#403d52"),
}
