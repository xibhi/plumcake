package apps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xibhi/plumcake/internal/core/config"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/scanner"
	"github.com/xibhi/plumcake/internal/core/trash"
	"github.com/xibhi/plumcake/internal/core/updater"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeFilter
	ModeRename
	ModeChangeIcon
	ModeConfirmDelete
	ModeSandbox
	ModeVersions
)

func (m Mode) IsModal() bool {
	return m != ModeNormal
}

type Model struct {
	Paths          *paths.Paths
	Config         config.Config
	Apps           []registry.App
	Filtered       []registry.App
	Cursor         int
	Mode           Mode
	SearchInput    textinput.Model
	RenameInput    textinput.Model
	IconInput      textinput.Model
	IconValidMsg   string
	IconIsValid    bool
	SandboxToolIdx int
	SandboxNet     bool
	VersionCursor  int
	LastRemovedApp *registry.App
	LastRemoveTime time.Time
	StatusMessage  string
	StatusIsError  bool
	Width          int
	Height         int
}

type UpdateCheckDoneMsg struct {
	Checked      int
	UpdatesFound int
}

func New(pathsInstance *paths.Paths) Model {
	cfg := config.Load()

	searchInput := textinput.New()
	searchInput.Placeholder = "Filter applications..."
	searchInput.CharLimit = 64
	searchInput.Width = 32

	renameInput := textinput.New()
	renameInput.Placeholder = "Enter new display name..."
	renameInput.CharLimit = 64
	renameInput.Width = 36

	iconInput := textinput.New()
	iconInput.Placeholder = "/path/to/icon.png or .svg"
	iconInput.CharLimit = 256
	iconInput.Width = 44

	modelInstance := Model{
		Paths:       pathsInstance,
		Config:      cfg,
		Mode:        ModeNormal,
		SearchInput: searchInput,
		RenameInput: renameInput,
		IconInput:   iconInput,
	}
	modelInstance.Refresh()
	return modelInstance
}

func (m *Model) Refresh() {
	allApps, _ := registry.LoadAll(m.Paths)
	m.Apps = allApps
	m.ApplyFilter()
}

func (m *Model) ApplyFilter() {
	query := strings.TrimSpace(m.SearchInput.Value())
	if query == "" {
		m.Filtered = m.Apps
	} else {
		var matched []registry.App
		for _, a := range m.Apps {
			if scanner.FuzzyMatch(query, a.Name) || scanner.FuzzyMatch(query, a.ID) || scanner.FuzzyMatch(query, a.ExecName) {
				matched = append(matched, a)
			}
		}
		m.Filtered = matched
	}
	if m.Cursor >= len(m.Filtered) {
		m.Cursor = max(0, len(m.Filtered)-1)
	}
}

func (m *Model) SelectedApp() *registry.App {
	if len(m.Filtered) > 0 && m.Cursor < len(m.Filtered) {
		return &m.Filtered[m.Cursor]
	}
	return nil
}

func (m Model) CheckAppUpdates(all bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var checkedCount int
		var foundCount int

		for i := range m.Apps {
			if !all && i != m.Cursor {
				continue
			}
			hasUp, _, err := updater.Check(ctx, &m.Apps[i], m.Config.IncludePrereleases)
			if err == nil {
				checkedCount++
				if hasUp {
					foundCount++
					_ = registry.Save(m.Paths, &m.Apps[i])
				}
			}
		}
		return UpdateCheckDoneMsg{Checked: checkedCount, UpdatesFound: foundCount}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch event := msg.(type) {
	case UpdateCheckDoneMsg:
		m.Refresh()
		if event.UpdatesFound > 0 {
			m.StatusMessage = fmt.Sprintf("Checked %d app(s): %d update(s) available!", event.Checked, event.UpdatesFound)
			m.StatusIsError = false
		} else {
			m.StatusMessage = fmt.Sprintf("Checked %d app(s): all up to date.", event.Checked)
			m.StatusIsError = false
		}
		return m, nil

	case tea.KeyMsg:
		switch m.Mode {
		case ModeFilter:
			switch event.String() {
			case "enter", "esc":
				m.Mode = ModeNormal
				m.SearchInput.Blur()
				return m, nil
			}
			m.SearchInput, cmd = m.SearchInput.Update(msg)
			m.ApplyFilter()
			return m, cmd

		case ModeRename:
			switch event.String() {
			case "enter":
				app := m.SelectedApp()
				newName := strings.TrimSpace(m.RenameInput.Value())
				if app != nil && newName != "" {
					err := registry.Rename(m.Paths, app, newName)
					if err != nil {
						m.StatusMessage = "Rename failed: " + err.Error()
						m.StatusIsError = true
					} else {
						m.StatusMessage = fmt.Sprintf("Renamed to '%s'", newName)
						m.StatusIsError = false
						m.Refresh()
					}
				}
				m.Mode = ModeNormal
				m.RenameInput.Blur()
				return m, nil
			case "esc":
				m.Mode = ModeNormal
				m.RenameInput.Blur()
				return m, nil
			}
			m.RenameInput, cmd = m.RenameInput.Update(msg)
			return m, cmd

		case ModeChangeIcon:
			switch event.String() {
			case "enter":
				app := m.SelectedApp()
				iconPath := strings.TrimSpace(m.IconInput.Value())
				if app != nil && m.IconIsValid {
					err := registry.ChangeIcon(m.Paths, app, iconPath)
					if err != nil {
						m.StatusMessage = "Icon update failed: " + err.Error()
						m.StatusIsError = true
					} else {
						m.StatusMessage = "Icon updated successfully"
						m.StatusIsError = false
						m.Refresh()
					}
				}
				m.Mode = ModeNormal
				m.IconInput.Blur()
				return m, nil
			case "esc":
				m.Mode = ModeNormal
				m.IconInput.Blur()
				return m, nil
			}
			m.IconInput, cmd = m.IconInput.Update(msg)
			pathVal := strings.TrimSpace(m.IconInput.Value())
			if pathVal == "" {
				m.IconValidMsg = ""
				m.IconIsValid = false
			} else {
				isPNG, isSVG, err := registry.ValidateImageFile(pathVal)
				if err != nil {
					m.IconValidMsg = "✗ invalid file: not a readable PNG or SVG"
					m.IconIsValid = false
				} else if isPNG {
					m.IconValidMsg = "✓ valid PNG image"
					m.IconIsValid = true
				} else if isSVG {
					m.IconValidMsg = "✓ valid SVG vector image"
					m.IconIsValid = true
				}
			}
			return m, cmd

		case ModeSandbox:
			tools := []string{"off", "firejail", "bwrap"}
			switch event.String() {
			case "left", "h":
				if m.SandboxToolIdx > 0 {
					m.SandboxToolIdx--
				}
			case "right", "l":
				if m.SandboxToolIdx < len(tools)-1 {
					m.SandboxToolIdx++
				}
			case "n":
				m.SandboxNet = !m.SandboxNet
			case "enter":
				app := m.SelectedApp()
				if app != nil {
					toolChoice := tools[m.SandboxToolIdx]
					err := registry.ApplySandbox(m.Paths, app, toolChoice, m.SandboxNet)
					if err != nil {
						m.StatusMessage = "Sandbox failed: " + err.Error()
						m.StatusIsError = true
					} else {
						m.StatusMessage = fmt.Sprintf("Sandbox updated: %s (network: %t)", toolChoice, m.SandboxNet)
						m.StatusIsError = false
						m.Refresh()
					}
				}
				m.Mode = ModeNormal
				return m, nil
			case "esc":
				m.Mode = ModeNormal
				return m, nil
			}
			return m, nil

		case ModeVersions:
			app := m.SelectedApp()
			if app == nil || len(app.VersionHistory) == 0 {
				m.Mode = ModeNormal
				return m, nil
			}
			switch event.String() {
			case "up", "k":
				if m.VersionCursor > 0 {
					m.VersionCursor--
				}
			case "down", "j":
				if m.VersionCursor < len(app.VersionHistory)-1 {
					m.VersionCursor++
				}
			case "enter":
				targetVer := app.VersionHistory[m.VersionCursor].Version
				err := updater.Rollback(m.Paths, app, targetVer)
				if err != nil {
					m.StatusMessage = "Rollback error: " + err.Error()
					m.StatusIsError = true
				} else {
					m.StatusMessage = fmt.Sprintf("Switched to version %s", targetVer)
					m.StatusIsError = false
					m.Refresh()
				}
				m.Mode = ModeNormal
				return m, nil
			case "p":
				updater.Prune(app, 2)
				_ = registry.Save(m.Paths, app)
				m.StatusMessage = "Pruned older versions beyond retention"
				m.StatusIsError = false
				m.Refresh()
				return m, nil
			case "esc":
				m.Mode = ModeNormal
				return m, nil
			}
			return m, nil

		case ModeConfirmDelete:
			switch event.String() {
			case "y", "Y":
				app := m.SelectedApp()
				m.Mode = ModeNormal
				if app != nil {
					m.LastRemovedApp = app
					m.LastRemoveTime = time.Now()
					err := trash.Move(m.Paths, app, m.Config.TrashRetentionDays)
					if err != nil {
						m.StatusMessage = "Remove failed: " + err.Error()
						m.StatusIsError = true
					} else {
						m.StatusMessage = fmt.Sprintf("Moved '%s' to trash (press 'z' to undo)", app.Name)
						m.StatusIsError = false
						m.Refresh()
					}
				}
				return m, nil
			case "n", "N", "esc":
				m.Mode = ModeNormal
				m.StatusMessage = ""
				return m, nil
			}
			return m, nil

		case ModeNormal:
			switch event.String() {
			case "up", "k":
				if m.Cursor > 0 {
					m.Cursor--
				}
			case "down", "j":
				if m.Cursor < len(m.Filtered)-1 {
					m.Cursor++
				}
			case "/":
				m.Mode = ModeFilter
				m.SearchInput.Focus()
				return m, textinput.Blink
			case "r":
				app := m.SelectedApp()
				if app != nil {
					m.Mode = ModeRename
					m.RenameInput.SetValue(app.Name)
					m.RenameInput.CursorEnd()
					m.RenameInput.Focus()
					return m, textinput.Blink
				}
			case "i":
				app := m.SelectedApp()
				if app != nil {
					m.Mode = ModeChangeIcon
					m.IconInput.SetValue("")
					m.IconValidMsg = ""
					m.IconIsValid = false
					m.IconInput.Focus()
					return m, textinput.Blink
				}
			case "l", "enter":
				app := m.SelectedApp()
				if app != nil {
					err := registry.Launch(app)
					if err != nil {
						m.StatusMessage = "Launch error: " + err.Error()
						m.StatusIsError = true
					} else {
						m.StatusMessage = fmt.Sprintf("Launched %s", app.ExecName)
						m.StatusIsError = false
					}
				}
			case "u", "delete":
				app := m.SelectedApp()
				if app != nil {
					if m.Config.ConfirmDelete {
						m.Mode = ModeConfirmDelete
					} else {
						m.LastRemovedApp = app
						m.LastRemoveTime = time.Now()
						_ = trash.Move(m.Paths, app, m.Config.TrashRetentionDays)
						m.StatusMessage = fmt.Sprintf("Moved '%s' to trash (press 'z' to undo)", app.Name)
						m.Refresh()
					}
				}
			case "z":
				if m.LastRemovedApp != nil && time.Since(m.LastRemoveTime) < 60*time.Second {
					trashItems, _ := trash.List(m.Paths)
					for _, item := range trashItems {
						if item.AppManifest.ID == m.LastRemovedApp.ID {
							_ = trash.Restore(m.Paths, item.TrashID)
							m.StatusMessage = fmt.Sprintf("Restored '%s' from trash!", item.AppManifest.Name)
							m.LastRemovedApp = nil
							m.Refresh()
							break
						}
					}
				}
			case "o":
				app := m.SelectedApp()
				if app != nil {
					_ = registry.OpenContainingFolder(app)
					m.StatusMessage = "Opened containing directory"
					m.StatusIsError = false
				}
			case "s":
				app := m.SelectedApp()
				if app != nil {
					m.Mode = ModeSandbox
					tools := []string{"off", "firejail", "bwrap"}
					m.SandboxToolIdx = 0
					for idx, t := range tools {
						if strings.EqualFold(app.Sandbox.Tool, t) {
							m.SandboxToolIdx = idx
							break
						}
					}
					m.SandboxNet = app.Sandbox.Network
				}
			case "v":
				app := m.SelectedApp()
				if app != nil && len(app.VersionHistory) > 0 {
					m.Mode = ModeVersions
					m.VersionCursor = 0
				}
			case "U":
				m.StatusMessage = "Checking updates for all applications..."
				return m, m.CheckAppUpdates(true)
			}
		}
	}

	return m, nil
}
