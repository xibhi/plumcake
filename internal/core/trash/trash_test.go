package trash

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func TestTrashRetentionAndRestoreConflict(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		PlumcakeHome:    tempDirectory,
		AppsDir:         filepath.Join(tempDirectory, "apps"),
		ApplicationsDir: filepath.Join(tempDirectory, "applications"),
		IconsDir:        filepath.Join(tempDirectory, "icons"),
		PixmapsDir:      filepath.Join(tempDirectory, "pixmaps"),
		BinDir:          filepath.Join(tempDirectory, "bin"),
	}

	appDirectory := filepath.Join(pathsInstance.AppsDir, "trashtest")
	_ = os.MkdirAll(appDirectory, 0755)

	binaryPath := filepath.Join(appDirectory, "trashtest.AppImage")
	_ = os.WriteFile(binaryPath, []byte("binary data"), 0755)

	app := registry.App{
		ID:           "trashtest",
		Name:         "Trash Test",
		Version:      "1.0.0",
		AppImagePath: binaryPath,
	}

	if err := Move(pathsInstance, &app, 7); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if registry.FileExists(binaryPath) {
		t.Fatalf("expected binary removed from apps dir after trash")
	}

	items, err := List(pathsInstance)
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 item in trash, got %d, err=%v", len(items), err)
	}
	if items[0].AppManifest.ID != "trashtest" {
		t.Fatalf("unexpected trash item: %+v", items[0])
	}

	_ = os.MkdirAll(appDirectory, 0755)
	errConflict := Restore(pathsInstance, items[0].TrashID)
	if errConflict == nil {
		t.Fatalf("expected restore conflict error when app already exists")
	}

	_ = os.RemoveAll(appDirectory)

	if err := Restore(pathsInstance, items[0].TrashID); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	if !registry.FileExists(binaryPath) {
		t.Fatalf("expected binary restored to apps dir")
	}

	_ = Move(pathsInstance, &app, -1)
	PurgeExpired(pathsInstance)
	remaining, _ := List(pathsInstance)
	for _, item := range remaining {
		if item.AppManifest.ID == "trashtest" {
			expiration, _ := time.Parse(time.RFC3339, item.ExpiresAt)
			if time.Now().After(expiration) {
				t.Fatalf("expected expired item purged from trash")
			}
		}
	}
}
