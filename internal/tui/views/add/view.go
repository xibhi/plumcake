package add

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xibhi/plumcake/internal/core/installer"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/trust"
	"github.com/xibhi/plumcake/internal/tui/components/styles"
)

func (m Model) View(themeStyles styles.Styles) string {
	var builder strings.Builder

	inputBox := themeStyles.Panel.Width(m.Width - 6).Render(
		m.Input.View(),
	)
	builder.WriteString("  " + inputBox + "\n\n")

	if m.Mode == ModeReview && m.ReviewItem != nil {
		builder.WriteString(m.renderReview(themeStyles) + "\n")
		return builder.String()
	}

	if m.Mode == ModeInstalling || m.Mode == ModeCompleted || m.Mode == ModeFailed {
		builder.WriteString(m.renderPipeline(themeStyles) + "\n")
		return builder.String()
	}

	scanStatus := themeStyles.Muted.Render("scan complete")
	if m.IsScanning {
		scanStatus = themeStyles.Warning.Render("scanning $HOME...")
	} else if m.DeepScanning {
		scanStatus = themeStyles.Warning.Render("deep scanning filesystem...")
	}

	headerText := fmt.Sprintf("  %-32s %-10s %-24s %-12s %s",
		"NAME", "SIZE", "FOLDER", "MODIFIED", "STATUS",
	)
	builder.WriteString("  " + themeStyles.Text.Bold(true).Render(fmt.Sprintf("Discovered AppImages (%d)", len(m.Filtered))) +
		"  " + scanStatus + "\n\n")
	builder.WriteString(themeStyles.Muted.Render(headerText) + "\n")
	builder.WriteString(themeStyles.Muted.Render("  "+strings.Repeat("─", m.Width-10)) + "\n")

	if len(m.Filtered) == 0 {
		builder.WriteString("\n" + themeStyles.Muted.Render("    No .AppImage binaries found matching query.\n    Paste URL, github.com/owner/repo, or press [F5] to scan deeper.") + "\n\n")
	} else {
		maxRows := 10
		for i, img := range m.Filtered {
			if i >= maxRows {
				break
			}

			cursor := "  "
			rowStyle := themeStyles.NormalRow
			if i == m.Cursor {
				cursor = themeStyles.Caret.Render("▎ ")
				rowStyle = themeStyles.SelectedRow
			}

			status := themeStyles.Success.Render("● added")
			if !img.IsAdded {
				status = themeStyles.Text.Render("○ ready")
			}

			row := fmt.Sprintf("%-32s %-10s %-24s %-12s %s",
				registry.Truncate(img.Name, 30),
				registry.FormatSize(img.Size),
				registry.Truncate(img.Folder, 22),
				img.ModTime.Format("2006-01-02"),
				status,
			)
			builder.WriteString(cursor + rowStyle.Width(m.Width-10).Render(row) + "\n")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func (m Model) renderReview(themeStyles styles.Styles) string {
	rev := m.ReviewItem
	var builder strings.Builder

	archMatchStr := themeStyles.Success.Render("✓ matches machine (" + rev.Arch + ")")
	if !rev.ArchMatches {
		archMatchStr = themeStyles.Error.Render("✗ non-matching architecture (" + rev.Arch + ")")
	}

	trustStr := themeStyles.Success.Render(rev.TrustResult.State.BadgeGlyph() + " " + string(rev.TrustResult.State))
	if rev.TrustResult.State == trust.SignedUnverified {
		trustStr = themeStyles.Warning.Render("◐ signed-unverified (key " + rev.TrustResult.SignatureKey + ")")
	} else if rev.TrustResult.State == trust.Unsigned {
		trustStr = themeStyles.Muted.Render("○ unsigned")
	} else if rev.TrustResult.State == trust.Mismatch {
		trustStr = themeStyles.Error.Render("✗ mismatch")
	}

	moveStr := "Move to ~/.local/share/plumcake"
	if !m.MoveFileChoice {
		moveStr = "Copy to ~/.local/share/plumcake"
	}

	content := fmt.Sprintf(
		"%s\n\n"+
			"  App Name:     %s\n"+
			"  Version:      %s\n"+
			"  Size:         %s\n"+
			"  Architecture: %s\n"+
			"  Source:       %s\n"+
			"  Trust State:  %s\n\n"+
			"%s\n"+
			"  Desktop Name: %s\n"+
			"  Exec line:    %s\n"+
			"  Categories:   %s\n\n"+
			"  File Action:  %s  [press 'm' to toggle move/copy]",
		themeStyles.Accent.Bold(true).Render("PRE-INSTALL INSPECTION REVIEW"),
		themeStyles.Text.Bold(true).Render(rev.Name),
		rev.Version,
		registry.FormatSize(rev.Size),
		archMatchStr,
		registry.Truncate(rev.SourceDisplay, 48),
		trustStr,
		themeStyles.Muted.Bold(true).Render("EMBEDDED DESKTOP FIELDS (SAFELY EXTRACTED):"),
		rev.DesktopName,
		registry.Truncate(rev.DesktopExec, 40),
		rev.DesktopCategory,
		themeStyles.KeyChip.Render(moveStr),
	)

	if len(rev.Warnings) > 0 {
		content += "\n\n" + themeStyles.Warning.Bold(true).Render("WARNINGS:")
		for _, w := range rev.Warnings {
			content += "\n  " + themeStyles.Warning.Render("! "+w)
		}
	}

	reviewBox := themeStyles.Border.Width(m.Width - 6).Render(
		content + "\n\n" +
			themeStyles.KeyChip.Render("[Enter]") + themeStyles.Muted.Render(" Proceed with install    ") +
			themeStyles.KeyChip.Render("[Esc]") + themeStyles.Muted.Render(" Cancel review"),
	)
	builder.WriteString("  " + reviewBox + "\n")
	return builder.String()
}

func (m Model) renderPipeline(themeStyles styles.Styles) string {
	var builder strings.Builder

	steps := []struct {
		id   installer.StepID
		name string
	}{
		{installer.StepFetch, "fetch"},
		{installer.StepVerify, "verify"},
		{installer.StepMakeExecutable, "make executable"},
		{installer.StepExtractIcon, "extract icon"},
		{installer.StepWriteDesktop, "write .desktop"},
		{installer.StepRefreshAppMenu, "refresh app menu"},
	}

	var stepRenders []string
	for idx, st := range steps {
		stepNum := idx + 1
		if stepNum < m.ActiveStepIdx || m.Mode == ModeCompleted {
			stepRenders = append(stepRenders, themeStyles.Success.Render("✓ "+st.name))
		} else if stepNum == m.ActiveStepIdx && m.Mode == ModeInstalling {
			arrow := "▶"
			if m.Config.ReducedMotion {
				arrow = "●"
			}
			stepRenders = append(stepRenders, themeStyles.Accent.Bold(true).Render(arrow+" "+st.name))
		} else {
			stepRenders = append(stepRenders, themeStyles.Muted.Render(st.name))
		}
	}

	pipelineRow := strings.Join(stepRenders, themeStyles.Muted.Render(" → "))

	pipelineBox := themeStyles.Border.Width(m.Width - 6).Render(
		themeStyles.Accent.Bold(true).Render("INSTALLATION PIPELINE") + "\n\n" +
			pipelineRow + "\n\n" +
			m.ProgressBar.View() + "\n\n" +
			themeStyles.Text.Render("  "+m.StepDetail),
	)
	builder.WriteString("  " + pipelineBox + "\n\n")

	if m.Mode == ModeCompleted && m.InstalledApp != nil {
		completeCard := themeStyles.Border.BorderForeground(themeStyles.Theme.Success).Width(m.Width - 6).Render(
			themeStyles.Success.Bold(true).Render(fmt.Sprintf("added ✓ %s", m.InstalledApp.Name)) + "\n\n" +
				themeStyles.Text.Render(fmt.Sprintf("Command: %s  •  Desktop: %s  •  Version: %s  •  Trust: %s",
					m.InstalledApp.ExecName, filepath.Base(m.InstalledApp.DesktopFile), m.InstalledApp.Version, m.InstalledApp.Trust)) + "\n\n" +
				themeStyles.KeyChip.Render("[l]") + themeStyles.Muted.Render(" launch now    ") +
				themeStyles.KeyChip.Render("[2]") + themeStyles.Muted.Render(" go to Apps    ") +
				themeStyles.KeyChip.Render("[Esc]") + themeStyles.Muted.Render(" add another"),
		)
		builder.WriteString("  " + completeCard + "\n")
	} else if m.Mode == ModeFailed {
		failCard := themeStyles.Border.BorderForeground(themeStyles.Theme.Error).Width(m.Width - 6).Render(
			themeStyles.Error.Bold(true).Render("add failed ✗") + "\n\n" +
				themeStyles.Text.Render(m.ErrorMessage) + "\n\n" +
				themeStyles.KeyChip.Render("[Esc]") + themeStyles.Muted.Render(" back to scanner"),
		)
		builder.WriteString("  " + failCard + "\n")
	}

	return builder.String()
}
