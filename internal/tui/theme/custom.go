package theme

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Directory(configHome string) string {
	if configHome == "" {
		configHome = os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			home, _ := os.UserHomeDir()
			configHome = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(configHome, "plumcake", "themes")
}

func Load(configHome string) ([]Palette, []string) {
	dir := Directory(configHome)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		oldDir := filepath.Join(filepath.Dir(dir), "adopt", "themes")
		if _, oldErr := os.Stat(oldDir); oldErr == nil {
			dir = oldDir
		} else {
			return nil, nil
		}
	}

	var customThemes []Palette
	var warnings []string

	builtinIdentifiers := make(map[string]bool)
	for _, t := range BuiltinThemes {
		builtinIdentifiers[strings.ToLower(t.ID)] = true
	}

	_ = filepath.WalkDir(dir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".toml") {
			return nil
		}

		themeKey := strings.TrimSuffix(strings.ToLower(entry.Name()), ".toml")
		if builtinIdentifiers[themeKey] {
			warnings = append(warnings, fmt.Sprintf("%s: collision with built-in theme id '%s', skipped", entry.Name(), themeKey))
			return nil
		}

		parsedPalette, parseErr := ParseCustomFile(filePath, themeKey)
		if parseErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", entry.Name(), parseErr))
			return nil
		}

		customThemes = append(customThemes, *parsedPalette)
		return nil
	})

	return customThemes, warnings
}

func ParseCustomFile(filePath, themeID string) (*Palette, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	palette := &Palette{
		ID:       themeID,
		Name:     themeID,
		IsCustom: true,
	}

	fieldMap := make(map[string]string)
	lines := strings.Split(string(fileData), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			fieldMap[k] = v
		}
	}

	if nameVal, exists := fieldMap["name"]; exists && nameVal != "" {
		palette.Name = nameVal
	}
	if lightVal, exists := fieldMap["light"]; exists {
		palette.IsLight = lightVal == "true" || lightVal == "1"
	}

	requiredRoles := []string{"bg", "surface", "text", "muted", "accent", "success", "warning", "error", "border", "selection"}
	for _, role := range requiredRoles {
		hexValue, exists := fieldMap[role]
		if !exists || hexValue == "" {
			return nil, fmt.Errorf("missing required semantic role '%s'", role)
		}
		if !strings.HasPrefix(hexValue, "#") || len(hexValue) != 7 {
			return nil, fmt.Errorf("invalid hex value for '%s': '%s' (expected #RRGGBB)", role, hexValue)
		}
	}

	palette.BG = lipgloss.Color(fieldMap["bg"])
	palette.Surface = lipgloss.Color(fieldMap["surface"])
	palette.Text = lipgloss.Color(fieldMap["text"])
	palette.Muted = lipgloss.Color(fieldMap["muted"])
	palette.Accent = lipgloss.Color(fieldMap["accent"])
	palette.Success = lipgloss.Color(fieldMap["success"])
	palette.Warning = lipgloss.Color(fieldMap["warning"])
	palette.Error = lipgloss.Color(fieldMap["error"])
	palette.Border = lipgloss.Color(fieldMap["border"])
	palette.Selection = lipgloss.Color(fieldMap["selection"])

	return palette, nil
}
