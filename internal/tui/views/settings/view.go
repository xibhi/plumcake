package settings

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/tui/components/styles"
	"github.com/xibhi/plumcake/internal/tui/theme"
)

func (m Model) View(themeStyles styles.Styles) string {
	var builder strings.Builder

	totalWidth := m.Width - 6
	if totalWidth < 74 {
		totalWidth = 74
	}
	leftWidth := 20
	rightWidth := totalWidth - leftWidth - 2

	sections := []struct {
		id   Section
		name string
	}{
		{SectionPreferences, "Preferences"},
		{SectionDoctor, "Doctor"},
		{SectionTrash, fmt.Sprintf("Trash (%d)", len(m.TrashItems))},
		{SectionLog, "Log"},
		{SectionPortability, "Portability"},
	}

	var menu strings.Builder
	for _, sec := range sections {
		cursor := "  "
		style := themeStyles.NormalRow
		if m.ActiveSection == sec.id {
			cursor = themeStyles.Caret.Render("▎ ")
			style = themeStyles.SelectedRow
		}
		menu.WriteString(cursor + style.Width(leftWidth-4).Render(sec.name) + "\n\n")
	}

	leftBox := themeStyles.Border.Width(leftWidth).Render(
		themeStyles.Accent.Bold(true).Render("SETTINGS") + "\n\n" + menu.String(),
	)

	var rightContent strings.Builder
	switch m.ActiveSection {
	case SectionPreferences:
		if m.InThemePicker {
			rightContent.WriteString(m.renderThemePicker(themeStyles, rightWidth))
		} else {
			currentThemeName := theme.Find(m.Config.Theme).Name
			prefItems := []struct {
				label string
				val   string
			}{
				{"Theme (press Enter to choose)", currentThemeName},
				{"Pre-install review", fmt.Sprintf("%t", m.Config.PreinstallInspect)},
				{"Reduced motion", fmt.Sprintf("%t", m.Config.ReducedMotion)},
				{"Check updates on start", fmt.Sprintf("%t", m.Config.CheckUpdatesOnStart)},
				{"Include prereleases", fmt.Sprintf("%t", m.Config.IncludePrereleases)},
				{"Deep scan on start", fmt.Sprintf("%t", m.Config.DeepScan)},
				{"Confirm before delete", fmt.Sprintf("%t", m.Config.ConfirmDelete)},
				{"Move instead of copy", fmt.Sprintf("%t", m.Config.DefaultMoveFile)},
				{"Versions to keep", fmt.Sprintf("%d", m.Config.KeepVersions)},
				{"Trash retention (days)", fmt.Sprintf("%d", m.Config.TrashRetentionDays)},
			}

			for i, item := range prefItems {
				cursor := "  "
				style := themeStyles.NormalRow
				if i == m.PrefCursor {
					cursor = themeStyles.Caret.Render("▎ ")
					style = themeStyles.SelectedRow
				}
				row := fmt.Sprintf("%-30s %s", item.label, themeStyles.Accent.Render(item.val))
				rightContent.WriteString(cursor + style.Width(rightWidth-4).Render(row) + "\n")
			}
		}

	case SectionDoctor:
		for i, chk := range m.DoctorChecks {
			cursor := "  "
			style := themeStyles.NormalRow
			if i == m.DoctorCursor {
				cursor = themeStyles.Caret.Render("▎ ")
				style = themeStyles.SelectedRow
			}

			statusGlyph := themeStyles.Success.Render("✓")
			if chk.Status == "warn" {
				statusGlyph = themeStyles.Warning.Render("!")
			} else if chk.Status == "fail" {
				statusGlyph = themeStyles.Error.Render("✗")
			}

			fixHint := ""
			if chk.CanFix {
				fixHint = themeStyles.KeyChip.Render(" [Enter to fix]")
			}

			row := fmt.Sprintf("%s %s%s\n    %s\n    %s",
				statusGlyph,
				themeStyles.Text.Bold(true).Render(chk.Title),
				fixHint,
				themeStyles.Muted.Render(chk.Message),
				themeStyles.Muted.Render(registry.Truncate(chk.Path, rightWidth-10)),
			)
			rightContent.WriteString(cursor + style.Width(rightWidth-4).Render(row) + "\n\n")
		}

	case SectionTrash:
		if len(m.TrashItems) == 0 {
			rightContent.WriteString(themeStyles.Muted.Render("  Trash is empty. Soft-removed apps are retained here.\n"))
		} else {
			for i, item := range m.TrashItems {
				cursor := "  "
				style := themeStyles.NormalRow
				if i == m.TrashCursor {
					cursor = themeStyles.Caret.Render("▎ ")
					style = themeStyles.SelectedRow
				}
				row := fmt.Sprintf("%-18s  deleted: %s  expires: %s\n    %s",
					item.AppManifest.Name,
					item.DeletedAt[:min(10, len(item.DeletedAt))],
					item.ExpiresAt[:min(10, len(item.ExpiresAt))],
					themeStyles.KeyChip.Render("[Enter to restore]"),
				)
				rightContent.WriteString(cursor + style.Width(rightWidth-4).Render(row) + "\n\n")
			}
			rightContent.WriteString("\n  " + themeStyles.KeyChip.Render("e") + themeStyles.Muted.Render(" empty trash completely\n"))
		}

	case SectionLog:
		if len(m.Logs) == 0 {
			rightContent.WriteString(themeStyles.Muted.Render("  No operation logs recorded yet.\n"))
		} else {
			for i, lg := range m.Logs {
				stStyle := themeStyles.Success
				if lg.Status == "fail" {
					stStyle = themeStyles.Error
				} else if lg.Status == "warn" {
					stStyle = themeStyles.Warning
				}

				row := fmt.Sprintf("[%s] %-12s %s: %s",
					lg.Timestamp[:min(19, len(lg.Timestamp))],
					themeStyles.Accent.Render(lg.Action),
					stStyle.Render(lg.Status),
					themeStyles.Text.Render(registry.Truncate(lg.Message, rightWidth-36)),
				)
				rightContent.WriteString(row + "\n")
				if i >= 12 {
					break
				}
			}
		}

	case SectionPortability:
		rightContent.WriteString(themeStyles.Text.Bold(true).Render("Export Apps Manifest:") + "\n")
		rightContent.WriteString(themeStyles.Muted.Render("Writes portable TOML manifest without absolute paths or tokens.") + "\n")
		rightContent.WriteString("  File: " + m.ExportPathIn.View() + "\n")
		rightContent.WriteString("  " + themeStyles.KeyChip.Render("[e]") + themeStyles.Muted.Render(" Edit export path & Enter to save\n\n"))

		rightContent.WriteString(themeStyles.Text.Bold(true).Render("Import Apps Manifest:") + "\n")
		rightContent.WriteString(themeStyles.Muted.Render("Validates manifest plan first, then adds each package.") + "\n")
		rightContent.WriteString("  File: " + m.ImportPathIn.View() + "\n")
		rightContent.WriteString("  " + themeStyles.KeyChip.Render("[i]") + themeStyles.Muted.Render(" Edit import path & Enter to evaluate plan\n\n"))

		if m.ImportPlan != nil {
			rightContent.WriteString(themeStyles.Accent.Bold(true).Render("Import Plan:") + "\n")
			rightContent.WriteString(fmt.Sprintf("  • Will install:     %d package(s)\n", len(m.ImportPlan.WillInstall)))
			rightContent.WriteString(fmt.Sprintf("  • Already present:  %d package(s)\n", len(m.ImportPlan.AlreadyPresent)))
		}
	}

	rightBox := themeStyles.Border.Width(rightWidth).Render(
		rightContent.String(),
	)

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
	builder.WriteString("  " + split + "\n\n")

	if m.StatusMessage != "" {
		stStyle := themeStyles.Success
		if m.StatusIsError {
			stStyle = themeStyles.Error
		}
		builder.WriteString("  " + stStyle.Render("● "+m.StatusMessage) + "\n\n")
	}

	return builder.String()
}

func (m Model) renderThemePicker(themeStyles styles.Styles, width int) string {
	var builder strings.Builder

	title := themeStyles.Accent.Bold(true).Render("THEME PICKER (LIVE PREVIEW)")
	builder.WriteString(title + "\n")

	if theme.IsNoColor() {
		builder.WriteString(themeStyles.Warning.Render("  (NO_COLOR is set in environment: terminal colors are disabled)") + "\n")
	}
	builder.WriteString(themeStyles.Muted.Render("  j/k or ↑/↓: move (previews immediately) · Enter: save · Esc: cancel\n\n"))

	allThemes := theme.GetAll()

	renderThemeRow := func(idx int, t theme.Palette) string {
		cursor := "  "
		style := themeStyles.NormalRow
		if idx == m.PickerCursor {
			cursor = themeStyles.Caret.Render("▎ ")
			style = themeStyles.SelectedRow
		}

		savedMarker := " "
		if t.ID == m.Config.Theme {
			savedMarker = themeStyles.Success.Render("●")
		}

		var swatches string
		if theme.IsNoColor() {
			swatches = "[ - - - - - ]"
		} else if t.IsSystem {
			swatches = fmt.Sprintf("[%s %s %s %s %s]",
				lipgloss.NewStyle().Foreground(t.Accent).Render("■"),
				lipgloss.NewStyle().Foreground(t.Success).Render("■"),
				lipgloss.NewStyle().Foreground(t.Warning).Render("■"),
				lipgloss.NewStyle().Foreground(t.Error).Render("■"),
				lipgloss.NewStyle().Foreground(t.Muted).Render("■"),
			)
		} else {
			swatches = fmt.Sprintf("[%s %s %s %s %s]",
				lipgloss.NewStyle().Foreground(theme.Resolve(t.Accent)).Render("■"),
				lipgloss.NewStyle().Foreground(theme.Resolve(t.Success)).Render("■"),
				lipgloss.NewStyle().Foreground(theme.Resolve(t.Warning)).Render("■"),
				lipgloss.NewStyle().Foreground(theme.Resolve(t.Error)).Render("■"),
				lipgloss.NewStyle().Foreground(theme.Resolve(t.Muted)).Render("■"),
			)
		}

		item := fmt.Sprintf("%s %-22s  %s", savedMarker, t.Name, swatches)
		return cursor + style.Width(width-8).Render(item) + "\n"
	}

	builder.WriteString(themeStyles.Muted.Bold(true).Render("  System") + "\n")
	for i, t := range allThemes {
		if t.IsSystem {
			builder.WriteString(renderThemeRow(i, t))
		}
	}
	builder.WriteString("\n")

	builder.WriteString(themeStyles.Muted.Bold(true).Render("  Dark") + "\n")
	for i, t := range allThemes {
		if !t.IsSystem && !t.IsLight && !t.IsCustom {
			builder.WriteString(renderThemeRow(i, t))
		}
	}
	builder.WriteString("\n")

	builder.WriteString(themeStyles.Muted.Bold(true).Render("  Light") + "\n")
	for i, t := range allThemes {
		if !t.IsSystem && t.IsLight && !t.IsCustom {
			builder.WriteString(renderThemeRow(i, t))
		}
	}

	hasCustom := false
	for _, t := range allThemes {
		if t.IsCustom {
			hasCustom = true
			break
		}
	}
	if hasCustom {
		builder.WriteString("\n" + themeStyles.Muted.Bold(true).Render("  Custom (~/.config/plumcake/themes/*.toml)") + "\n")
		for i, t := range allThemes {
			if t.IsCustom {
				builder.WriteString(renderThemeRow(i, t))
			}
		}
	}

	if len(m.CustomWarnings) > 0 {
		builder.WriteString("\n" + themeStyles.Warning.Render("  Custom theme warnings:") + "\n")
		for _, w := range m.CustomWarnings {
			builder.WriteString(themeStyles.Muted.Render("    • "+w) + "\n")
		}
	}

	return builder.String()
}
