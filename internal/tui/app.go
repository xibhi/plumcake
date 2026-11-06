package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xibhi/plumcake/internal/core/config"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/tui/components/help"
	"github.com/xibhi/plumcake/internal/tui/components/styles"
	"github.com/xibhi/plumcake/internal/tui/theme"
	"github.com/xibhi/plumcake/internal/tui/views/add"
	"github.com/xibhi/plumcake/internal/tui/views/apps"
	"github.com/xibhi/plumcake/internal/tui/views/settings"
)

type Tab int

const (
	TabAdd Tab = iota
	TabApps
	TabSettings
)

type App struct {
	Paths        *paths.Paths
	Config       config.Config
	ActiveTab    Tab
	Styles       styles.Styles
	AddView      add.Model
	AppsView     apps.Model
	SettingsView settings.Model
	ShowHelp     bool
	Width        int
	Height       int
}

func New(pathsInstance *paths.Paths) App {
	cfg := config.Load()
	currentTheme := theme.Find(cfg.Theme)
	theme.Current = currentTheme
	themeStyles := styles.New(currentTheme)

	appInstance := App{
		Paths:        pathsInstance,
		Config:       cfg,
		ActiveTab:    TabAdd,
		Styles:       themeStyles,
		AddView:      add.New(pathsInstance, cfg),
		AppsView:     apps.New(pathsInstance),
		SettingsView: settings.New(pathsInstance),
		Width:        100,
		Height:       28,
	}
	appInstance.resizeSubmodels()
	return appInstance
}

func (a *App) resizeSubmodels() {
	a.AddView.Width = a.Width
	a.AddView.Height = a.Height
	a.AppsView.Width = a.Width
	a.AppsView.Height = a.Height
	a.SettingsView.Width = a.Width
	a.SettingsView.Height = a.Height
}

func (a App) Init() tea.Cmd {
	return a.AddView.StartInitialScan()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd

	switch event := msg.(type) {
	case tea.WindowSizeMsg:
		a.Width = max(80, event.Width)
		a.Height = max(24, event.Height)
		a.resizeSubmodels()

	case settings.ThemePreviewMsg:
		theme.Current = theme.Palette(event)
		a.Styles = styles.New(theme.Current)

	case settings.ThemeSavedMsg:
		a.Config = config.Config(event)
		theme.Current = theme.Find(a.Config.Theme)
		a.Styles = styles.New(theme.Current)

	case tea.KeyMsg:
		if event.String() == "?" && !a.isInputActive() {
			a.ShowHelp = !a.ShowHelp
			return a, nil
		}

		if a.ShowHelp {
			if event.String() == "esc" || event.String() == "?" || event.String() == "q" {
				a.ShowHelp = false
				return a, nil
			}
			return a, nil
		}

		if !a.isInputActive() && (event.String() == "ctrl+c" || (event.String() == "q" && !a.AppsView.Mode.IsModal())) {
			return a, tea.Quit
		}

		isWriting := a.isTextInputWriting()
		if !a.isBusy() {
			switch event.String() {
			case "tab":
				if !isWriting {
					a.ActiveTab = (a.ActiveTab + 1) % 3
					a.refreshActiveTab()
					return a, nil
				}
			case "shift+tab":
				if a.ActiveTab == 0 {
					a.ActiveTab = TabSettings
				} else {
					a.ActiveTab--
				}
				a.refreshActiveTab()
				return a, nil
			case "1":
				if !isWriting {
					a.ActiveTab = TabAdd
					a.refreshActiveTab()
					return a, nil
				}
			case "2":
				if !isWriting {
					a.ActiveTab = TabApps
					a.refreshActiveTab()
					return a, nil
				}
			case "3":
				if !isWriting {
					a.ActiveTab = TabSettings
					a.refreshActiveTab()
					return a, nil
				}
			}
		}
	}

	switch a.ActiveTab {
	case TabAdd:
		var cmd tea.Cmd
		a.AddView, cmd = a.AddView.Update(msg)
		if cmd != nil {
			commands = append(commands, cmd)
		}

	case TabApps:
		var cmd tea.Cmd
		a.AppsView, cmd = a.AppsView.Update(msg)
		if cmd != nil {
			commands = append(commands, cmd)
		}

	case TabSettings:
		var cmd tea.Cmd
		a.SettingsView, cmd = a.SettingsView.Update(msg)
		if cmd != nil {
			commands = append(commands, cmd)
		}
	}

	return a, tea.Batch(commands...)
}

func (a *App) refreshActiveTab() {
	switch a.ActiveTab {
	case TabAdd:
		a.AddView.ApplyFilter()
	case TabApps:
		a.AppsView.Refresh()
	case TabSettings:
		a.SettingsView.Refresh()
	}
}

func (a *App) isInputActive() bool {
	if a.ActiveTab == TabAdd && a.AddView.Input.Focused() {
		return true
	}
	if a.ActiveTab == TabApps && a.AppsView.Mode.IsModal() {
		return true
	}
	return false
}

func (a *App) isTextInputWriting() bool {
	if a.ActiveTab == TabAdd && a.AddView.Input.Focused() {
		return true
	}
	if a.ActiveTab == TabApps && (a.AppsView.Mode == apps.ModeFilter || a.AppsView.Mode == apps.ModeRename || a.AppsView.Mode == apps.ModeChangeIcon) {
		return true
	}
	if a.ActiveTab == TabSettings && a.SettingsView.InSubAction {
		return true
	}
	return false
}

func (a *App) isBusy() bool {
	return a.AddView.Mode == add.ModeInstalling || a.SettingsView.DoctorChecks == nil
}

func (a App) View() string {
	var builder strings.Builder

	wordmark := a.Styles.Wordmark.Render("plumcake")
	spacer := "   "

	tab1 := a.Styles.TabPassive.Render("Add")
	tab2 := a.Styles.TabPassive.Render("Apps")
	tab3 := a.Styles.TabPassive.Render("Settings")

	switch a.ActiveTab {
	case TabAdd:
		tab1 = a.Styles.TabActive.Render("Add")
	case TabApps:
		tab2 = a.Styles.TabActive.Render("Apps")
	case TabSettings:
		tab3 = a.Styles.TabActive.Render("Settings")
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, "  ", wordmark, spacer, tab1, spacer, tab2, spacer, tab3)
	builder.WriteString("\n" + tabBar + "\n\n")

	switch a.ActiveTab {
	case TabAdd:
		builder.WriteString(a.AddView.View(a.Styles))
	case TabApps:
		builder.WriteString(a.AppsView.View(a.Styles))
	case TabSettings:
		builder.WriteString(a.SettingsView.View(a.Styles))
	}

	builder.WriteString("\n")
	var hints string
	switch a.ActiveTab {
	case TabAdd:
		if a.AddView.Mode == add.ModeBrowsing {
			hints = fmt.Sprintf("  %s enter    %s type link or path    %s scan deeper    %s help    %s quit",
				a.Styles.KeyChip.Render("install:"),
				a.Styles.Muted.Render("/:"),
				a.Styles.Muted.Render("F5:"),
				a.Styles.Muted.Render("?:"),
				a.Styles.Muted.Render("q:"),
			)
		} else {
			hints = fmt.Sprintf("  %s launch now    %s apps screen    %s back to scanner",
				a.Styles.KeyChip.Render("l:"),
				a.Styles.KeyChip.Render("2:"),
				a.Styles.KeyChip.Render("Esc:"),
			)
		}

	case TabApps:
		hints = fmt.Sprintf("  %s rename    %s change icon    %s launch    %s remove    %s open folder    %s search    %s help",
			a.Styles.KeyChip.Render("r:"),
			a.Styles.KeyChip.Render("i:"),
			a.Styles.KeyChip.Render("l:"),
			a.Styles.KeyChip.Render("u:"),
			a.Styles.KeyChip.Render("o:"),
			a.Styles.KeyChip.Render("/:"),
			a.Styles.KeyChip.Render("?:"),
		)

	case TabSettings:
		hints = fmt.Sprintf("  %s switch pane    %s select    %s toggle / fix    %s refresh    %s help",
			a.Styles.KeyChip.Render("h/l:"),
			a.Styles.KeyChip.Render("j/k:"),
			a.Styles.KeyChip.Render("Enter:"),
			a.Styles.KeyChip.Render("r:"),
			a.Styles.KeyChip.Render("?:"),
		)
	}

	footerBar := a.Styles.Footer.Width(a.Width - 4).Render(hints)
	builder.WriteString("  " + footerBar + "\n")

	if a.ShowHelp {
		return help.Render(a.Styles, a.Width, a.Height)
	}

	return builder.String()
}
