package doctor

import (
	"path/filepath"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func TestDoctorChecksAndFixes(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		Home:            tempDirectory,
		AppsDir:         filepath.Join(tempDirectory, "apps"),
		ApplicationsDir: filepath.Join(tempDirectory, "applications"),
		IconsDir:        filepath.Join(tempDirectory, "icons"),
		PixmapsDir:      filepath.Join(tempDirectory, "pixmaps"),
		BinDir:          filepath.Join(tempDirectory, "bin"),
	}

	checks := Run(pathsInstance, nil)
	if len(checks) < 7 {
		t.Fatalf("expected at least 7 doctor checks, got %d", len(checks))
	}

	if err := Fix(pathsInstance, "apps_dir"); err != nil {
		t.Fatalf("fix apps_dir failed: %v", err)
	}
	if err := Fix(pathsInstance, "desktop_dirs"); err != nil {
		t.Fatalf("fix desktop_dirs failed: %v", err)
	}

	checksAfter := Run(pathsInstance, nil)
	for _, checkItem := range checksAfter {
		if checkItem.ID == "apps_dir" && checkItem.Status != Pass {
			t.Errorf("expected apps_dir to pass after fix, got %s", checkItem.Status)
		}
		if checkItem.ID == "desktop_dirs" && checkItem.Status != Pass {
			t.Errorf("expected desktop_dirs to pass after fix, got %s", checkItem.Status)
		}
	}

	dummyApp := registry.App{
		ID:           "broken",
		Name:         "Broken App",
		AppImagePath: filepath.Join(tempDirectory, "nonexistent.AppImage"),
		WrapperFile:  filepath.Join(pathsInstance.BinDir, "broken"),
	}
	brokenChecks := Run(pathsInstance, []registry.App{dummyApp})
	foundBroken := false
	for _, c := range brokenChecks {
		if c.ID == "broken_apps" && c.Status == Fail {
			foundBroken = true
			break
		}
	}
	if !foundBroken {
		t.Errorf("expected broken_apps to fail for missing binary")
	}
}
