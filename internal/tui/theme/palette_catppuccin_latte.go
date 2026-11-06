package theme

import (
	"github.com/charmbracelet/lipgloss"
)

var CatppuccinLatte = Palette{
	ID:        "catppuccin-latte",
	Name:      "Catppuccin Latte",
	IsLight:   true,
	BG:        lipgloss.Color("#eff1f5"),
	Surface:   lipgloss.Color("#ccd0da"),
	Text:      lipgloss.Color("#4c4f69"),
	Muted:     lipgloss.Color("#6c6f85"),
	Accent:    lipgloss.Color("#8839ef"),
	Success:   lipgloss.Color("#179299"),
	Warning:   lipgloss.Color("#e64553"),
	Error:     lipgloss.Color("#d20f39"),
	Border:    lipgloss.Color("#bcc0cc"),
	Selection: lipgloss.Color("#acb0be"),
}
