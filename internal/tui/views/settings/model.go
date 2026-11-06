package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xibhi/plumcake/internal/core/config"
	"github.com/xibhi/plumcake/internal/core/doctor"
	"github.com/xibhi/plumcake/internal/core/history"
	"github.com/xibhi/plumcake/internal/core/manifest"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/trash"
	"github.com/xibhi/plumcake/internal/tui/theme"
)

type Section int

const (
	SectionPreferences Section = iota
	SectionDoctor
	SectionTrash
	SectionLog
	SectionPortability
)

type ThemePreviewMsg theme.Palette
type ThemeSavedMsg config.Config

type Model struct {
	Paths           *paths.Paths
	Config          config.Config
	ActiveSection   Section
	PrefCursor      int
	DoctorCursor    int
	TrashCursor     int
	LogCursor       int
	InThemePicker   bool
	OriginalThemeID string
	PickerCursor    int
	DoctorChecks    []doctor.Check
	TrashItems      []trash.Item
	Logs            []history.Entry
	CustomWarnings  []string
	ImportPlan      *manifest.ImportPlan
	ExportPathIn    textinput.Model
	ImportPathIn    textinput.Model
	InSubAction     bool
	StatusMessage   string
	StatusIsError   bool
	Width           int
	Height          int
}

func New(pathsInstance *paths.Paths) Model {
	cfg := config.Load()

	home, _ := os.UserHomeDir()
	exportInput := textinput.New()
	exportInput.SetValue(filepath.Join(home, "plumcake-manifest.toml"))
	exportInput.Width = 40

	importInput := textinput.New()
	importInput.Placeholder = "/path/to/plumcake-manifest.toml"
	importInput.Width = 40

	allThemes := theme.GetAll()
	pickerIndex := 0
	for i, t := range allThemes {
		if strings.EqualFold(t.ID, cfg.Theme) {
			pickerIndex = i
			break
		}
	}

	modelInstance := Model{
		Paths:         pathsInstance,
		Config:        cfg,
		PickerCursor:  pickerIndex,
		ActiveSection: SectionPreferences,
		ExportPathIn:  exportInput,
		ImportPathIn:  importInput,
	}
	modelInstance.Refresh()
	return modelInstance
}

func (m *Model) Refresh() {
	allApps, _ := registry.LoadAll(m.Paths)
	m.DoctorChecks = doctor.Run(m.Paths, allApps)
	m.TrashItems, _ = trash.List(m.Paths)
	m.Logs, _ = history.ReadRecent(m.Paths, 40)
	_, m.CustomWarnings = theme.Load("")
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch event := msg.(type) {
	case tea.KeyMsg:
		if m.InThemePicker {
			allThemes := theme.GetAll()
			switch event.String() {
			case "up", "k":
				if m.PickerCursor > 0 {
					m.PickerCursor--
					theme.Current = allThemes[m.PickerCursor]
					return m, func() tea.Msg { return ThemePreviewMsg(theme.Current) }
				}
			case "down", "j":
				if m.PickerCursor < len(allThemes)-1 {
					m.PickerCursor++
					theme.Current = allThemes[m.PickerCursor]
					return m, func() tea.Msg { return ThemePreviewMsg(theme.Current) }
				}
			case "enter":
				selectedTheme := allThemes[m.PickerCursor]
				m.Config.Theme = selectedTheme.ID
				_ = config.Save(m.Config)
				theme.Current = selectedTheme
				m.InThemePicker = false
				m.StatusMessage = "Saved theme: " + selectedTheme.Name
				m.StatusIsError = false
				return m, func() tea.Msg { return ThemeSavedMsg(m.Config) }
			case "esc":
				m.InThemePicker = false
				theme.Current = theme.Find(m.OriginalThemeID)
				for i, t := range allThemes {
					if t.ID == theme.Current.ID {
						m.PickerCursor = i
						break
					}
				}
				m.StatusMessage = "Theme change canceled"
				return m, func() tea.Msg { return ThemePreviewMsg(theme.Current) }
			}
			return m, nil
		}

		if m.InSubAction {
			switch event.String() {
			case "enter":
				if m.ActiveSection == SectionPortability {
					if m.ExportPathIn.Focused() {
						destinationPath := strings.TrimSpace(m.ExportPathIn.Value())
						err := manifest.Export(m.Paths, destinationPath)
						if err != nil {
							m.StatusMessage = "Export failed: " + err.Error()
							m.StatusIsError = true
						} else {
							m.StatusMessage = "Exported manifest to: " + destinationPath
							m.StatusIsError = false
						}
						m.InSubAction = false
						m.ExportPathIn.Blur()
						return m, nil
					} else if m.ImportPathIn.Focused() {
						sourcePath := strings.TrimSpace(m.ImportPathIn.Value())
						importPlan, err := manifest.Plan(m.Paths, sourcePath)
						if err != nil {
							m.StatusMessage = "Import parse failed: " + err.Error()
							m.StatusIsError = true
						} else {
							m.ImportPlan = importPlan
							m.StatusMessage = fmt.Sprintf("Import plan: %d to install, %d present", len(importPlan.WillInstall), len(importPlan.AlreadyPresent))
							m.StatusIsError = false
						}
						m.InSubAction = false
						m.ImportPathIn.Blur()
						return m, nil
					}
				}
			case "esc":
				m.InSubAction = false
				m.ExportPathIn.Blur()
				m.ImportPathIn.Blur()
				return m, nil
			}

			if m.ExportPathIn.Focused() {
				m.ExportPathIn, cmd = m.ExportPathIn.Update(msg)
				return m, cmd
			}
			if m.ImportPathIn.Focused() {
				m.ImportPathIn, cmd = m.ImportPathIn.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		switch event.String() {
		case "h", "left":
			if m.ActiveSection > 0 {
				m.ActiveSection--
			} else {
				m.ActiveSection = SectionPortability
			}
			return m, nil

		case "l", "right":
			m.ActiveSection = (m.ActiveSection + 1) % 5
			return m, nil

		case "up", "k":
			switch m.ActiveSection {
			case SectionPreferences:
				if m.PrefCursor > 0 {
					m.PrefCursor--
				}
			case SectionDoctor:
				if m.DoctorCursor > 0 {
					m.DoctorCursor--
				}
			case SectionTrash:
				if m.TrashCursor > 0 {
					m.TrashCursor--
				}
			case SectionLog:
				if m.LogCursor > 0 {
					m.LogCursor--
				}
			}

		case "down", "j":
			switch m.ActiveSection {
			case SectionPreferences:
				if m.PrefCursor < 9 {
					m.PrefCursor++
				}
			case SectionDoctor:
				if m.DoctorCursor < len(m.DoctorChecks)-1 {
					m.DoctorCursor++
				}
			case SectionTrash:
				if m.TrashCursor < len(m.TrashItems)-1 {
					m.TrashCursor++
				}
			case SectionLog:
				if m.LogCursor < len(m.Logs)-1 {
					m.LogCursor++
				}
			}

		case "enter", " ":
			switch m.ActiveSection {
			case SectionPreferences:
				switch m.PrefCursor {
				case 0:
					m.InThemePicker = true
					m.OriginalThemeID = m.Config.Theme
					allThemes := theme.GetAll()
					for i, t := range allThemes {
						if t.ID == m.Config.Theme {
							m.PickerCursor = i
							break
						}
					}
					return m, nil
				case 1:
					m.Config.PreinstallInspect = !m.Config.PreinstallInspect
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Pre-install review: %t", m.Config.PreinstallInspect)
				case 2:
					m.Config.ReducedMotion = !m.Config.ReducedMotion
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Reduced motion: %t", m.Config.ReducedMotion)
				case 3:
					m.Config.CheckUpdatesOnStart = !m.Config.CheckUpdatesOnStart
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Check updates on start: %t", m.Config.CheckUpdatesOnStart)
				case 4:
					m.Config.IncludePrereleases = !m.Config.IncludePrereleases
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Include GitHub prereleases: %t", m.Config.IncludePrereleases)
				case 5:
					m.Config.DeepScan = !m.Config.DeepScan
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Deep scan on start: %t", m.Config.DeepScan)
				case 6:
					m.Config.ConfirmDelete = !m.Config.ConfirmDelete
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Confirm before delete: %t", m.Config.ConfirmDelete)
				case 7:
					m.Config.DefaultMoveFile = !m.Config.DefaultMoveFile
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Move instead of copy: %t", m.Config.DefaultMoveFile)
				case 8:
					if m.Config.KeepVersions >= 5 {
						m.Config.KeepVersions = 1
					} else {
						m.Config.KeepVersions++
					}
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Versions kept: %d", m.Config.KeepVersions)
				case 9:
					if m.Config.TrashRetentionDays >= 30 {
						m.Config.TrashRetentionDays = 3
					} else {
						m.Config.TrashRetentionDays += 7
					}
					_ = config.Save(m.Config)
					m.StatusMessage = fmt.Sprintf("Trash retention: %d days", m.Config.TrashRetentionDays)
				}

			case SectionDoctor:
				if len(m.DoctorChecks) > 0 && m.DoctorCursor < len(m.DoctorChecks) {
					checkItem := m.DoctorChecks[m.DoctorCursor]
					if checkItem.CanFix {
						err := doctor.Fix(m.Paths, checkItem.ID)
						if err != nil {
							m.StatusMessage = "Fix failed: " + err.Error()
							m.StatusIsError = true
						} else {
							m.StatusMessage = "Repaired: " + checkItem.Title
							m.StatusIsError = false
							m.Refresh()
						}
					}
				}

			case SectionTrash:
				if len(m.TrashItems) > 0 && m.TrashCursor < len(m.TrashItems) {
					item := m.TrashItems[m.TrashCursor]
					err := trash.Restore(m.Paths, item.TrashID)
					if err != nil {
						m.StatusMessage = "Restore failed: " + err.Error()
						m.StatusIsError = true
					} else {
						m.StatusMessage = fmt.Sprintf("Restored %s from trash!", item.AppManifest.Name)
						m.StatusIsError = false
						m.Refresh()
					}
				}

			case SectionPortability:
				m.InSubAction = true
				m.ExportPathIn.Focus()
				return m, textinput.Blink
			}

		case "e":
			if m.ActiveSection == SectionTrash {
				_ = trash.Empty(m.Paths)
				m.StatusMessage = "Trash emptied"
				m.StatusIsError = false
				m.Refresh()
			} else if m.ActiveSection == SectionPortability {
				m.InSubAction = true
				m.ExportPathIn.Focus()
				return m, textinput.Blink
			}

		case "i":
			if m.ActiveSection == SectionPortability {
				m.InSubAction = true
				m.ImportPathIn.Focus()
				return m, textinput.Blink
			}

		case "r":
			m.Refresh()
			m.StatusMessage = "Refreshed"
			m.StatusIsError = false
		}
	}

	return m, nil
}
