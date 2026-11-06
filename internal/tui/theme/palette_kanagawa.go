package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var Kanagawa = Palette{
	ID:        "kanagawa",
	Name:      "Kanagawa",
	IsLight:   false,
	BG:        lipgloss.Color("#1f1f28"),
	Surface:   lipgloss.Color("#2a2a37"),
	Text:      lipgloss.Color("#dcd7ba"),
	Muted:     lipgloss.Color("#8a8980"),
	Accent:    lipgloss.Color("#7e9cd8"),
	Success:   lipgloss.Color("#76946a"),
	Warning:   lipgloss.Color("#dca561"),
	Error:     lipgloss.Color("#c34043"),
	Border:    lipgloss.Color("#363646"),
	Selection: lipgloss.Color("#2d4f67"),
}
