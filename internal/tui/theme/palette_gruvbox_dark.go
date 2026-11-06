package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var GruvboxDark = Palette{
	ID:        "gruvbox-dark",
	Name:      "Gruvbox Dark",
	IsLight:   false,
	BG:        lipgloss.Color("#282828"),
	Surface:   lipgloss.Color("#3c3836"),
	Text:      lipgloss.Color("#ebdbb2"),
	Muted:     lipgloss.Color("#a89984"),
	Accent:    lipgloss.Color("#fe8019"),
	Success:   lipgloss.Color("#b8bb26"),
	Warning:   lipgloss.Color("#fabd2f"),
	Error:     lipgloss.Color("#fb4934"),
	Border:    lipgloss.Color("#504945"),
	Selection: lipgloss.Color("#504945"),
}
