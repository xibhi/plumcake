package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/xibhi/plumcake/internal/tui/theme"
)

type Styles struct {
	Theme theme.Palette

	Text    lipgloss.Style
	Muted   lipgloss.Style
	Accent  lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style

	Wordmark   lipgloss.Style
	TabActive  lipgloss.Style
	TabPassive lipgloss.Style

	Surface lipgloss.Style
	Border  lipgloss.Style
	Panel   lipgloss.Style

	SelectedRow lipgloss.Style
	NormalRow   lipgloss.Style
	Caret       lipgloss.Style

	KeyChip lipgloss.Style
	Footer  lipgloss.Style
}

func New(t theme.Palette) Styles {
	if theme.IsNoColor() {
		return Styles{
			Theme: t,

			Text:    lipgloss.NewStyle(),
			Muted:   lipgloss.NewStyle().Faint(true),
			Accent:  lipgloss.NewStyle().Bold(true),
			Success: lipgloss.NewStyle().Bold(true),
			Warning: lipgloss.NewStyle().Bold(true),
			Error:   lipgloss.NewStyle().Bold(true),

			Wordmark: lipgloss.NewStyle().Bold(true),

			TabActive: lipgloss.NewStyle().
				Bold(true).
				Underline(true),

			TabPassive: lipgloss.NewStyle().Faint(true),

			Surface: lipgloss.NewStyle(),

			Border: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()),

			Panel: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Padding(0, 1),

			SelectedRow: lipgloss.NewStyle().
				Bold(true).
				Underline(true),

			NormalRow: lipgloss.NewStyle(),

			Caret: lipgloss.NewStyle().Bold(true),

			KeyChip: lipgloss.NewStyle().Bold(true),

			Footer: lipgloss.NewStyle().Faint(true),
		}
	}

	surface := theme.Resolve(t.Surface)
	text := theme.Resolve(t.Text)
	muted := theme.Resolve(t.Muted)
	accent := theme.Resolve(t.Accent)
	success := theme.Resolve(t.Success)
	warning := theme.Resolve(t.Warning)
	errColor := theme.Resolve(t.Error)
	border := theme.Resolve(t.Border)
	selection := theme.Resolve(t.Selection)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(text).
		Background(selection)

	if t.IsSystem {
		selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(text)
	}

	return Styles{
		Theme: t,

		Text:    lipgloss.NewStyle().Foreground(text),
		Muted:   lipgloss.NewStyle().Foreground(muted),
		Accent:  lipgloss.NewStyle().Foreground(accent).Bold(true),
		Success: lipgloss.NewStyle().Foreground(success).Bold(true),
		Warning: lipgloss.NewStyle().Foreground(warning).Bold(true),
		Error:   lipgloss.NewStyle().Foreground(errColor).Bold(true),

		Wordmark: lipgloss.NewStyle().Bold(true).Foreground(accent),

		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Underline(true),

		TabPassive: lipgloss.NewStyle().
			Foreground(muted),

		Surface: lipgloss.NewStyle().
			Background(surface),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),

		SelectedRow: selectedStyle,

		NormalRow: lipgloss.NewStyle().
			Foreground(text),

		Caret: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		KeyChip: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		Footer: lipgloss.NewStyle().
			Foreground(muted),
	}
}
