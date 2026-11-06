package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinThemesRegistry(t *testing.T) {
	if len(BuiltinThemes) != 12 {
		t.Fatalf("expected exactly 12 built-in themes, got %d", len(BuiltinThemes))
	}

	expectedOrder := []string{
		"system",
		"catppuccin-mocha",
		"catppuccin-macchiato",
		"catppuccin-latte",
		"tokyo-night",
		"gruvbox-dark",
		"nord",
		"dracula",
		"rose-pine",
		"kanagawa",
		"everforest",
		"one-dark",
	}

	seenIDs := make(map[string]bool)
	for i, themeInstance := range BuiltinThemes {
		if themeInstance.ID != expectedOrder[i] {
			t.Errorf("theme index %d: expected ID '%s', got '%s'", i, expectedOrder[i], themeInstance.ID)
		}
		if seenIDs[themeInstance.ID] {
			t.Errorf("duplicate theme ID: %s", themeInstance.ID)
		}
		seenIDs[themeInstance.ID] = true

		if themeInstance.ID == "solarized-dark" {
			t.Errorf("solarized-dark must not be in the built-in theme registry")
		}
	}

	system := BuiltinThemes[0]
	if !system.IsSystem {
		t.Errorf("theme 0 must be marked IsSystem")
	}
	roles := []string{
		string(system.BG), string(system.Surface), string(system.Text),
		string(system.Muted), string(system.Accent), string(system.Success),
		string(system.Warning), string(system.Error), string(system.Border),
		string(system.Selection),
	}
	for _, r := range roles {
		if strings.HasPrefix(r, "#") {
			t.Errorf("System theme must not use hex colors, found: %s", r)
		}
	}
}

func TestWCAGContrastRatios(t *testing.T) {
	for _, themeInstance := range BuiltinThemes {
		if themeInstance.IsSystem {
			continue
		}

		bg := string(themeInstance.BG)

		textRatio := ContrastRatio(string(themeInstance.Text), bg)
		if textRatio < 4.5 {
			t.Errorf("Theme %s: text contrast ratio %.2f < 4.5:1 (text: %s, bg: %s)",
				themeInstance.Name, textRatio, themeInstance.Text, bg)
		}

		mutedRatio := ContrastRatio(string(themeInstance.Muted), bg)
		if mutedRatio < 3.0 {
			t.Errorf("Theme %s: muted contrast ratio %.2f < 3.0:1 (muted: %s, bg: %s)",
				themeInstance.Name, mutedRatio, themeInstance.Muted, bg)
		}

		accentRatio := ContrastRatio(string(themeInstance.Accent), bg)
		if accentRatio < 3.0 {
			t.Errorf("Theme %s: accent contrast ratio %.2f < 3.0:1 (accent: %s, bg: %s)",
				themeInstance.Name, accentRatio, themeInstance.Accent, bg)
		}

		successRatio := ContrastRatio(string(themeInstance.Success), bg)
		if successRatio < 3.0 {
			t.Errorf("Theme %s: success contrast ratio %.2f < 3.0:1 (success: %s, bg: %s)",
				themeInstance.Name, successRatio, themeInstance.Success, bg)
		}

		warningRatio := ContrastRatio(string(themeInstance.Warning), bg)
		if warningRatio < 3.0 {
			t.Errorf("Theme %s: warning contrast ratio %.2f < 3.0:1 (warning: %s, bg: %s)",
				themeInstance.Name, warningRatio, themeInstance.Warning, bg)
		}

		errorRatio := ContrastRatio(string(themeInstance.Error), bg)
		if errorRatio < 3.0 {
			t.Errorf("Theme %s: error contrast ratio %.2f < 3.0:1 (error: %s, bg: %s)",
				themeInstance.Name, errorRatio, themeInstance.Error, bg)
		}
	}
}

func TestColorFallbackConversion(t *testing.T) {
	blackCode := HexToANSI256("#000000")
	if blackCode != 16 && blackCode != 0 {
		t.Errorf("expected black to map to 16 or 0, got %d", blackCode)
	}

	whiteCode := HexToANSI256("#ffffff")
	if whiteCode != 231 && whiteCode != 15 {
		t.Errorf("expected white to map to 231 or 15, got %d", whiteCode)
	}

	origColorTerm := os.Getenv("COLORTERM")
	defer os.Setenv("COLORTERM", origColorTerm)

	os.Setenv("COLORTERM", "")
	resolved := Resolve("#cba6f7")
	if strings.HasPrefix(string(resolved), "#") {
		t.Errorf("expected hex color to be degraded to ANSI256 code, got %s", resolved)
	}
}

func TestNoColorHandling(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", origNoColor)

	os.Setenv("NO_COLOR", "1")
	if !IsNoColor() {
		t.Fatalf("expected IsNoColor() == true when NO_COLOR=1")
	}

	resolved := Resolve("#cba6f7")
	if string(resolved) != "" {
		t.Errorf("expected empty color under NO_COLOR, got %s", resolved)
	}
}

func TestUnknownThemeIDFallback(t *testing.T) {
	t1 := Find("solarized-dark")
	if t1.ID != "system" {
		t.Errorf("expected solarized-dark to fall back to system, got %s", t1.ID)
	}

	t2 := Find("nonexistent-theme-404")
	if t2.ID != "system" {
		t.Errorf("expected nonexistent theme to fall back to system, got %s", t2.ID)
	}

	t3 := Find("catppuccin-latte")
	if t3.ID != "catppuccin-latte" {
		t.Errorf("expected catppuccin-latte, got %s", t3.ID)
	}
}

func TestCustomThemeLoading(t *testing.T) {
	tempDirectory := t.TempDir()
	origConfig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origConfig)

	os.Setenv("XDG_CONFIG_HOME", tempDirectory)
	themesDirectory := filepath.Join(tempDirectory, "plumcake", "themes")
	_ = os.MkdirAll(themesDirectory, 0755)

	validContent := `name = "My Neon"
light = false
bg = "#0a0a0a"
surface = "#1a1a1a"
text = "#ffffff"
muted = "#888888"
accent = "#00ffcc"
success = "#00ff00"
warning = "#ffcc00"
error = "#ff0033"
border = "#333333"
selection = "#222222"
`
	_ = os.WriteFile(filepath.Join(themesDirectory, "my-neon.toml"), []byte(validContent), 0644)

	missingContent := `name = "Broken"
bg = "#000000"
text = "#ffffff"
`
	_ = os.WriteFile(filepath.Join(themesDirectory, "broken.toml"), []byte(missingContent), 0644)

	badHexContent := `name = "Bad Hex"
bg = "not-a-hex"
surface = "#111111"
text = "#ffffff"
muted = "#888888"
accent = "#00ffcc"
success = "#00ff00"
warning = "#ffcc00"
error = "#ff0033"
border = "#333333"
selection = "#222222"
`
	_ = os.WriteFile(filepath.Join(themesDirectory, "bad-hex.toml"), []byte(badHexContent), 0644)

	collisionContent := validContent
	_ = os.WriteFile(filepath.Join(themesDirectory, "dracula.toml"), []byte(collisionContent), 0644)

	customThemes, warnings := Load("")

	if len(customThemes) != 1 {
		t.Fatalf("expected 1 valid custom theme, got %d", len(customThemes))
	}
	if customThemes[0].Name != "My Neon" || customThemes[0].ID != "my-neon" {
		t.Fatalf("unexpected custom theme parsed: %+v", customThemes[0])
	}

	if len(warnings) < 3 {
		t.Fatalf("expected at least 3 warnings, got %d", len(warnings))
	}

	all := GetAll()
	if len(all) != 13 {
		t.Fatalf("expected 12 built-in + 1 custom = 13 themes, got %d", len(all))
	}
}
