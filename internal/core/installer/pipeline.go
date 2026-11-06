package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xibhi/plumcake/internal/core/appimage"
	"github.com/xibhi/plumcake/internal/core/history"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/trust"
	"github.com/xibhi/plumcake/internal/core/updater"
)

type StepID string

const (
	StepFetch          StepID = "fetch"
	StepVerify         StepID = "verify"
	StepMakeExecutable StepID = "make executable"
	StepExtractIcon    StepID = "extract icon"
	StepWriteDesktop   StepID = "write .desktop"
	StepRefreshAppMenu StepID = "refresh app menu"
)

type Event struct {
	Step       StepID
	StepIndex  int
	TotalSteps int
	Progress   float64
	Detail     string
	BytesDone  int64
	BytesTotal int64
	Speed      int64
	Done       bool
	Err        error
	Manifest   *registry.App
}

type Runner struct {
	Paths *paths.Paths
}

func NewRunner(pathsInstance *paths.Paths) *Runner {
	return &Runner{Paths: pathsInstance}
}

func (r *Runner) Run(
	ctx context.Context,
	source string,
	portable bool,
	moveSource bool,
	expectedDigest string,
	out chan<- Event,
) {
	defer close(out)

	totalSteps := 6
	emit := func(step StepID, idx int, prog float64, detail string, done bool, err error, m *registry.App) {
		select {
		case <-ctx.Done():
			return
		case out <- Event{
			Step:       step,
			StepIndex:  idx,
			TotalSteps: totalSteps,
			Progress:   prog,
			Detail:     detail,
			Done:       done,
			Err:        err,
			Manifest:   m,
		}:
		}
	}

	temporaryDirectory, err := os.MkdirTemp("", "plumcake_pipe_*")
	if err != nil {
		emit(StepFetch, 1, 0, "Failed creating temp dir", false, err, nil)
		return
	}
	defer os.RemoveAll(temporaryDirectory)

	var localAppImagePath string
	isRemote := strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")

	emit(StepFetch, 1, 0.1, "Fetching binary source...", false, nil, nil)
	if isRemote {
		request, err := http.NewRequestWithContext(ctx, "GET", source, nil)
		if err != nil {
			emit(StepFetch, 1, 0, "Invalid URL", false, err, nil)
			return
		}
		request.Header.Set("User-Agent", "plumcake/1.0.1 (Linux; x86_64)")

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			emit(StepFetch, 1, 0, "Network download failed", false, err, nil)
			return
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			emit(StepFetch, 1, 0, fmt.Sprintf("HTTP %d: download refused", response.StatusCode), false, fmt.Errorf("HTTP error %d", response.StatusCode), nil)
			return
		}

		filename := filepath.Base(response.Request.URL.Path)
		if filename == "" || filename == "/" || !strings.HasSuffix(strings.ToLower(filename), ".appimage") {
			filename = "application.AppImage"
		}
		localAppImagePath = filepath.Join(temporaryDirectory, filename)

		outputFile, err := os.Create(localAppImagePath)
		if err != nil {
			emit(StepFetch, 1, 0, "Cannot write temp file", false, err, nil)
			return
		}

		totalBytes := response.ContentLength
		var downloaded int64
		buffer := make([]byte, 64*1024)
		startTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				outputFile.Close()
				return
			default:
			}

			bytesRead, readErr := response.Body.Read(buffer)
			if bytesRead > 0 {
				_, writeErr := outputFile.Write(buffer[:bytesRead])
				if writeErr != nil {
					outputFile.Close()
					emit(StepFetch, 1, 0, "Write error", false, writeErr, nil)
					return
				}
				downloaded += int64(bytesRead)

				speed := int64(float64(downloaded) / max(0.001, time.Since(startTime).Seconds()))
				prog := 0.2
				if totalBytes > 0 {
					prog = 0.1 + (float64(downloaded)/float64(totalBytes))*0.15
				}

				select {
				case out <- Event{
					Step:       StepFetch,
					StepIndex:  1,
					TotalSteps: totalSteps,
					Progress:   prog,
					Detail:     fmt.Sprintf("Downloading (%s / %s) %s/s", registry.FormatSize(downloaded), registry.FormatSize(totalBytes), registry.FormatSize(speed)),
					BytesDone:  downloaded,
					BytesTotal: totalBytes,
					Speed:      speed,
				}:
				default:
				}
			}

			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				outputFile.Close()
				emit(StepFetch, 1, 0, "Download interrupted", false, readErr, nil)
				return
			}
		}
		outputFile.Close()
	} else {
		resolved, err := filepath.Abs(source)
		if err != nil || !registry.FileExists(resolved) {
			emit(StepFetch, 1, 0, fmt.Sprintf("File not found: %s", source), false, fmt.Errorf("source file not found"), nil)
			return
		}
		localAppImagePath = resolved
	}

	emit(StepVerify, 2, 0.35, "Verifying ELF magic, checksum & trust...", false, nil, nil)
	valid, architecture, appImageType, err := appimage.Validate(localAppImagePath)
	if err != nil || !valid {
		emit(StepVerify, 2, 0.35, "Invalid AppImage: missing ELF header or signature", false, fmt.Errorf("not an AppImage binary"), nil)
		return
	}

	trustResult := trust.VerifyBinary(ctx, localAppImagePath, source, expectedDigest)
	if trustResult.IsHardFail {
		emit(StepVerify, 2, 0.35, trustResult.Message, false, fmt.Errorf("verification failed: %s", trustResult.Message), nil)
		return
	}

	emit(StepMakeExecutable, 3, 0.50, fmt.Sprintf("Applying executable bits (%s %s)...", architecture, appImageType), false, nil, nil)
	if err := os.Chmod(localAppImagePath, 0755); err != nil {
		emit(StepMakeExecutable, 3, 0.50, "Failed setting permissions", false, err, nil)
		return
	}

	emit(StepExtractIcon, 4, 0.65, "Extracting icons & desktop assets without FUSE...", false, nil, nil)
	tempExtractDir, err := os.MkdirTemp("", "plumcake_extract_*")
	if err != nil {
		emit(StepExtractIcon, 4, 0.65, "Failed creating extraction directory", false, err, nil)
		return
	}
	defer os.RemoveAll(tempExtractDir)

	extractPatterns := []string{
		"*.desktop",
		"usr/share/applications/*.desktop",
		".DirIcon*",
		"*.png",
		"*.svg",
		"usr/share/icons/*",
		"usr/share/pixmaps/*",
	}
	for _, pattern := range extractPatterns {
		cmd := exec.CommandContext(ctx, localAppImagePath, "--appimage-extract", pattern)
		cmd.Dir = tempExtractDir
		_ = cmd.Run()
	}

	squashRoot := filepath.Join(tempExtractDir, "squashfs-root")
	if _, err := os.Stat(squashRoot); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, localAppImagePath, "--appimage-extract")
		cmd.Dir = tempExtractDir
		_ = cmd.Run()
	}

	desktopMeta := parseExtractedDesktop(squashRoot, localAppImagePath)

	appID := slugify(desktopMeta.Name)
	if appID == "" {
		appID = slugify(filepath.Base(localAppImagePath))
	}
	if appID == "" {
		appID = "app"
	}

	execName := appID
	if desktopMeta.Exec != "" {
		fields := strings.Fields(desktopMeta.Exec)
		if len(fields) > 0 {
			candidateExec := slugify(filepath.Base(fields[0]))
			if candidateExec != "" {
				execName = candidateExec
			}
		}
	}

	iconFiles := installExtractedIcons(r.Paths, squashRoot, appID, desktopMeta.Icon)

	emit(StepWriteDesktop, 5, 0.85, "Writing .desktop entry & ~/.local/bin wrapper...", false, nil, nil)

	appDir := filepath.Join(r.Paths.AppsDir, appID)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		emit(StepWriteDesktop, 5, 0.85, "Failed creating app storage directory", false, err, nil)
		return
	}

	destBinaryPath := filepath.Join(appDir, filepath.Base(localAppImagePath))
	if destBinaryPath != localAppImagePath {
		if err := registry.CopyFile(localAppImagePath, destBinaryPath); err != nil {
			emit(StepWriteDesktop, 5, 0.85, "Failed saving binary", false, err, nil)
			return
		}
	}
	_ = os.Chmod(destBinaryPath, 0755)

	if portable {
		_ = os.MkdirAll(destBinaryPath+".home", 0755)
		_ = os.MkdirAll(destBinaryPath+".config", 0755)
	}

	if err := os.MkdirAll(r.Paths.BinDir, 0755); err != nil {
		emit(StepWriteDesktop, 5, 0.85, "Failed creating user bin directory", false, err, nil)
		return
	}

	wrapperPath := filepath.Join(r.Paths.BinDir, execName)
	wrapperScript := fmt.Sprintf("#!/usr/bin/env bash\nexec \"%s\" \"$@\"\n", destBinaryPath)
	if err := os.WriteFile(wrapperPath, []byte(wrapperScript), 0755); err != nil {
		emit(StepWriteDesktop, 5, 0.85, "Failed creating binary wrapper", false, err, nil)
		return
	}

	if err := os.MkdirAll(r.Paths.ApplicationsDir, 0755); err != nil {
		emit(StepWriteDesktop, 5, 0.85, "Failed creating applications directory", false, err, nil)
		return
	}

	desktopPath := filepath.Join(r.Paths.ApplicationsDir, fmt.Sprintf("plumcake-%s.desktop", appID))
	desktopContent := writeDesktopContent(desktopMeta, appID, wrapperPath)
	if err := os.WriteFile(desktopPath, []byte(desktopContent), 0644); err != nil {
		emit(StepWriteDesktop, 5, 0.85, "Failed creating desktop file", false, err, nil)
		return
	}

	fileInfo, _ := os.Stat(destBinaryPath)
	var binarySize int64
	if fileInfo != nil {
		binarySize = fileInfo.Size()
	}

	now := time.Now().Format(time.RFC3339)
	createdApp := &registry.App{
		SchemaVersion:    2,
		ID:               appID,
		Name:             desktopMeta.Name,
		Version:          desktopMeta.Version,
		ExecName:         execName,
		WMClass:          desktopMeta.WMClass,
		Comment:          desktopMeta.Comment,
		Categories:       desktopMeta.Categories,
		InstalledAt:      now,
		UpdatedAt:        now,
		AppImagePath:     destBinaryPath,
		OriginalFilename: filepath.Base(localAppImagePath),
		BinarySHA256:     trustResult.ActualHash,
		BinarySize:       binarySize,
		DesktopFile:      desktopPath,
		IconFiles:        iconFiles,
		WrapperFile:      wrapperPath,
		Portable:         portable,
		Terminal:         desktopMeta.Terminal,
		MimeTypes:        desktopMeta.MimeTypes,
		Trust:            trustResult.State,
		Checksum:         trustResult.ActualHash,
		Signature:        trustResult.SignatureKey,
		Sandbox: registry.SandboxConfig{
			Enabled:      false,
			Tool:         "off",
			Network:      true,
			OriginalExec: wrapperPath + " %U",
		},
	}

	if isRemote {
		createdApp.SourceURL = &source
		createdApp.UpdateInfo.SourceURL = source
		if strings.Contains(source, "github.com") {
			createdApp.UpdateInfo.SourceType = "github"
		} else {
			createdApp.UpdateInfo.SourceType = "url"
		}
	}

	apps, _ := registry.LoadAll(r.Paths)
	for _, existing := range apps {
		if existing.ID == appID {
			createdApp.VersionHistory = existing.VersionHistory
			createdApp.Sandbox = existing.Sandbox
			break
		}
	}

	if err := registry.Save(r.Paths, createdApp); err != nil {
		emit(StepWriteDesktop, 5, 0.85, "Failed saving application registry manifest", false, err, nil)
		return
	}
	_ = updater.ArchiveVersion(r.Paths, createdApp, 2)

	emit(StepRefreshAppMenu, 6, 0.95, "Refreshing Freedesktop databases & icon caches...", false, nil, nil)
	registry.RefreshDesktopDatabase(r.Paths)
	registry.RefreshIconCache(r.Paths)

	if !isRemote && moveSource && createdApp != nil {
		if strings.HasPrefix(source, r.Paths.Home) && !strings.HasPrefix(source, r.Paths.AppsDir) {
			if !strings.HasPrefix(source, "/usr") && !strings.HasPrefix(source, "/opt") {
				_ = os.Remove(source)
			}
		}
	}

	history.Record(r.Paths, "install", createdApp.ID, "success",
		fmt.Sprintf("Added %s (v%s, trust: %s)", createdApp.Name, createdApp.Version, createdApp.Trust))

	emit(StepRefreshAppMenu, 6, 1.0, "Complete!", true, nil, createdApp)
}

type ExtractedMetadata struct {
	Name       string
	Exec       string
	Icon       string
	Comment    string
	Categories []string
	Terminal   bool
	MimeTypes  []string
	WMClass    string
	Version    string
}

var versionPattern = regexp.MustCompile(`(?:^|[-_vV])([0-9]+(?:\.[0-9]+)+(?:\.[0-9]+)?)`)

func extractVersion(filename string) string {
	clean := strings.TrimSuffix(strings.TrimSuffix(filename, ".AppImage"), ".appimage")
	match := versionPattern.FindStringSubmatch(clean)
	if len(match) > 1 && match[1] != "" {
		return match[1]
	}
	return "1.0.0"
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r == '-' || r == '_' || r == ' ' || r == '.' {
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "-") {
				sb.WriteRune('-')
			}
		}
	}
	return strings.Trim(sb.String(), "-")
}

func resolveSquashPath(squashRoot, path string) string {
	current := path
	for range 10 {
		info, err := os.Lstat(current)
		if err != nil || (info.Mode()&os.ModeSymlink == 0) {
			break
		}
		target, err := os.Readlink(current)
		if err != nil {
			break
		}
		if filepath.IsAbs(target) {
			current = filepath.Join(squashRoot, strings.TrimPrefix(target, "/"))
		} else {
			current = filepath.Join(filepath.Dir(current), target)
		}
	}
	return current
}

func parseExtractedDesktop(squashRoot, localAppImagePath string) ExtractedMetadata {
	meta := ExtractedMetadata{
		Version: extractVersion(filepath.Base(localAppImagePath)),
	}

	var candidates []string
	if entries, err := os.ReadDir(squashRoot); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".desktop") {
				candidates = append(candidates, filepath.Join(squashRoot, entry.Name()))
			}
		}
	}
	shareDir := filepath.Join(squashRoot, "usr", "share", "applications")
	if entries, err := os.ReadDir(shareDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".desktop") {
				candidates = append(candidates, filepath.Join(shareDir, entry.Name()))
			}
		}
	}

	for _, cand := range candidates {
		resolved := resolveSquashPath(squashRoot, cand)
		data, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		inDesktopEntry := false
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") {
				inDesktopEntry = line == "[Desktop Entry]"
				continue
			}
			if !inDesktopEntry || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.Trim(strings.TrimSpace(v), "\"'")
			switch k {
			case "Name":
				if meta.Name == "" {
					meta.Name = v
				}
			case "Exec":
				if meta.Exec == "" {
					meta.Exec = v
				}
			case "Icon":
				if meta.Icon == "" {
					meta.Icon = v
				}
			case "Comment":
				if meta.Comment == "" {
					meta.Comment = v
				}
			case "Categories":
				for _, c := range strings.Split(v, ";") {
					c = strings.TrimSpace(c)
					if c != "" {
						meta.Categories = append(meta.Categories, c)
					}
				}
			case "Terminal":
				meta.Terminal = v == "true" || v == "1"
			case "MimeType":
				for _, m := range strings.Split(v, ";") {
					m = strings.TrimSpace(m)
					if m != "" {
						meta.MimeTypes = append(meta.MimeTypes, m)
					}
				}
			case "StartupWMClass":
				if meta.WMClass == "" {
					meta.WMClass = v
				}
			case "X-AppImage-Version":
				if v != "" {
					meta.Version = v
				}
			}
		}
		if meta.Name != "" {
			break
		}
	}

	if meta.Name == "" {
		base := filepath.Base(localAppImagePath)
		base = strings.TrimSuffix(strings.TrimSuffix(base, ".AppImage"), ".appimage")
		meta.Name = base
	}
	if meta.WMClass == "" {
		meta.WMClass = slugify(meta.Name)
	}

	return meta
}

func installExtractedIcons(pathsInstance *paths.Paths, squashRoot, appID, iconHint string) []string {
	if squashRoot == "" {
		return nil
	}
	if _, err := os.Stat(squashRoot); os.IsNotExist(err) {
		return nil
	}

	var candidates []string
	_ = filepath.Walk(squashRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		lower := strings.ToLower(p)
		base := strings.ToLower(info.Name())
		if strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".svg") || base == ".diricon" {
			candidates = append(candidates, p)
		}
		return nil
	})

	var chosenPrimary string
	var chosenIsSVG bool
	var chosenIsPNG bool

	matchHints := []string{}
	if iconHint != "" {
		cleanHint := strings.ToLower(strings.TrimSpace(iconHint))
		matchHints = append(matchHints, cleanHint+".svg", cleanHint+".png", cleanHint)
	}

	findMatching := func(pattern string) string {
		for _, cand := range candidates {
			base := strings.ToLower(filepath.Base(cand))
			if base == pattern {
				resolved := resolveSquashPath(squashRoot, cand)
				isPNG, isSVG, err := registry.ValidateImageFile(resolved)
				if err == nil && (isPNG || isSVG) {
					chosenIsPNG = isPNG
					chosenIsSVG = isSVG
					return resolved
				}
			}
		}
		return ""
	}

	for _, hint := range matchHints {
		if matched := findMatching(hint); matched != "" {
			chosenPrimary = matched
			break
		}
	}

	if chosenPrimary == "" {
		dirIconPath := filepath.Join(squashRoot, ".DirIcon")
		resolved := resolveSquashPath(squashRoot, dirIconPath)
		isPNG, isSVG, err := registry.ValidateImageFile(resolved)
		if err == nil && (isPNG || isSVG) {
			chosenPrimary = resolved
			chosenIsPNG = isPNG
			chosenIsSVG = isSVG
		}
	}

	if chosenPrimary == "" {
		for _, cand := range candidates {
			resolved := resolveSquashPath(squashRoot, cand)
			_, isSVG, err := registry.ValidateImageFile(resolved)
			if err == nil && isSVG {
				chosenPrimary = resolved
				chosenIsSVG = true
				break
			}
		}
	}

	if chosenPrimary == "" {
		for _, cand := range candidates {
			resolved := resolveSquashPath(squashRoot, cand)
			isPNG, _, err := registry.ValidateImageFile(resolved)
			if err == nil && isPNG {
				chosenPrimary = resolved
				chosenIsPNG = true
				break
			}
		}
	}

	if chosenPrimary == "" {
		return nil
	}

	iconIdentifier := "plumcake-" + appID
	var installedIcons []string

	if chosenIsSVG {
		targetScalable := filepath.Join(pathsInstance.IconsDir, "hicolor", "scalable", "apps", iconIdentifier+".svg")
		targetPixmap := filepath.Join(pathsInstance.PixmapsDir, iconIdentifier+".svg")
		_ = os.MkdirAll(filepath.Dir(targetScalable), 0755)
		_ = os.MkdirAll(filepath.Dir(targetPixmap), 0755)
		if err := registry.CopyFile(chosenPrimary, targetScalable); err == nil {
			installedIcons = append(installedIcons, targetScalable)
		}
		if err := registry.CopyFile(chosenPrimary, targetPixmap); err == nil {
			installedIcons = append(installedIcons, targetPixmap)
		}
	} else if chosenIsPNG {
		targetPNG := filepath.Join(pathsInstance.IconsDir, "hicolor", "256x256", "apps", iconIdentifier+".png")
		targetPixmap := filepath.Join(pathsInstance.PixmapsDir, iconIdentifier+".png")
		_ = os.MkdirAll(filepath.Dir(targetPNG), 0755)
		_ = os.MkdirAll(filepath.Dir(targetPixmap), 0755)
		if err := registry.CopyFile(chosenPrimary, targetPNG); err == nil {
			installedIcons = append(installedIcons, targetPNG)
		}
		if err := registry.CopyFile(chosenPrimary, targetPixmap); err == nil {
			installedIcons = append(installedIcons, targetPixmap)
		}

		for _, cand := range candidates {
			resolved := resolveSquashPath(squashRoot, cand)
			isPNG, _, err := registry.ValidateImageFile(resolved)
			if err != nil || !isPNG {
				continue
			}
			parts := strings.Split(resolved, string(filepath.Separator))
			for i, p := range parts {
				if p == "hicolor" && i+2 < len(parts) && parts[i+2] == "apps" {
					size := parts[i+1]
					targetSizeDir := filepath.Join(pathsInstance.IconsDir, "hicolor", size, "apps")
					_ = os.MkdirAll(targetSizeDir, 0755)
					destPath := filepath.Join(targetSizeDir, iconIdentifier+".png")
					if err := registry.CopyFile(resolved, destPath); err == nil {
						installedIcons = append(installedIcons, destPath)
					}
					break
				}
			}
		}
	}

	return installedIcons
}

func writeDesktopContent(meta ExtractedMetadata, appID, wrapperPath string) string {
	var sb strings.Builder
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Type=Application\n")
	sb.WriteString(fmt.Sprintf("Name=%s\n", meta.Name))
	sb.WriteString(fmt.Sprintf("Exec=%s %%U\n", wrapperPath))
	sb.WriteString(fmt.Sprintf("Icon=plumcake-%s\n", appID))
	if meta.Comment != "" {
		sb.WriteString(fmt.Sprintf("Comment=%s\n", meta.Comment))
	}
	if meta.Terminal {
		sb.WriteString("Terminal=true\n")
	} else {
		sb.WriteString("Terminal=false\n")
	}
	if len(meta.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("Categories=%s;\n", strings.Join(meta.Categories, ";")))
	}
	if meta.WMClass != "" {
		sb.WriteString(fmt.Sprintf("StartupWMClass=%s\n", meta.WMClass))
	}
	if len(meta.MimeTypes) > 0 {
		sb.WriteString(fmt.Sprintf("MimeType=%s;\n", strings.Join(meta.MimeTypes, ";")))
	}
	sb.WriteString("X-Plumcake-Managed=true\n")
	sb.WriteString(fmt.Sprintf("X-Plumcake-ID=%s\n", appID))
	if meta.Version != "" {
		sb.WriteString(fmt.Sprintf("X-Plumcake-Version=%s\n", meta.Version))
	}
	return sb.String()
}
