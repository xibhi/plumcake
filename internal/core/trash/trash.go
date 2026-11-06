package trash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xibhi/plumcake/internal/core/history"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

type Item struct {
	TrashID     string       `json:"trash_id"`
	DeletedAt   string       `json:"deleted_at"`
	ExpiresAt   string       `json:"expires_at"`
	AppManifest registry.App `json:"app_manifest"`
	TrashPath   string       `json:"trash_path"`
}

func Directory(pathsInstance *paths.Paths) string {
	return filepath.Join(pathsInstance.PlumcakeHome, "trash")
}

func Move(pathsInstance *paths.Paths, app *registry.App, retentionDays int) error {
	trashRoot := Directory(pathsInstance)
	now := time.Now()
	trashIdentifier := fmt.Sprintf("%s-%d", app.ID, now.Unix())
	itemDirectory := filepath.Join(trashRoot, trashIdentifier)

	if err := os.MkdirAll(itemDirectory, 0755); err != nil {
		return err
	}

	filesDirectory := filepath.Join(itemDirectory, "files")
	_ = os.MkdirAll(filesDirectory, 0755)

	if registry.FileExists(app.AppImagePath) {
		destBin := filepath.Join(filesDirectory, filepath.Base(app.AppImagePath))
		_ = registry.CopyFile(app.AppImagePath, destBin)
		_ = os.Remove(app.AppImagePath)
	}

	if registry.FileExists(app.DesktopFile) {
		destDesk := filepath.Join(filesDirectory, filepath.Base(app.DesktopFile))
		_ = registry.CopyFile(app.DesktopFile, destDesk)
		_ = os.Remove(app.DesktopFile)
	}

	for _, iconPath := range app.IconFiles {
		if registry.FileExists(iconPath) {
			destIcon := filepath.Join(filesDirectory, filepath.Base(iconPath))
			_ = registry.CopyFile(iconPath, destIcon)
			_ = os.Remove(iconPath)
		}
	}

	if registry.FileExists(app.WrapperFile) {
		_ = os.Remove(app.WrapperFile)
	}

	appDirectory := filepath.Join(pathsInstance.AppsDir, app.ID)
	_ = os.RemoveAll(appDirectory)

	item := Item{
		TrashID:     trashIdentifier,
		DeletedAt:   now.Format(time.RFC3339),
		ExpiresAt:   now.AddDate(0, 0, retentionDays).Format(time.RFC3339),
		AppManifest: *app,
		TrashPath:   itemDirectory,
	}

	data, err := json.MarshalIndent(item, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(itemDirectory, "trash_manifest.json"), data, 0644)
	}

	history.Record(pathsInstance, "remove_to_trash", app.ID, "success",
		fmt.Sprintf("Moved %s to trash", app.Name))

	registry.RefreshDesktopDatabase(pathsInstance)
	registry.RefreshIconCache(pathsInstance)
	return nil
}

func List(pathsInstance *paths.Paths) ([]Item, error) {
	trashRoot := Directory(pathsInstance)
	if _, err := os.Stat(trashRoot); os.IsNotExist(err) {
		oldTrash := filepath.Join(pathsInstance.OldAdoptHome, "trash")
		if _, oldErr := os.Stat(oldTrash); oldErr == nil {
			trashRoot = oldTrash
		} else {
			return nil, nil
		}
	}

	entries, err := os.ReadDir(trashRoot)
	if err != nil {
		return nil, err
	}

	var items []Item
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(trashRoot, entry.Name(), "trash_manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var item Item
		if err := json.Unmarshal(data, &item); err == nil {
			item.TrashPath = filepath.Join(trashRoot, entry.Name())
			items = append(items, item)
		}
	}
	return items, nil
}

func Restore(pathsInstance *paths.Paths, trashIdentifier string) error {
	trashRoot := Directory(pathsInstance)
	itemDirectory := filepath.Join(trashRoot, trashIdentifier)
	if _, err := os.Stat(itemDirectory); os.IsNotExist(err) {
		oldItem := filepath.Join(pathsInstance.OldAdoptHome, "trash", trashIdentifier)
		if _, oldErr := os.Stat(oldItem); oldErr == nil {
			itemDirectory = oldItem
		}
	}

	manifestPath := filepath.Join(itemDirectory, "trash_manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("trash manifest not found")
	}

	var item Item
	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}

	app := item.AppManifest
	existingDirectory := filepath.Join(pathsInstance.AppsDir, app.ID)
	if _, err := os.Stat(existingDirectory); err == nil {
		return fmt.Errorf("conflict: application '%s' is already installed. Remove it before restoring", app.ID)
	}

	filesDirectory := filepath.Join(itemDirectory, "files")
	_ = os.MkdirAll(existingDirectory, 0755)

	binSource := filepath.Join(filesDirectory, filepath.Base(app.AppImagePath))
	if registry.FileExists(binSource) {
		_ = registry.CopyFile(binSource, app.AppImagePath)
		_ = os.Chmod(app.AppImagePath, 0755)
	}

	deskSource := filepath.Join(filesDirectory, filepath.Base(app.DesktopFile))
	if registry.FileExists(deskSource) {
		_ = os.MkdirAll(filepath.Dir(app.DesktopFile), 0755)
		_ = registry.CopyFile(deskSource, app.DesktopFile)
	}

	for _, iconPath := range app.IconFiles {
		iconSource := filepath.Join(filesDirectory, filepath.Base(iconPath))
		if registry.FileExists(iconSource) {
			_ = os.MkdirAll(filepath.Dir(iconPath), 0755)
			_ = registry.CopyFile(iconSource, iconPath)
		}
	}

	if app.WrapperFile != "" {
		_ = os.MkdirAll(filepath.Dir(app.WrapperFile), 0755)
		wrapperScript := fmt.Sprintf("#!/usr/bin/env bash\nexec \"%s\" \"$@\"\n", app.AppImagePath)
		_ = os.WriteFile(app.WrapperFile, []byte(wrapperScript), 0755)
	}

	_ = registry.Save(pathsInstance, &app)
	_ = os.RemoveAll(itemDirectory)

	history.Record(pathsInstance, "restore", app.ID, "success",
		fmt.Sprintf("Restored %s from trash", app.Name))

	registry.RefreshDesktopDatabase(pathsInstance)
	registry.RefreshIconCache(pathsInstance)
	return nil
}

func PurgeExpired(pathsInstance *paths.Paths) {
	items, err := List(pathsInstance)
	if err != nil || len(items) == 0 {
		return
	}

	now := time.Now()
	for _, item := range items {
		if expiration, err := time.Parse(time.RFC3339, item.ExpiresAt); err == nil {
			if now.After(expiration) {
				_ = os.RemoveAll(item.TrashPath)
			}
		}
	}
}

func Empty(pathsInstance *paths.Paths) error {
	trashRoot := Directory(pathsInstance)
	_ = os.RemoveAll(filepath.Join(pathsInstance.OldAdoptHome, "trash"))
	return os.RemoveAll(trashRoot)
}
