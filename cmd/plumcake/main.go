package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xibhi/plumcake/internal/core/migrate"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/trash"
	"github.com/xibhi/plumcake/internal/tui"
)

const version = "1.0.1"

func main() {
	if len(os.Args) > 1 {
		argument := os.Args[1]
		if argument == "--version" || argument == "-v" {
			fmt.Printf("plumcake %s - A sweet way to run AppImages\n", version)
			os.Exit(0)
		}
		if argument == "--help" || argument == "-h" {
			fmt.Printf("plumcake %s - A sweet way to run AppImages\n\nUsage:\n  plumcake        Launch full-screen TUI\n  plumcake -v     Show version\n  plumcake -h     Show help\n", version)
			os.Exit(0)
		}

		fmt.Println("plumcake is a TUI. Run `plumcake` with no arguments.")
		os.Exit(1)
	}

	pathsInstance := paths.New()
	_ = migrate.RunMigrationIfNeeded(pathsInstance)
	trash.PurgeExpired(pathsInstance)
	appInstance := tui.New(pathsInstance)
	teaProgram := tea.NewProgram(appInstance, tea.WithAltScreen())

	if _, err := teaProgram.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "plumcake error: %v\n", err)
		os.Exit(1)
	}
}
