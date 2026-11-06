package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var OneDark = Palette{
	ID:        "one-dark",
	Name:      "One Dark",
	IsLight:   false,
	BG:        lipgloss.Color("#282c34"),
	Surface:   lipgloss.Color("#2c313a"),
	Text:      lipgloss.Color("#abb2bf"),
	Muted:     lipgloss.Color("#7f8796"),
	Accent:    lipgloss.Color("#61afef"),
	Success:   lipgloss.Color("#98c379"),
	Warning:   lipgloss.Color("#e5c07b"),
	Error:     lipgloss.Color("#e06c75"),
	Border:    lipgloss.Color("#3e4451"),
	Selection: lipgloss.Color("#3e4451"),
}
