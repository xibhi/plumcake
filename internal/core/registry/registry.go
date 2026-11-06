package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/sandbox"
	"github.com/xibhi/plumcake/internal/core/trust"
)

func LoadAll(pathsInstance *paths.Paths) ([]App, error) {
	if _, err := os.Stat(pathsInstance.AppsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(pathsInstance.AppsDir)
	if err != nil {
		return nil, err
	}

	var apps []App
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(pathsInstance.AppsDir, entry.Name(), "app.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			oldManifestPath := filepath.Join(pathsInstance.AppsDir, entry.Name(), "adopt.json")
			data, err = os.ReadFile(oldManifestPath)
			if err != nil {
				continue
			}
		}

		var app App
		if err := json.Unmarshal(data, &app); err == nil {
			if Migrate(&app) {
				_ = Save(pathsInstance, &app)
			}
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func Migrate(app *App) bool {
	migrated := false
	if app.SchemaVersion < 2 {
		app.SchemaVersion = 2
		migrated = true
	}

	if app.Trust == "" {
		if app.BinarySHA256 != "" {
			app.Trust = trust.Verified
		} else {
			app.Trust = trust.Unsigned
		}
		migrated = true
	}

	if app.Checksum == "" && app.BinarySHA256 != "" {
		app.Checksum = app.BinarySHA256
		migrated = true
	}

	if app.Sandbox.Tool == "" {
		app.Sandbox = SandboxConfig{
			Enabled:      false,
			Tool:         "off",
			Network:      true,
			OriginalExec: app.WrapperFile,
		}
		migrated = true
	}

	if app.UpdateInfo.SourceType == "" {
		sourceType := "file"
		sourceURL := ""
		if app.SourceURL != nil && *app.SourceURL != "" {
			sourceURL = *app.SourceURL
			if strings.Contains(sourceURL, "github.com") {
				sourceType = "github"
			} else {
				sourceType = "url"
			}
		}
		app.UpdateInfo = UpdateConfig{
			SourceType: sourceType,
			SourceURL:  sourceURL,
		}
		migrated = true
	}

	if len(app.VersionHistory) == 0 && app.Version != "" {
		app.VersionHistory = []VersionRecord{
			{
				Version:     app.Version,
				Path:        app.AppImagePath,
				InstalledAt: app.InstalledAt,
				Size:        app.BinarySize,
				SHA256:      app.BinarySHA256,
				Trust:       app.Trust,
			},
		}
		migrated = true
	}

	return migrated
}

func Save(pathsInstance *paths.Paths, app *App) error {
	directory := filepath.Join(pathsInstance.AppsDir, app.ID)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "app.json"), data, 0644)
}

func Rename(pathsInstance *paths.Paths, app *App, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("name cannot be empty")
	}

	app.Name = newName

	if app.DesktopFile != "" && FileExists(app.DesktopFile) {
		content, err := os.ReadFile(app.DesktopFile)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "Name=") {
					lines[i] = "Name=" + newName
					break
				}
			}
			_ = os.WriteFile(app.DesktopFile, []byte(strings.Join(lines, "\n")), 0644)
		}
	}

	if err := Save(pathsInstance, app); err != nil {
		return err
	}

	RefreshDesktopDatabase(pathsInstance)
	return nil
}

func ValidateImageFile(filePath string) (bool, bool, error) {
	fileDescriptor, err := os.Open(filePath)
	if err != nil {
		return false, false, err
	}
	defer fileDescriptor.Close()

	header := make([]byte, 512)
	bytesRead, err := io.ReadFull(fileDescriptor, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return false, false, err
	}
	header = header[:bytesRead]

	if bytes.HasPrefix(header, []byte("\x89PNG\r\n\x1a\n")) {
		return true, false, nil
	}

	contentLower := strings.ToLower(string(header))
	if strings.Contains(contentLower, "<svg") || strings.HasSuffix(strings.ToLower(filePath), ".svg") {
		return false, true, nil
	}

	return false, false, fmt.Errorf("file is not a valid PNG or SVG image")
}

func ChangeIcon(pathsInstance *paths.Paths, app *App, newIconPath string) error {
	isPNG, isSVG, err := ValidateImageFile(newIconPath)
	if err != nil {
		return err
	}

	iconIdentifier := "plumcake-" + app.ID
	var newIconFiles []string

	if isSVG {
		targetScalable := filepath.Join(pathsInstance.IconsDir, "hicolor", "scalable", "apps", iconIdentifier+".svg")
		targetPixmap := filepath.Join(pathsInstance.PixmapsDir, iconIdentifier+".svg")

		_ = os.MkdirAll(filepath.Dir(targetScalable), 0755)
		_ = os.MkdirAll(filepath.Dir(targetPixmap), 0755)

		if err := CopyFile(newIconPath, targetScalable); err != nil {
			return err
		}
		_ = CopyFile(newIconPath, targetPixmap)
		newIconFiles = append(newIconFiles, targetScalable, targetPixmap)
	} else if isPNG {
		targetPNG := filepath.Join(pathsInstance.IconsDir, "hicolor", "256x256", "apps", iconIdentifier+".png")
		targetPixmap := filepath.Join(pathsInstance.PixmapsDir, iconIdentifier+".png")

		_ = os.MkdirAll(filepath.Dir(targetPNG), 0755)
		_ = os.MkdirAll(filepath.Dir(targetPixmap), 0755)

		if err := CopyFile(newIconPath, targetPNG); err != nil {
			return err
		}
		_ = CopyFile(newIconPath, targetPixmap)
		newIconFiles = append(newIconFiles, targetPNG, targetPixmap)
	}

	app.IconFiles = newIconFiles
	if err := Save(pathsInstance, app); err != nil {
		return err
	}

	RefreshIconCache(pathsInstance)
	return nil
}

func ApplySandbox(pathsInstance *paths.Paths, app *App, tool string, allowNetwork bool) error {
	tools := sandbox.Detect()
	toolClean := strings.ToLower(strings.TrimSpace(tool))

	if toolClean == "firejail" && !tools.HasFirejail {
		return fmt.Errorf("firejail is not installed on system")
	}
	if toolClean == "bwrap" && !tools.HasBwrap {
		return fmt.Errorf("bubblewrap is not installed on system")
	}

	if app.Sandbox.OriginalExec == "" {
		app.Sandbox.OriginalExec = app.WrapperFile + " %U"
	}

	app.Sandbox.Tool = toolClean
	app.Sandbox.Network = allowNetwork
	app.Sandbox.Enabled = toolClean != "off" && toolClean != ""

	newExec := sandbox.GenerateExec(app.Sandbox.OriginalExec, app.AppImagePath, toolClean, allowNetwork)

	if app.DesktopFile != "" && FileExists(app.DesktopFile) {
		content, err := os.ReadFile(app.DesktopFile)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if strings.HasPrefix(line, "Exec=") {
					lines[i] = "Exec=" + newExec
					break
				}
			}
			_ = os.WriteFile(app.DesktopFile, []byte(strings.Join(lines, "\n")), 0644)
		}
	}

	if err := Save(pathsInstance, app); err != nil {
		return err
	}

	RefreshDesktopDatabase(pathsInstance)
	return nil
}

func Launch(app *App) error {
	target := app.WrapperFile
	if target == "" || !FileExists(target) {
		target = app.AppImagePath
	}
	if target == "" || !FileExists(target) {
		return fmt.Errorf("executable not found on disk")
	}

	command := exec.Command(target)
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return command.Start()
}

func OpenContainingFolder(app *App) error {
	directory := filepath.Dir(app.AppImagePath)
	if directory == "" || !FileExists(directory) {
		directory = filepath.Dir(app.WrapperFile)
	}
	if directory == "" {
		return fmt.Errorf("folder not found")
	}

	command := exec.Command("xdg-open", directory)
	return command.Start()
}

func RefreshDesktopDatabase(pathsInstance *paths.Paths) {
	if _, err := exec.LookPath("update-desktop-database"); err == nil {
		_ = exec.Command("update-desktop-database", pathsInstance.ApplicationsDir).Run()
	}
}

func RefreshIconCache(pathsInstance *paths.Paths) {
	if _, err := exec.LookPath("gtk-update-icon-cache"); err == nil {
		hicolor := filepath.Join(pathsInstance.IconsDir, "hicolor")
		_ = exec.Command("gtk-update-icon-cache", "-q", "-t", "-f", hicolor).Run()
	}
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func CopyFile(sourcePath, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}
