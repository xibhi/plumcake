package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var System = Palette{
	ID:        "system",
	Name:      "System",
	IsLight:   false,
	IsSystem:  true,
	BG:        lipgloss.Color("0"),
	Surface:   lipgloss.Color("8"),
	Text:      lipgloss.Color("7"),
	Muted:     lipgloss.Color("8"),
	Accent:    lipgloss.Color("4"),
	Success:   lipgloss.Color("2"),
	Warning:   lipgloss.Color("3"),
	Error:     lipgloss.Color("1"),
	Border:    lipgloss.Color("8"),
	Selection: lipgloss.Color("8"),
}
