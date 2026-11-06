package migrate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xibhi/plumcake/internal/core/history"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func RunMigrationIfNeeded(pathsInstance *paths.Paths) error {
	migratedAnything := false

	if pathsInstance.OldAdoptConfig != "" && pathsInstance.OldAdoptConfig != pathsInstance.ConfigHome {
		if _, err := os.Stat(pathsInstance.OldAdoptConfig); err == nil {
			targetConfig := filepath.Join(pathsInstance.ConfigHome, "plumcake")
			if _, targetErr := os.Stat(targetConfig); os.IsNotExist(targetErr) {
				_ = os.MkdirAll(filepath.Dir(targetConfig), 0755)
				if err := os.Rename(pathsInstance.OldAdoptConfig, targetConfig); err == nil {
					migratedAnything = true
				}
			}
		}
	}

	if pathsInstance.OldAdoptState != "" && pathsInstance.OldAdoptState != pathsInstance.StateHome {
		if _, err := os.Stat(pathsInstance.OldAdoptState); err == nil {
			targetState := filepath.Join(pathsInstance.StateHome, "plumcake")
			if _, targetErr := os.Stat(targetState); os.IsNotExist(targetErr) {
				_ = os.MkdirAll(filepath.Dir(targetState), 0755)
				if err := os.Rename(pathsInstance.OldAdoptState, targetState); err == nil {
					migratedAnything = true
				}
			}
		}
	}

	if pathsInstance.OldAdoptHome != "" && pathsInstance.OldAdoptHome != pathsInstance.PlumcakeHome {
		if _, err := os.Stat(pathsInstance.OldAdoptHome); err == nil {
			targetData := pathsInstance.PlumcakeHome
			if _, targetErr := os.Stat(targetData); os.IsNotExist(targetErr) {
				_ = os.MkdirAll(filepath.Dir(targetData), 0755)
				if err := os.Rename(pathsInstance.OldAdoptHome, targetData); err == nil {
					migratedAnything = true
				}
			}
		}
	}

	apps, err := registry.LoadAll(pathsInstance)
	if err == nil {
		for i := range apps {
			needsSave := false
			oldPathFragment := filepath.Join("adopt", "apps")
			newPathFragment := filepath.Join("plumcake", "apps")

			if strings.Contains(apps[i].AppImagePath, oldPathFragment) {
				apps[i].AppImagePath = strings.Replace(apps[i].AppImagePath, oldPathFragment, newPathFragment, 1)
				needsSave = true
			}

			if strings.HasPrefix(filepath.Base(apps[i].DesktopFile), "adopt-") {
				oldBase := filepath.Base(apps[i].DesktopFile)
				newBase := strings.Replace(oldBase, "adopt-", "plumcake-", 1)
				newDesktopPath := filepath.Join(filepath.Dir(apps[i].DesktopFile), newBase)
				if registry.FileExists(apps[i].DesktopFile) {
					_ = os.Rename(apps[i].DesktopFile, newDesktopPath)
				}
				apps[i].DesktopFile = newDesktopPath
				needsSave = true
			}

			var updatedIcons []string
			for _, iconPath := range apps[i].IconFiles {
				if strings.Contains(filepath.Base(iconPath), "adopt-") {
					oldBase := filepath.Base(iconPath)
					newBase := strings.Replace(oldBase, "adopt-", "plumcake-", 1)
					newIconPath := filepath.Join(filepath.Dir(iconPath), newBase)
					if registry.FileExists(iconPath) {
						_ = os.Rename(iconPath, newIconPath)
					}
					updatedIcons = append(updatedIcons, newIconPath)
					needsSave = true
				} else {
					updatedIcons = append(updatedIcons, iconPath)
				}
			}
			apps[i].IconFiles = updatedIcons

			for vIdx := range apps[i].VersionHistory {
				if strings.Contains(apps[i].VersionHistory[vIdx].Path, oldPathFragment) {
					apps[i].VersionHistory[vIdx].Path = strings.Replace(apps[i].VersionHistory[vIdx].Path, oldPathFragment, newPathFragment, 1)
					needsSave = true
				}
			}

			if apps[i].DesktopFile != "" && registry.FileExists(apps[i].DesktopFile) {
				content, readErr := os.ReadFile(apps[i].DesktopFile)
				if readErr == nil {
					deskContent := string(content)
					if strings.Contains(deskContent, "X-Adopt-") || strings.Contains(deskContent, "adopt-") || strings.Contains(deskContent, oldPathFragment) {
						deskContent = strings.ReplaceAll(deskContent, "X-Adopt-Managed", "X-Plumcake-Managed")
						deskContent = strings.ReplaceAll(deskContent, "X-Adopt-ID", "X-Plumcake-ID")
						deskContent = strings.ReplaceAll(deskContent, "X-Adopt-Version", "X-Plumcake-Version")
						deskContent = strings.ReplaceAll(deskContent, "Icon=adopt-", "Icon=plumcake-")
						deskContent = strings.ReplaceAll(deskContent, oldPathFragment, newPathFragment)
						_ = os.WriteFile(apps[i].DesktopFile, []byte(deskContent), 0644)
					}
				}
			}

			if needsSave {
				_ = registry.Save(pathsInstance, &apps[i])
				migratedAnything = true
			}
		}
	}

	if migratedAnything {
		history.Record(pathsInstance, "migrate", "system", "success", "Migrated legacy adopt data to plumcake")
		registry.RefreshDesktopDatabase(pathsInstance)
		registry.RefreshIconCache(pathsInstance)
	}

	return nil
}
