package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	tempDirectory := t.TempDir()
	originalConfig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalConfig)

	os.Setenv("XDG_CONFIG_HOME", tempDirectory)

	initialConfig := Load()
	if initialConfig.Theme != "system" {
		t.Fatalf("expected default system theme, got %s", initialConfig.Theme)
	}

	initialConfig.Theme = "nord"
	initialConfig.ReducedMotion = true
	initialConfig.KeepVersions = 4

	if err := Save(initialConfig); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	reloadedConfig := Load()
	if reloadedConfig.Theme != "nord" {
		t.Fatalf("expected nord theme, got %s", reloadedConfig.Theme)
	}
	if !reloadedConfig.ReducedMotion {
		t.Fatalf("expected reduced motion true")
	}
	if reloadedConfig.KeepVersions != 4 {
		t.Fatalf("expected keep versions 4, got %d", reloadedConfig.KeepVersions)
	}

	configPath := filepath.Join(tempDirectory, "plumcake", "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}
