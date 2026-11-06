package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/trust"
)

func TestRollbackAndPrune(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		AppsDir: filepath.Join(tempDirectory, "apps"),
	}
	appDirectory := filepath.Join(pathsInstance.AppsDir, "demoapp")
	if err := os.MkdirAll(appDirectory, 0755); err != nil {
		t.Fatal(err)
	}

	activeBinary := filepath.Join(appDirectory, "demoapp.AppImage")
	if err := os.WriteFile(activeBinary, []byte("version 2 content"), 0755); err != nil {
		t.Fatal(err)
	}

	app := &registry.App{
		ID:           "demoapp",
		Name:         "Demo App",
		Version:      "2.0.0",
		AppImagePath: activeBinary,
		BinarySHA256: "v2hash",
		VersionHistory: []registry.VersionRecord{
			{
				Version: "2.0.0",
				Path:    activeBinary,
				Size:    17,
				SHA256:  "v2hash",
				Trust:   trust.Verified,
			},
		},
	}

	if err := ArchiveVersion(pathsInstance, app, 2); err != nil {
		t.Fatal(err)
	}

	v1Archive := filepath.Join(appDirectory, "versions", "demoapp-1.0.0.AppImage")
	_ = os.MkdirAll(filepath.Dir(v1Archive), 0755)
	if err := os.WriteFile(v1Archive, []byte("version 1 content"), 0755); err != nil {
		t.Fatal(err)
	}

	app.VersionHistory = append(app.VersionHistory, registry.VersionRecord{
		Version: "1.0.0",
		Path:    v1Archive,
		Size:    17,
		SHA256:  "v1hash",
		Trust:   trust.Verified,
	})

	if err := Rollback(pathsInstance, app, "1.0.0"); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	if app.Version != "1.0.0" {
		t.Fatalf("expected app version 1.0.0, got: %s", app.Version)
	}

	activeContent, err := os.ReadFile(activeBinary)
	if err != nil || string(activeContent) != "version 1 content" {
		t.Fatalf("active binary not restored to v1 content: %s", string(activeContent))
	}

	Prune(app, 1)
	if len(app.VersionHistory) > 1 {
		t.Fatalf("expected version history pruned to 1, got %d", len(app.VersionHistory))
	}
}
