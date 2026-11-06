package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func TestFreshInstallNoMigration(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		Home:            tempDirectory,
		DataHome:        filepath.Join(tempDirectory, "share"),
		ConfigHome:      filepath.Join(tempDirectory, "config"),
		StateHome:       filepath.Join(tempDirectory, "state"),
		PlumcakeHome:    filepath.Join(tempDirectory, "share", "plumcake"),
		AppsDir:         filepath.Join(tempDirectory, "share", "plumcake", "apps"),
		ApplicationsDir: filepath.Join(tempDirectory, "share", "applications"),
		IconsDir:        filepath.Join(tempDirectory, "share", "icons"),
		PixmapsDir:      filepath.Join(tempDirectory, "share", "pixmaps"),
		BinDir:          filepath.Join(tempDirectory, "bin"),
		OldAdoptHome:    filepath.Join(tempDirectory, "share", "adopt"),
		OldAdoptConfig:  filepath.Join(tempDirectory, "config", "adopt"),
		OldAdoptState:   filepath.Join(tempDirectory, "state", "adopt"),
	}

	err := RunMigrationIfNeeded(pathsInstance)
	if err != nil {
		t.Fatalf("expected nil error on fresh install: %v", err)
	}

	if _, err := os.Stat(pathsInstance.OldAdoptHome); !os.IsNotExist(err) {
		t.Errorf("old adopt home should not exist on fresh install")
	}
}

func TestMigrationFromOldAdoptDirectories(t *testing.T) {
	tempDirectory := t.TempDir()
	shareDir := filepath.Join(tempDirectory, "share")
	configDir := filepath.Join(tempDirectory, "config")
	stateDir := filepath.Join(tempDirectory, "state")

	oldAdoptData := filepath.Join(shareDir, "adopt")
	oldAdoptApps := filepath.Join(oldAdoptData, "apps", "demoapp")
	_ = os.MkdirAll(oldAdoptApps, 0755)

	oldBinaryPath := filepath.Join(oldAdoptApps, "demoapp.AppImage")
	_ = os.WriteFile(oldBinaryPath, []byte("fake elf"), 0755)

	oldDesktopDir := filepath.Join(shareDir, "applications")
	_ = os.MkdirAll(oldDesktopDir, 0755)
	oldDesktopFile := filepath.Join(oldDesktopDir, "adopt-demoapp.desktop")
	desktopBody := "[Desktop Entry]\nName=Demo App\nExec=/usr/bin/demo\nIcon=adopt-demoapp\nX-Adopt-Managed=true\nX-Adopt-ID=demoapp\n"
	_ = os.WriteFile(oldDesktopFile, []byte(desktopBody), 0644)

	oldConfigDir := filepath.Join(configDir, "adopt")
	_ = os.MkdirAll(oldConfigDir, 0755)
	_ = os.WriteFile(filepath.Join(oldConfigDir, "config.toml"), []byte("theme = \"nord\"\n"), 0644)

	pathsInstance := &paths.Paths{
		Home:            tempDirectory,
		DataHome:        shareDir,
		ConfigHome:      configDir,
		StateHome:       stateDir,
		PlumcakeHome:    filepath.Join(shareDir, "plumcake"),
		AppsDir:         filepath.Join(shareDir, "plumcake", "apps"),
		ApplicationsDir: oldDesktopDir,
		IconsDir:        filepath.Join(shareDir, "icons"),
		PixmapsDir:      filepath.Join(shareDir, "pixmaps"),
		BinDir:          filepath.Join(tempDirectory, "bin"),
		OldAdoptHome:    oldAdoptData,
		OldAdoptConfig:  oldConfigDir,
		OldAdoptState:   filepath.Join(stateDir, "adopt"),
	}

	appManifest := registry.App{
		SchemaVersion:    2,
		ID:               "demoapp",
		Name:             "Demo App",
		Version:          "1.0.0",
		AppImagePath:     oldBinaryPath,
		DesktopFile:      oldDesktopFile,
		OriginalFilename: "demoapp.AppImage",
	}
	_ = registry.Save(&paths.Paths{AppsDir: filepath.Join(oldAdoptData, "apps")}, &appManifest)

	err := RunMigrationIfNeeded(pathsInstance)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	newConfigPath := filepath.Join(configDir, "plumcake", "config.toml")
	if _, err := os.Stat(newConfigPath); os.IsNotExist(err) {
		t.Errorf("expected config migrated to plumcake/config.toml")
	}

	newAppsDir := filepath.Join(shareDir, "plumcake", "apps", "demoapp")
	if _, err := os.Stat(newAppsDir); os.IsNotExist(err) {
		t.Errorf("expected apps migrated to plumcake/apps/demoapp")
	}

	newDesktopFile := filepath.Join(oldDesktopDir, "plumcake-demoapp.desktop")
	if _, err := os.Stat(newDesktopFile); os.IsNotExist(err) {
		t.Errorf("expected desktop file renamed to plumcake-demoapp.desktop")
	}

	deskBytes, _ := os.ReadFile(newDesktopFile)
	deskStr := string(deskBytes)
	if !strings.Contains(deskStr, "X-Plumcake-Managed") {
		t.Errorf("expected X-Plumcake-Managed in migrated desktop file: %s", deskStr)
	}
	if !strings.Contains(deskStr, "Icon=plumcake-") {
		t.Errorf("expected Icon=plumcake- in migrated desktop file: %s", deskStr)
	}

	apps, err := registry.LoadAll(pathsInstance)
	if err != nil || len(apps) != 1 {
		t.Fatalf("expected 1 app loaded from plumcake registry, got %d, err=%v", len(apps), err)
	}
	if !strings.Contains(apps[0].AppImagePath, "plumcake/apps") {
		t.Errorf("expected AppImagePath updated to plumcake/apps: %s", apps[0].AppImagePath)
	}
	if !strings.HasSuffix(apps[0].DesktopFile, "plumcake-demoapp.desktop") {
		t.Errorf("expected DesktopFile updated: %s", apps[0].DesktopFile)
	}
}

func TestMigrationBothDirectoriesPresent(t *testing.T) {
	tempDirectory := t.TempDir()
	shareDir := filepath.Join(tempDirectory, "share")

	oldAdoptData := filepath.Join(shareDir, "adopt")
	_ = os.MkdirAll(filepath.Join(oldAdoptData, "apps", "oldapp"), 0755)

	newPlumcakeData := filepath.Join(shareDir, "plumcake")
	_ = os.MkdirAll(filepath.Join(newPlumcakeData, "apps", "newapp"), 0755)

	pathsInstance := &paths.Paths{
		Home:            tempDirectory,
		DataHome:        shareDir,
		ConfigHome:      filepath.Join(tempDirectory, "config"),
		StateHome:       filepath.Join(tempDirectory, "state"),
		PlumcakeHome:    newPlumcakeData,
		AppsDir:         filepath.Join(newPlumcakeData, "apps"),
		ApplicationsDir: filepath.Join(shareDir, "applications"),
		IconsDir:        filepath.Join(shareDir, "icons"),
		PixmapsDir:      filepath.Join(shareDir, "pixmaps"),
		BinDir:          filepath.Join(tempDirectory, "bin"),
		OldAdoptHome:    oldAdoptData,
	}

	err := RunMigrationIfNeeded(pathsInstance)
	if err != nil {
		t.Fatalf("expected no crash when both directories exist: %v", err)
	}

	if _, err := os.Stat(oldAdoptData); os.IsNotExist(err) {
		t.Errorf("old adopt directory must not be overwritten when plumcake already exists")
	}
}
