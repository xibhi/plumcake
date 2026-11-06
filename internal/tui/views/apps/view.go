package apps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/sandbox"
	"github.com/xibhi/plumcake/internal/core/trust"
	"github.com/xibhi/plumcake/internal/tui/components/styles"
)

func (m Model) View(themeStyles styles.Styles) string {
	var builder strings.Builder

	if len(m.Apps) == 0 {
		emptyMsg := themeStyles.Muted.Render("  No applications added yet.\n\n  Go to [1 Add] to install an AppImage.")
		if m.StatusMessage != "" {
			stStyle := themeStyles.Success
			if m.StatusIsError {
				stStyle = themeStyles.Error
			}
			emptyMsg += "\n\n  " + stStyle.Render("● "+m.StatusMessage)
		}
		return emptyMsg
	}

	totalWidth := m.Width - 6
	if totalWidth < 74 {
		totalWidth = 74
	}
	leftWidth := int(float64(totalWidth) * 0.38)
	if leftWidth < 32 {
		leftWidth = 32
	}
	rightWidth := totalWidth - leftWidth - 2

	var listContent strings.Builder
	if m.Mode == ModeFilter {
		listContent.WriteString(themeStyles.Muted.Render("filter: ") + m.SearchInput.View() + "\n\n")
	}

	for i, app := range m.Filtered {
		cursor := "  "
		rowStyle := themeStyles.NormalRow
		if i == m.Cursor {
			cursor = themeStyles.Caret.Render("▎ ")
			rowStyle = themeStyles.SelectedRow
		}

		trustBadge := themeStyles.Success.Render(app.Trust.BadgeGlyph())
		if app.Trust == trust.SignedUnverified {
			trustBadge = themeStyles.Warning.Render("◐")
		} else if app.Trust == trust.Unsigned {
			trustBadge = themeStyles.Muted.Render("○")
		} else if app.Trust == trust.Mismatch {
			trustBadge = themeStyles.Error.Render("✗")
		}

		updateMarker := " "
		if app.UpdateInfo.HasUpdate {
			updateMarker = themeStyles.Accent.Bold(true).Render("↑")
		}

		sandboxMarker := " "
		if app.Sandbox.Enabled {
			sandboxMarker = themeStyles.KeyChip.Render("⛨")
		}

		ver := app.Version
		if ver == "" {
			ver = "1.0"
		}
		item := fmt.Sprintf("%s%s%s %-14s %s",
			trustBadge,
			updateMarker,
			sandboxMarker,
			registry.Truncate(app.Name, 14),
			themeStyles.Muted.Render(registry.Truncate(ver, 8)),
		)
		listContent.WriteString(cursor + rowStyle.Width(leftWidth-4).Render(item) + "\n")
	}

	leftBox := themeStyles.Border.Width(leftWidth).Render(
		themeStyles.Accent.Bold(true).Render(fmt.Sprintf("Installed (%d)", len(m.Filtered))) + "\n\n" +
			listContent.String(),
	)

	var rightBox string
	cur := m.SelectedApp()
	if cur != nil {
		sourceType := "Local binary"
		hasUpdateSource := "No"
		if cur.SourceURL != nil && *cur.SourceURL != "" {
			sourceType = *cur.SourceURL
			hasUpdateSource = "Yes (" + registry.Truncate(sourceType, 28) + ")"
		}

		iconStatus := "Missing"
		if len(cur.IconFiles) > 0 {
			iconStatus = fmt.Sprintf("%d file(s) installed", len(cur.IconFiles))
		}

		sbLabel := "Off"
		if cur.Sandbox.Enabled {
			netStr := "net off"
			if cur.Sandbox.Network {
				netStr = "net on"
			}
			sbLabel = fmt.Sprintf("%s (%s)", cur.Sandbox.Tool, netStr)
		}

		upMarker := "Up to date"
		if cur.UpdateInfo.HasUpdate {
			upMarker = fmt.Sprintf("Update available: %s (press 'u')", cur.UpdateInfo.LatestVersion)
		}

		details := fmt.Sprintf(
			"%s  %s  %s\n"+
				"%s\n\n"+
				"%s\n"+
				"  Trust state:    %s %s\n"+
				"  SHA256 digest:  %s\n"+
				"  Sandboxing:     %s  [press 's' to configure]\n"+
				"  Update status:  %s\n"+
				"  Installed size: %s\n"+
				"  Installed date: %s\n"+
				"  Update source:  %s\n\n"+
				"%s\n"+
				"  Binary path:    %s\n"+
				"  Desktop file:   %s\n"+
				"  Icon files:     %s\n"+
				"  CLI command:    %s\n"+
				"  WMClass:        %s\n"+
				"  Versions kept:  %d archived [press 'v' for history]",
			themeStyles.Accent.Bold(true).Render(cur.Name),
			themeStyles.Muted.Render("v"+cur.Version),
			themeStyles.Success.Render("["+string(cur.Trust)+"]"),
			themeStyles.Muted.Render(cur.Comment),
			themeStyles.Muted.Bold(true).Render("TRUST & SECURITY"),
			cur.Trust.BadgeGlyph(),
			string(cur.Trust),
			registry.Truncate(cur.BinarySHA256, 32),
			sbLabel,
			upMarker,
			registry.FormatSize(cur.BinarySize),
			cur.InstalledAt[:min(19, len(cur.InstalledAt))],
			hasUpdateSource,
			themeStyles.Muted.Bold(true).Render("SYSTEM INTEGRATION"),
			registry.Truncate(cur.AppImagePath, 34),
			registry.Truncate(cur.DesktopFile, 34),
			iconStatus,
			cur.ExecName,
			cur.WMClass,
			len(cur.VersionHistory),
		)
		rightBox = themeStyles.Border.Width(rightWidth).Render(details)
	}

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
	builder.WriteString("  " + split + "\n\n")

	switch m.Mode {
	case ModeRename:
		modal := themeStyles.Border.BorderForeground(themeStyles.Theme.Accent).Width(totalWidth).Render(
			themeStyles.Accent.Bold(true).Render("Rename Application") + "\n\n" +
				m.RenameInput.View() + "\n\n" +
				themeStyles.Muted.Render("Enter to save · Esc to cancel"),
		)
		builder.WriteString("  " + modal + "\n\n")

	case ModeChangeIcon:
		modal := themeStyles.Border.BorderForeground(themeStyles.Theme.Accent).Width(totalWidth).Render(
			themeStyles.Accent.Bold(true).Render("Change Application Icon") + "\n\n" +
				m.IconInput.View() + "\n" +
				m.IconValidMsg + "\n\n" +
				themeStyles.Muted.Render("Enter to apply · Esc to cancel"),
		)
		builder.WriteString("  " + modal + "\n\n")

	case ModeSandbox:
		avail := sandbox.Detect()
		firejailHint := "(available)"
		if !avail.HasFirejail {
			firejailHint = "(not installed: sudo pacman -S firejail)"
		}
		bwrapHint := "(available)"
		if !avail.HasBwrap {
			bwrapHint = "(not installed: sudo pacman -S bubblewrap)"
		}

		tools := []string{"off", "firejail", "bwrap"}
		toolRender := ""
		for idx, t := range tools {
			if idx == m.SandboxToolIdx {
				toolRender += themeStyles.Accent.Bold(true).Render("["+t+"]") + "  "
			} else {
				toolRender += themeStyles.Muted.Render(t) + "  "
			}
		}

		netStatus := "ALLOW NETWORK (on)"
		if !m.SandboxNet {
			netStatus = "ISOLATE NETWORK (off)"
		}

		modal := themeStyles.Border.BorderForeground(themeStyles.Theme.Accent).Width(totalWidth).Render(
			themeStyles.Accent.Bold(true).Render("Application Sandbox Settings: "+cur.Name) + "\n\n" +
				"  Engine:  " + toolRender + "\n" +
				"           firejail " + themeStyles.Muted.Render(firejailHint) + "  bwrap " + themeStyles.Muted.Render(bwrapHint) + "\n\n" +
				"  Network: " + themeStyles.KeyChip.Render(netStatus) + " (press 'n' to toggle)\n\n" +
				themeStyles.Muted.Render("←/→: select engine · n: toggle network · Enter: save · Esc: cancel"),
		)
		builder.WriteString("  " + modal + "\n\n")

	case ModeVersions:
		var verList strings.Builder
		for idx, v := range cur.VersionHistory {
			cursor := "  "
			if idx == m.VersionCursor {
				cursor = themeStyles.Caret.Render("▎ ")
			}
			verList.WriteString(fmt.Sprintf("%s %-12s  %s  %s\n",
				cursor,
				themeStyles.Accent.Render(v.Version),
				registry.FormatSize(v.Size),
				themeStyles.Muted.Render(v.InstalledAt[:min(19, len(v.InstalledAt))]),
			))
		}

		modal := themeStyles.Border.BorderForeground(themeStyles.Theme.Accent).Width(totalWidth).Render(
			themeStyles.Accent.Bold(true).Render(fmt.Sprintf("Version Rollback: %s", cur.Name)) + "\n\n" +
				verList.String() + "\n" +
				themeStyles.Muted.Render("Enter: switch to version · p: prune old archives · Esc: close"),
		)
		builder.WriteString("  " + modal + "\n\n")

	case ModeConfirmDelete:
		modal := themeStyles.Border.BorderForeground(themeStyles.Theme.Error).Width(totalWidth).Render(
			themeStyles.Error.Bold(true).Render(fmt.Sprintf("Move '%s' to Trash?", cur.Name)) + "\n\n" +
				themeStyles.Text.Render("Files will be archived in trash (retained for 7 days) and can be undone.") + "\n\n" +
				themeStyles.KeyChip.Render("y") + themeStyles.Muted.Render(" confirm    ") +
				themeStyles.KeyChip.Render("n / Esc") + themeStyles.Muted.Render(" cancel"),
		)
		builder.WriteString("  " + modal + "\n\n")
	}

	if m.StatusMessage != "" {
		stStyle := themeStyles.Success
		if m.StatusIsError {
			stStyle = themeStyles.Error
		}
		builder.WriteString("  " + stStyle.Render("● "+m.StatusMessage) + "\n\n")
	}

	return builder.String()
}
