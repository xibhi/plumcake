package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xibhi/plumcake/internal/core/history"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func ReadEmbeddedUpdateInfo(ctx context.Context, appImagePath string) string {
	command := exec.CommandContext(ctx, appImagePath, "--appimage-updateinformation")
	output, err := command.CombinedOutput()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}
	return ""
}

func Check(ctx context.Context, app *registry.App, includePrerelease bool) (bool, string, error) {
	app.UpdateInfo.CheckedAt = time.Now().Format(time.RFC3339)

	sourceURL := ""
	if app.SourceURL != nil {
		sourceURL = *app.SourceURL
	}
	if sourceURL == "" && app.UpdateInfo.SourceURL != "" {
		sourceURL = app.UpdateInfo.SourceURL
	}

	if owner, repo, isRepo := ParseGitHubRepoInput(sourceURL); isRepo {
		assets, err := FetchGitHubReleaseAssets(ctx, owner, repo, includePrerelease)
		if err != nil {
			return false, "", err
		}
		if len(assets) > 0 {
			latestTag := strings.TrimPrefix(assets[0].TagName, "v")
			currentVersion := strings.TrimPrefix(app.Version, "v")
			if latestTag != "" && latestTag != currentVersion {
				app.UpdateInfo.HasUpdate = true
				app.UpdateInfo.LatestVersion = assets[0].TagName
				app.UpdateInfo.SourceURL = assets[0].DownloadURL
				return true, assets[0].TagName, nil
			}
		}
	}

	updateString := ReadEmbeddedUpdateInfo(ctx, app.AppImagePath)
	if updateString != "" {
		parts := strings.Split(updateString, "|")
		if len(parts) >= 3 && parts[0] == "gh-releases-zsync" {
			owner, repo := parts[1], parts[2]
			assets, err := FetchGitHubReleaseAssets(ctx, owner, repo, includePrerelease)
			if err == nil && len(assets) > 0 {
				latestTag := strings.TrimPrefix(assets[0].TagName, "v")
				currentVersion := strings.TrimPrefix(app.Version, "v")
				if latestTag != "" && latestTag != currentVersion {
					app.UpdateInfo.HasUpdate = true
					app.UpdateInfo.LatestVersion = assets[0].TagName
					app.UpdateInfo.SourceURL = assets[0].DownloadURL
					return true, assets[0].TagName, nil
				}
			}
		}
	}

	app.UpdateInfo.HasUpdate = false
	return false, "", nil
}

func ArchiveVersion(pathsInstance *paths.Paths, app *registry.App, keepCount int) error {
	versionsDirectory := filepath.Join(pathsInstance.AppsDir, app.ID, "versions")
	if err := os.MkdirAll(versionsDirectory, 0755); err != nil {
		return err
	}

	archiveFilename := fmt.Sprintf("%s-%s.AppImage", app.ID, app.Version)
	archivePath := filepath.Join(versionsDirectory, archiveFilename)

	if registry.FileExists(app.AppImagePath) && (!registry.FileExists(archivePath) || archivePath != app.AppImagePath) {
		if err := registry.CopyFile(app.AppImagePath, archivePath); err != nil {
			return err
		}
		_ = os.Chmod(archivePath, 0755)
	}

	fileInformation, _ := os.Stat(app.AppImagePath)
	var size int64
	if fileInformation != nil {
		size = fileInformation.Size()
	}

	record := registry.VersionRecord{
		Version:     app.Version,
		Path:        archivePath,
		InstalledAt: time.Now().Format(time.RFC3339),
		Size:        size,
		SHA256:      app.BinarySHA256,
		Trust:       app.Trust,
	}

	var newHistory []registry.VersionRecord
	for _, recorded := range app.VersionHistory {
		if recorded.Version != app.Version {
			newHistory = append(newHistory, recorded)
		}
	}
	app.VersionHistory = append([]registry.VersionRecord{record}, newHistory...)

	Prune(app, keepCount)
	return registry.Save(pathsInstance, app)
}

func Prune(app *registry.App, keepCount int) {
	if keepCount < 1 {
		keepCount = 2
	}
	if len(app.VersionHistory) <= keepCount {
		return
	}

	for i := keepCount; i < len(app.VersionHistory); i++ {
		filePath := app.VersionHistory[i].Path
		if filePath != "" && registry.FileExists(filePath) && filePath != app.AppImagePath {
			_ = os.Remove(filePath)
		}
	}
	app.VersionHistory = app.VersionHistory[:keepCount]
}

func Rollback(pathsInstance *paths.Paths, app *registry.App, targetVersion string) error {
	var targetRecord *registry.VersionRecord
	for i := range app.VersionHistory {
		if app.VersionHistory[i].Version == targetVersion {
			targetRecord = &app.VersionHistory[i]
			break
		}
	}

	if targetRecord == nil || !registry.FileExists(targetRecord.Path) {
		return fmt.Errorf("archived version %s not found on disk", targetVersion)
	}

	temporaryTarget := app.AppImagePath + ".swap"
	if err := registry.CopyFile(targetRecord.Path, temporaryTarget); err != nil {
		return err
	}
	_ = os.Chmod(temporaryTarget, 0755)

	if err := os.Rename(temporaryTarget, app.AppImagePath); err != nil {
		return err
	}

	app.Version = targetRecord.Version
	app.BinarySHA256 = targetRecord.SHA256
	app.Checksum = targetRecord.SHA256
	app.Trust = targetRecord.Trust
	app.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := registry.Save(pathsInstance, app); err != nil {
		return err
	}

	history.Record(pathsInstance, "rollback", app.ID, "success",
		fmt.Sprintf("Rolled back %s to version %s", app.Name, targetVersion))

	registry.RefreshDesktopDatabase(pathsInstance)
	return nil
}
