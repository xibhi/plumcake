package help

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xibhi/plumcake/internal/tui/components/styles"
)

func Render(themeStyles styles.Styles, width, height int) string {
	var builder strings.Builder

	title := themeStyles.Accent.Bold(true).Render("plumcake") + themeStyles.Muted.Render(" · A sweet way to run AppImages")
	divider := themeStyles.Muted.Render(strings.Repeat("─", 52))

	sections := []struct {
		name string
		keys [][2]string
	}{
		{
			name: "GLOBAL",
			keys: [][2]string{
				{"Tab / Shift+Tab", "Next / previous tab"},
				{"1 / 2 / 3", "Jump to Add / Apps / Settings"},
				{"?", "Toggle this help modal"},
				{"q / Ctrl+C", "Quit plumcake"},
			},
		},
		{
			name: "ADD SCREEN",
			keys: [][2]string{
				{"↑ / ↓ or j / k", "Navigate detected AppImages"},
				{"Enter", "Add selected file"},
				{"/", "Focus input field for URL or path"},
				{"s", "Scan deeper (entire $HOME)"},
				{"p", "Toggle portable mode (sandboxed)"},
			},
		},
		{
			name: "APPS SCREEN",
			keys: [][2]string{
				{"↑ / ↓ or j / k", "Navigate installed apps"},
				{"r", "Rename application (updates .desktop Name)"},
				{"i", "Change icon (validates PNG/SVG)"},
				{"l", "Launch application detached"},
				{"u", "Remove / uninstall cleanly"},
				{"o", "Open containing folder in file manager"},
				{"/", "Live search / filter apps"},
			},
		},
		{
			name: "SETTINGS SCREEN",
			keys: [][2]string{
				{"Enter", "Apply doctor fix / save preference"},
				{"← / → or h / l", "Toggle preference options"},
				{"r", "Refresh system doctor audits"},
			},
		},
	}

	for _, section := range sections {
		builder.WriteString(themeStyles.Accent.Bold(true).Render(section.name) + "\n")
		for _, keyPair := range section.keys {
			builder.WriteString("  " + themeStyles.KeyChip.Width(18).Render(keyPair[0]) + " " + themeStyles.Text.Render(keyPair[1]) + "\n")
		}
		builder.WriteString("\n")
	}

	content := title + "\n" + divider + "\n\n" + builder.String() + "\n" +
		themeStyles.Muted.Render("Press '?' or 'Esc' to close")

	boxWidth := 60
	if width < 66 {
		boxWidth = width - 6
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(themeStyles.Theme.Accent).
		Background(themeStyles.Theme.BG).
		Padding(1, 3).
		Width(boxWidth).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
