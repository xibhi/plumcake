package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xibhi/plumcake/internal/core/appimage"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

type Result struct {
	Path        string
	Name        string
	Size        int64
	Folder      string
	ModTime     time.Time
	Arch        string
	Type        string
	IsAdded     bool
	ExistingApp *registry.App
}

func ShouldSkip(path string, home string, appsDir string) bool {
	clean := filepath.Clean(path)

	systemSkips := []string{
		"/proc", "/sys", "/dev", "/run", "/var",
		"/var/lib/flatpak", "/var/lib/snapd", "/snap",
	}
	for _, skip := range systemSkips {
		if clean == skip || strings.HasPrefix(clean, skip+string(filepath.Separator)) {
			return true
		}
	}

	if (clean == "/tmp" || strings.HasPrefix(clean, "/tmp"+string(filepath.Separator))) && (home == "" || !strings.HasPrefix(clean, home)) {
		return true
	}
	if appsDir != "" && (clean == appsDir || strings.HasPrefix(clean, appsDir+string(filepath.Separator))) {
		return true
	}

	oldAdoptApps := filepath.Join(home, ".local", "share", "adopt", "apps")
	if clean == oldAdoptApps || strings.HasPrefix(clean, oldAdoptApps+string(filepath.Separator)) {
		return true
	}

	base := filepath.Base(clean)
	if base == ".git" || base == "node_modules" || base == "flatpak" || base == ".cache" || base == ".snapshots" {
		return true
	}

	if home != "" {
		flatpakUser := filepath.Join(home, ".local", "share", "flatpak")
		if clean == flatpakUser || strings.HasPrefix(clean, flatpakUser+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

func ScanInitial(ctx context.Context, pathsInstance *paths.Paths, addedMap map[string]*registry.App, out chan<- Result) {
	defer close(out)

	prioritizedDirs := []string{
		".",
		filepath.Join(pathsInstance.Home, "Downloads"),
		filepath.Join(pathsInstance.Home, "Desktop"),
		filepath.Join(pathsInstance.Home, "Applications"),
		filepath.Join(pathsInstance.Home, ".local", "bin"),
		pathsInstance.Home,
		"/opt",
		"/usr/local/bin",
	}

	seen := make(map[string]bool)

	for _, dir := range prioritizedDirs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if ShouldSkip(dir, pathsInstance.Home, pathsInstance.AppsDir) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".appimage") {
				continue
			}

			fullPath := filepath.Join(dir, name)
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				absPath = fullPath
			}

			if seen[absPath] {
				continue
			}
			seen[absPath] = true

			valid, arch, appType, err := appimage.Validate(absPath)
			if !valid || err != nil {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			var existing *registry.App
			if app, ok := addedMap[absPath]; ok {
				existing = app
			} else if app, ok := addedMap[name]; ok {
				existing = app
			}

			out <- Result{
				Path:        absPath,
				Name:        name,
				Size:        info.Size(),
				Folder:      filepath.Dir(absPath),
				ModTime:     info.ModTime(),
				Arch:        arch,
				Type:        appType,
				IsAdded:     existing != nil,
				ExistingApp: existing,
			}
		}
	}
}

func ScanDeep(ctx context.Context, pathsInstance *paths.Paths, addedMap map[string]*registry.App, seen map[string]bool, out chan<- Result) {
	defer close(out)

	_ = filepath.WalkDir(pathsInstance.Home, func(p string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return fs.SkipAll
		default:
		}

		if err != nil {
			return nil
		}

		if d.IsDir() {
			if ShouldSkip(p, pathsInstance.Home, pathsInstance.AppsDir) {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".appimage") {
			return nil
		}

		absPath, err := filepath.Abs(p)
		if err != nil {
			absPath = p
		}

		if seen[absPath] {
			return nil
		}
		seen[absPath] = true

		valid, arch, appType, err := appimage.Validate(absPath)
		if !valid || err != nil {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		var existing *registry.App
		if app, ok := addedMap[absPath]; ok {
			existing = app
		} else if app, ok := addedMap[name]; ok {
			existing = app
		}

		out <- Result{
			Path:        absPath,
			Name:        name,
			Size:        info.Size(),
			Folder:      filepath.Dir(absPath),
			ModTime:     info.ModTime(),
			Arch:        arch,
			Type:        appType,
			IsAdded:     existing != nil,
			ExistingApp: existing,
		}
		return nil
	})
}

func FuzzyMatch(pattern, text string) bool {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	if pattern == "" {
		return true
	}

	pIdx := 0
	pLen := len(pattern)

	for i := range len(text) {
		if text[i] == pattern[pIdx] {
			pIdx++
			if pIdx == pLen {
				return true
			}
		}
	}
	return false
}
