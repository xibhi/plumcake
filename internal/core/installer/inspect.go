package installer

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xibhi/plumcake/internal/core/appimage"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/trust"
)

type PreInstallInspection struct {
	Name            string
	Version         string
	Size            int64
	Arch            string
	ArchMatches     bool
	SourceDisplay   string
	IsHTTPS         bool
	TrustResult     trust.Result
	DesktopName     string
	DesktopExec     string
	DesktopCategory string
	Warnings        []string
	IsAlreadyAdded  bool
	ExistingApp     *registry.App
	LocalBinaryPath string
}

func InspectPreInstall(ctx context.Context, pathsInstance *paths.Paths, localPath, sourceURL string, expectedDigest string) (*PreInstallInspection, error) {
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}

	valid, architecture, _, err := appimage.Validate(localPath)
	if err != nil || !valid {
		return nil, fmt.Errorf("file is not a valid AppImage ELF binary")
	}

	machineArch := runtime.GOARCH
	archMatches := (machineArch == "amd64" && architecture == "x86_64") || (machineArch == "arm64" && architecture == "aarch64") || architecture == "universal"

	sourceDisplay := localPath
	isHTTPS := false
	if sourceURL != "" {
		if parsedURL, err := url.Parse(sourceURL); err == nil && parsedURL.Host != "" {
			sourceDisplay = parsedURL.Host + parsedURL.Path
			isHTTPS = parsedURL.Scheme == "https"
		} else {
			sourceDisplay = sourceURL
			isHTTPS = strings.HasPrefix(sourceURL, "https://")
		}
	}

	trustResult := trust.VerifyBinary(ctx, localPath, sourceURL, expectedDigest)

	temporaryWorkDirectory, err := os.MkdirTemp("", "plumcake_inspect_*")
	if err == nil {
		defer os.RemoveAll(temporaryWorkDirectory)
		_ = os.Chmod(localPath, 0755)

		commandDesktop := exec.CommandContext(ctx, localPath, "--appimage-extract", "*.desktop")
		commandDesktop.Dir = temporaryWorkDirectory
		_ = commandDesktop.Run()

		commandShare := exec.CommandContext(ctx, localPath, "--appimage-extract", "usr/share/applications/*")
		commandShare.Dir = temporaryWorkDirectory
		_ = commandShare.Run()
	}

	desktopName := ""
	desktopExec := ""
	desktopCategory := ""

	squashRoot := filepath.Join(temporaryWorkDirectory, "squashfs-root")
	var desktopCandidates []string
	if entries, err := os.ReadDir(squashRoot); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".desktop") {
				desktopCandidates = append(desktopCandidates, filepath.Join(squashRoot, entry.Name()))
			}
		}
	}
	appShareDirectory := filepath.Join(squashRoot, "usr", "share", "applications")
	if entries, err := os.ReadDir(appShareDirectory); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".desktop") {
				desktopCandidates = append(desktopCandidates, filepath.Join(appShareDirectory, entry.Name()))
			}
		}
	}

	for _, candidate := range desktopCandidates {
		resolvedPath := candidate
		if info, err := os.Lstat(candidate); err == nil && (info.Mode()&os.ModeSymlink != 0) {
			if linkTarget, err := os.Readlink(candidate); err == nil {
				if !filepath.IsAbs(linkTarget) {
					resolvedPath = filepath.Join(filepath.Dir(candidate), linkTarget)
				} else {
					resolvedPath = filepath.Join(squashRoot, strings.TrimPrefix(linkTarget, "/"))
				}
			}
		}
		if data, err := os.ReadFile(resolvedPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Name=") && desktopName == "" {
					desktopName = strings.Trim(strings.TrimPrefix(line, "Name="), "\"'")
				} else if strings.HasPrefix(line, "Exec=") && desktopExec == "" {
					desktopExec = strings.Trim(strings.TrimPrefix(line, "Exec="), "\"'")
				} else if strings.HasPrefix(line, "Categories=") && desktopCategory == "" {
					desktopCategory = strings.Trim(strings.TrimPrefix(line, "Categories="), "\"'")
				}
			}
			if desktopName != "" {
				break
			}
		}
	}

	apps, _ := registry.LoadAll(pathsInstance)
	var existingApp *registry.App
	for i := range apps {
		if apps[i].OriginalFilename == filepath.Base(localPath) || (desktopName != "" && strings.EqualFold(apps[i].Name, desktopName)) {
			existingApp = &apps[i]
			break
		}
	}

	var warnings []string
	if !archMatches {
		warnings = append(warnings, fmt.Sprintf("Architecture mismatch: binary is %s, machine is %s", architecture, machineArch))
	}
	if sourceURL != "" && !isHTTPS && strings.HasPrefix(sourceURL, "http://") {
		warnings = append(warnings, "Insecure source: downloaded over unencrypted HTTP")
	}
	if trustResult.State == trust.Mismatch {
		warnings = append(warnings, "CRITICAL: Digest mismatch! Package may be modified or corrupted.")
	} else if trustResult.State == trust.Unsigned {
		warnings = append(warnings, "No digital signature or published checksum found.")
	}
	if existingApp != nil {
		warnings = append(warnings, fmt.Sprintf("Already installed as '%s' (reinstall/upgrade)", existingApp.Name))
	}
	if desktopExec != "" {
		execLower := strings.ToLower(desktopExec)
		if strings.Contains(execLower, "sudo") || strings.Contains(execLower, "curl") || strings.Contains(execLower, "wget") || strings.Contains(execLower, "bash -c") || strings.Contains(execLower, "sh -c") {
			warnings = append(warnings, fmt.Sprintf("Unusual Exec line: '%s'", registry.Truncate(desktopExec, 36)))
		}
	}

	displayName := desktopName
	if displayName == "" {
		displayName = filepath.Base(localPath)
	}

	return &PreInstallInspection{
		Name:            displayName,
		Version:         "1.0",
		Size:            fileInfo.Size(),
		Arch:            architecture,
		ArchMatches:     archMatches,
		SourceDisplay:   sourceDisplay,
		IsHTTPS:         isHTTPS,
		TrustResult:     trustResult,
		DesktopName:     desktopName,
		DesktopExec:     desktopExec,
		DesktopCategory: desktopCategory,
		Warnings:        warnings,
		IsAlreadyAdded:  existingApp != nil,
		ExistingApp:     existingApp,
		LocalBinaryPath: localPath,
	}, nil
}
