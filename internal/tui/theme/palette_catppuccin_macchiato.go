package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var CatppuccinMacchiato = Palette{
	ID:        "catppuccin-macchiato",
	Name:      "Catppuccin Macchiato",
	IsLight:   false,
	BG:        lipgloss.Color("#24273a"),
	Surface:   lipgloss.Color("#363a4f"),
	Text:      lipgloss.Color("#cad3f5"),
	Muted:     lipgloss.Color("#a5adcb"),
	Accent:    lipgloss.Color("#c6a0f6"),
	Success:   lipgloss.Color("#a6da95"),
	Warning:   lipgloss.Color("#eed49f"),
	Error:     lipgloss.Color("#ed8796"),
	Border:    lipgloss.Color("#494d64"),
	Selection: lipgloss.Color("#5b6078"),
}
