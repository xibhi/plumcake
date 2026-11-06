package paths

import (
	"os"
	"path/filepath"
)

type Paths struct {
	Home            string
	DataHome        string
	ConfigHome      string
	CacheHome       string
	StateHome       string
	PlumcakeHome    string
	AppsDir         string
	BinDir          string
	ApplicationsDir string
	IconsDir        string
	PixmapsDir      string
	BackendPy       string
	OldAdoptHome    string
	OldAdoptConfig  string
	OldAdoptState   string
}

func New() *Paths {
	home, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		cacheHome = filepath.Join(home, ".cache")
	}
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}

	plumcakeHome := filepath.Join(dataHome, "plumcake")
	appsDir := filepath.Join(plumcakeHome, "apps")
	binDir := filepath.Join(home, ".local", "bin")
	applicationsDir := filepath.Join(dataHome, "applications")
	iconsDir := filepath.Join(dataHome, "icons")
	pixmapsDir := filepath.Join(dataHome, "pixmaps")

	oldAdoptHome := filepath.Join(dataHome, "adopt")
	oldAdoptConfig := filepath.Join(configHome, "adopt")
	oldAdoptState := filepath.Join(stateHome, "adopt")

	backendPy := os.Getenv("PLUMCAKE_BACKEND_PY")

	return &Paths{
		Home:            home,
		DataHome:        dataHome,
		ConfigHome:      configHome,
		CacheHome:       cacheHome,
		StateHome:       stateHome,
		PlumcakeHome:    plumcakeHome,
		AppsDir:         appsDir,
		BinDir:          binDir,
		ApplicationsDir: applicationsDir,
		IconsDir:        iconsDir,
		PixmapsDir:      pixmapsDir,
		BackendPy:       backendPy,
		OldAdoptHome:    oldAdoptHome,
		OldAdoptConfig:  oldAdoptConfig,
		OldAdoptState:   oldAdoptState,
	}
}
