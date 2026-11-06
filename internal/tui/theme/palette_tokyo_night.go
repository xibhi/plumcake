package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var TokyoNight = Palette{
	ID:        "tokyo-night",
	Name:      "Tokyo Night",
	IsLight:   false,
	BG:        lipgloss.Color("#1a1b26"),
	Surface:   lipgloss.Color("#24283b"),
	Text:      lipgloss.Color("#c0caf5"),
	Muted:     lipgloss.Color("#7aa2f7"),
	Accent:    lipgloss.Color("#7aa2f7"),
	Success:   lipgloss.Color("#9ece6a"),
	Warning:   lipgloss.Color("#e0af68"),
	Error:     lipgloss.Color("#f7768e"),
	Border:    lipgloss.Color("#414868"),
	Selection: lipgloss.Color("#283457"),
}
