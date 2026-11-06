package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var Nord = Palette{
	ID:        "nord",
	Name:      "Nord",
	IsLight:   false,
	BG:        lipgloss.Color("#2e3440"),
	Surface:   lipgloss.Color("#3b4252"),
	Text:      lipgloss.Color("#eceff4"),
	Muted:     lipgloss.Color("#94a3b8"),
	Accent:    lipgloss.Color("#88c0d0"),
	Success:   lipgloss.Color("#a3be8c"),
	Warning:   lipgloss.Color("#ebcb8b"),
	Error:     lipgloss.Color("#bf616a"),
	Border:    lipgloss.Color("#434c5e"),
	Selection: lipgloss.Color("#4c566a"),
}
