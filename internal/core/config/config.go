package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Theme               string `toml:"theme"`
	InstallDir          string `toml:"install_dir"`
	ReducedMotion       bool   `toml:"reduced_motion"`
	DeepScan            bool   `toml:"deep_scan"`
	ConfirmDelete       bool   `toml:"confirm_delete"`
	PreinstallInspect   bool   `toml:"preinstall_inspect"`
	CheckUpdatesOnStart bool   `toml:"check_updates_on_start"`
	IncludePrereleases  bool   `toml:"include_prereleases"`
	KeepVersions        int    `toml:"keep_versions"`
	TrashRetentionDays  int    `toml:"trash_retention_days"`
	DefaultMoveFile     bool   `toml:"default_move_file"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	return Config{
		Theme:               "system",
		InstallDir:          filepath.Join(dataHome, "plumcake", "apps"),
		ReducedMotion:       false,
		DeepScan:            false,
		ConfirmDelete:       true,
		PreinstallInspect:   true,
		CheckUpdatesOnStart: false,
		IncludePrereleases:  false,
		KeepVersions:        2,
		TrashRetentionDays:  7,
		DefaultMoveFile:     true,
	}
}

func FilePath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "plumcake", "config.toml")
}

func Load() Config {
	cfg := Default()
	filePath := FilePath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		oldAdoptPath := filepath.Join(filepath.Dir(filepath.Dir(filePath)), "adopt", "config.toml")
		data, err = os.ReadFile(oldAdoptPath)
		if err != nil {
			return cfg
		}
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		switch key {
		case "theme":
			if val != "" {
				cfg.Theme = val
			}
		case "install_dir":
			if val != "" {
				cfg.InstallDir = strings.Replace(val, "/adopt/apps", "/plumcake/apps", 1)
			}
		case "reduced_motion":
			cfg.ReducedMotion = val == "true" || val == "1"
		case "deep_scan":
			cfg.DeepScan = val == "true" || val == "1"
		case "confirm_delete":
			cfg.ConfirmDelete = val == "true" || val == "1"
		case "preinstall_inspect":
			cfg.PreinstallInspect = val == "true" || val == "1"
		case "check_updates_on_start":
			cfg.CheckUpdatesOnStart = val == "true" || val == "1"
		case "include_prereleases":
			cfg.IncludePrereleases = val == "true" || val == "1"
		case "keep_versions":
			if number, err := strconv.Atoi(val); err == nil && number >= 1 {
				cfg.KeepVersions = number
			}
		case "trash_retention_days":
			if number, err := strconv.Atoi(val); err == nil && number >= 0 {
				cfg.TrashRetentionDays = number
			}
		case "default_move_file":
			cfg.DefaultMoveFile = val == "true" || val == "1"
		}
	}
	return cfg
}

func Save(cfg Config) error {
	filePath := FilePath()
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`theme = "%s"
install_dir = "%s"
reduced_motion = %t
deep_scan = %t
confirm_delete = %t
preinstall_inspect = %t
check_updates_on_start = %t
include_prereleases = %t
keep_versions = %d
trash_retention_days = %d
default_move_file = %t
`,
		cfg.Theme,
		cfg.InstallDir,
		cfg.ReducedMotion,
		cfg.DeepScan,
		cfg.ConfirmDelete,
		cfg.PreinstallInspect,
		cfg.CheckUpdatesOnStart,
		cfg.IncludePrereleases,
		cfg.KeepVersions,
		cfg.TrashRetentionDays,
		cfg.DefaultMoveFile,
	)

	return os.WriteFile(filePath, []byte(content), 0644)
}
