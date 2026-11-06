package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/sandbox"
)

type Status string

const (
	Pass Status = "pass"
	Warn Status = "warn"
	Fail Status = "fail"
)

type Check struct {
	ID       string
	Title    string
	Status   Status
	Message  string
	Path     string
	CanFix   bool
	FixLabel string
}

func Run(pathsInstance *paths.Paths, apps []registry.App) []Check {
	var checks []Check

	appsDirStatus := Pass
	appsDirMsg := "Directory exists and is writable"
	canFixAppsDir := false
	if err := os.MkdirAll(pathsInstance.AppsDir, 0755); err != nil {
		appsDirStatus = Fail
		appsDirMsg = fmt.Sprintf("Cannot create/write: %v", err)
	} else {
		testFile := filepath.Join(pathsInstance.AppsDir, ".doctor_write_test")
		if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
			appsDirStatus = Fail
			appsDirMsg = "Directory is not writable"
		} else {
			_ = os.Remove(testFile)
		}
	}
	checks = append(checks, Check{
		ID:       "apps_dir",
		Title:    "Storage Directory",
		Status:   appsDirStatus,
		Message:  appsDirMsg,
		Path:     pathsInstance.AppsDir,
		CanFix:   canFixAppsDir,
		FixLabel: "Create directory",
	})

	pathEnv := os.Getenv("PATH")
	binStatus := Pass
	binMsg := "Directory found in $PATH"
	canFixBin := false
	if !strings.Contains(pathEnv, pathsInstance.BinDir) {
		binStatus = Warn
		binMsg = "Directory is not in current $PATH"
		canFixBin = true
	}
	checks = append(checks, Check{
		ID:       "bin_path",
		Title:    "User Binary PATH Integration",
		Status:   binStatus,
		Message:  binMsg,
		Path:     pathsInstance.BinDir,
		CanFix:   canFixBin,
		FixLabel: "Add to shell profile",
	})

	deskDirStatus := Pass
	deskDirMsg := "Applications and icon directories exist"
	canFixDesk := false
	if err := os.MkdirAll(pathsInstance.ApplicationsDir, 0755); err != nil {
		deskDirStatus = Warn
		deskDirMsg = "Failed creating applications directory"
		canFixDesk = true
	}
	_ = os.MkdirAll(pathsInstance.IconsDir, 0755)
	_ = os.MkdirAll(pathsInstance.PixmapsDir, 0755)

	checks = append(checks, Check{
		ID:       "desktop_dirs",
		Title:    "Freedesktop Application & Icon Directories",
		Status:   deskDirStatus,
		Message:  deskDirMsg,
		Path:     pathsInstance.ApplicationsDir,
		CanFix:   canFixDesk,
		FixLabel: "Create directories",
	})

	fuseStatus := Pass
	fuseMsg := "FUSE device available (/dev/fuse)"
	if _, err := os.Stat("/dev/fuse"); os.IsNotExist(err) {
		fuseStatus = Warn
		fuseMsg = "/dev/fuse not found (AppImages run via --appimage-extract fallback)"
	}
	checks = append(checks, Check{
		ID:      "fuse_status",
		Title:   "FUSE Subsystem Availability",
		Status:  fuseStatus,
		Message: fuseMsg,
		Path:    "/dev/fuse",
		CanFix:  false,
	})

	toolStatus := Pass
	var toolParts []string
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		toolParts = append(toolParts, "update-desktop-database")
	}
	if _, err := exec.LookPath("gtk-update-icon-cache"); err == nil {
		toolParts = append(toolParts, "gtk-update-icon-cache")
	}
	if len(toolParts) == 0 {
		toolStatus = Warn
		toolParts = append(toolParts, "No tools found (system uses standard filesystem polling)")
	}
	checks = append(checks, Check{
		ID:      "desktop_tools",
		Title:   "System Desktop Integration Tools",
		Status:  toolStatus,
		Message: strings.Join(toolParts, ", "),
		Path:    "/usr/bin",
		CanFix:  false,
	})

	orphanCount := 0
	if entries, err := os.ReadDir(pathsInstance.ApplicationsDir); err == nil {
		for _, e := range entries {
			if (strings.HasPrefix(e.Name(), "plumcake-") || strings.HasPrefix(e.Name(), "adopt-")) && strings.HasSuffix(e.Name(), ".desktop") {
				found := false
				for _, app := range apps {
					if filepath.Base(app.DesktopFile) == e.Name() {
						if registry.FileExists(app.AppImagePath) {
							found = true
						}
						break
					}
				}
				if !found {
					orphanCount++
				}
			}
		}
	}
	orphanStatus := Pass
	orphanMsg := "No orphaned desktop entries found"
	canFixOrphan := false
	if orphanCount > 0 {
		orphanStatus = Warn
		orphanMsg = fmt.Sprintf("Found %d orphaned desktop file(s) without matching AppImage", orphanCount)
		canFixOrphan = true
	}
	checks = append(checks, Check{
		ID:       "orphaned_desktop",
		Title:    "Orphaned Desktop Entries",
		Status:   orphanStatus,
		Message:  orphanMsg,
		Path:     pathsInstance.ApplicationsDir,
		CanFix:   canFixOrphan,
		FixLabel: "Clean orphaned files",
	})

	brokenCount := 0
	for _, app := range apps {
		if !registry.FileExists(app.AppImagePath) || !registry.FileExists(app.WrapperFile) {
			brokenCount++
		}
	}
	brokenStatus := Pass
	brokenMsg := fmt.Sprintf("%d app(s) fully operational", len(apps))
	canFixBroken := false
	if brokenCount > 0 {
		brokenStatus = Fail
		brokenMsg = fmt.Sprintf("%d app(s) have missing binaries or wrappers", brokenCount)
		canFixBroken = true
	}
	checks = append(checks, Check{
		ID:       "broken_apps",
		Title:    "Applications Integrity",
		Status:   brokenStatus,
		Message:  brokenMsg,
		Path:     pathsInstance.AppsDir,
		CanFix:   canFixBroken,
		FixLabel: "Repair broken integrations",
	})

	sandboxTools := sandbox.Detect()
	sbStatus := Pass
	sbMsg := "Sandboxing engines available (firejail, bwrap)"
	if !sandboxTools.HasFirejail && !sandboxTools.HasBwrap {
		sbStatus = Warn
		sbMsg = "Neither firejail nor bwrap detected; optional app sandboxing unavailable"
	} else if !sandboxTools.HasFirejail {
		sbMsg = "Bubblewrap (bwrap) available; firejail not installed"
	} else if !sandboxTools.HasBwrap {
		sbMsg = "Firejail available; bwrap not installed"
	}
	checks = append(checks, Check{
		ID:      "sandbox_tools",
		Title:   "Application Sandboxing Engines",
		Status:  sbStatus,
		Message: sbMsg,
		Path:    "/usr/bin",
		CanFix:  false,
	})

	return checks
}

func Fix(pathsInstance *paths.Paths, checkID string) error {
	switch checkID {
	case "apps_dir":
		return os.MkdirAll(pathsInstance.AppsDir, 0755)

	case "desktop_dirs":
		_ = os.MkdirAll(pathsInstance.ApplicationsDir, 0755)
		_ = os.MkdirAll(pathsInstance.IconsDir, 0755)
		_ = os.MkdirAll(pathsInstance.PixmapsDir, 0755)
		return nil

	case "bin_path":
		home, _ := os.UserHomeDir()
		profile := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(profile); os.IsNotExist(err) {
			profile = filepath.Join(home, ".profile")
		}
		data, _ := os.ReadFile(profile)
		exportLine := fmt.Sprintf("\nexport PATH=\"%s:$PATH\"\n", pathsInstance.BinDir)
		if !strings.Contains(string(data), pathsInstance.BinDir) {
			fileDescriptor, err := os.OpenFile(profile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer fileDescriptor.Close()
			_, _ = fileDescriptor.WriteString(exportLine)
		}
		return nil

	case "orphaned_desktop":
		apps, _ := registry.LoadAll(pathsInstance)
		if entries, err := os.ReadDir(pathsInstance.ApplicationsDir); err == nil {
			for _, e := range entries {
				if (strings.HasPrefix(e.Name(), "plumcake-") || strings.HasPrefix(e.Name(), "adopt-")) && strings.HasSuffix(e.Name(), ".desktop") {
					desktopPath := filepath.Join(pathsInstance.ApplicationsDir, e.Name())
					found := false
					for _, app := range apps {
						if filepath.Base(app.DesktopFile) == e.Name() && registry.FileExists(app.AppImagePath) {
							found = true
							break
						}
					}
					if !found {
						_ = os.Remove(desktopPath)
					}
				}
			}
		}
		registry.RefreshDesktopDatabase(pathsInstance)
		return nil

	case "broken_apps":
		apps, _ := registry.LoadAll(pathsInstance)
		for _, app := range apps {
			if registry.FileExists(app.AppImagePath) && !registry.FileExists(app.WrapperFile) && app.WrapperFile != "" {
				_ = os.MkdirAll(filepath.Dir(app.WrapperFile), 0755)
				wrapperScript := fmt.Sprintf("#!/usr/bin/env bash\nexec \"%s\" \"$@\"\n", app.AppImagePath)
				_ = os.WriteFile(app.WrapperFile, []byte(wrapperScript), 0755)
			}
		}
		return nil
	}

	return nil
}
