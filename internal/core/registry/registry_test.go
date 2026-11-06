package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
)

func TestRegistryLifecycle(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		AppsDir:         filepath.Join(tempDirectory, "apps"),
		ApplicationsDir: filepath.Join(tempDirectory, "applications"),
		IconsDir:        filepath.Join(tempDirectory, "icons"),
		PixmapsDir:      filepath.Join(tempDirectory, "pixmaps"),
	}

	app := &App{
		SchemaVersion: 1,
		ID:            "testapp",
		Name:          "Initial Name",
		Version:       "1.0.0",
		BinarySHA256:  "sha256abc",
		AppImagePath:  filepath.Join(tempDirectory, "test.AppImage"),
		DesktopFile:   filepath.Join(pathsInstance.ApplicationsDir, "plumcake-testapp.desktop"),
	}

	if !Migrate(app) {
		t.Fatalf("expected schema migration to return true for v1")
	}
	if app.SchemaVersion != 2 {
		t.Fatalf("expected schema version 2 after migration")
	}

	if err := Save(pathsInstance, app); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	allApps, err := LoadAll(pathsInstance)
	if err != nil || len(allApps) != 1 {
		t.Fatalf("expected 1 app loaded, got %d, err=%v", len(allApps), err)
	}

	_ = os.MkdirAll(pathsInstance.ApplicationsDir, 0755)
	desktopContent := "[Desktop Entry]\nName=Initial Name\nExec=testapp\n"
	_ = os.WriteFile(app.DesktopFile, []byte(desktopContent), 0644)

	if err := Rename(pathsInstance, app, "Updated Name"); err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if app.Name != "Updated Name" {
		t.Fatalf("expected Updated Name, got %s", app.Name)
	}

	data, _ := os.ReadFile(app.DesktopFile)
	if string(data) != "[Desktop Entry]\nName=Updated Name\nExec=testapp\n" {
		t.Fatalf("desktop file content mismatch: %s", string(data))
	}
}

func TestImageValidation(t *testing.T) {
	tempDirectory := t.TempDir()

	pngPath := filepath.Join(tempDirectory, "icon.png")
	pngHeader := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 20)...)
	_ = os.WriteFile(pngPath, pngHeader, 0644)

	isPNG, isSVG, err := ValidateImageFile(pngPath)
	if err != nil || !isPNG || isSVG {
		t.Fatalf("expected valid PNG, got isPNG=%t, isSVG=%t, err=%v", isPNG, isSVG, err)
	}

	svgPath := filepath.Join(tempDirectory, "icon.svg")
	svgContent := []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>")
	_ = os.WriteFile(svgPath, svgContent, 0644)

	isPNG, isSVG, err = ValidateImageFile(svgPath)
	if err != nil || isPNG || !isSVG {
		t.Fatalf("expected valid SVG, got isPNG=%t, isSVG=%t, err=%v", isPNG, isSVG, err)
	}

	badPath := filepath.Join(tempDirectory, "bad.txt")
	_ = os.WriteFile(badPath, []byte("random text"), 0644)
	isPNG, isSVG, err = ValidateImageFile(badPath)
	if err == nil {
		t.Fatalf("expected error for non-image file")
	}
}
