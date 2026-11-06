package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var Dracula = Palette{
	ID:        "dracula",
	Name:      "Dracula",
	IsLight:   false,
	BG:        lipgloss.Color("#282a36"),
	Surface:   lipgloss.Color("#44475a"),
	Text:      lipgloss.Color("#f8f8f2"),
	Muted:     lipgloss.Color("#949fba"),
	Accent:    lipgloss.Color("#bd93f9"),
	Success:   lipgloss.Color("#50fa7b"),
	Warning:   lipgloss.Color("#f1fa8c"),
	Error:     lipgloss.Color("#ff5555"),
	Border:    lipgloss.Color("#6272a4"),
	Selection: lipgloss.Color("#44475a"),
}
