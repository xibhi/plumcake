package manifest

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

type AppEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Version     string `json:"version"`
	Checksum    string `json:"checksum"`
	CustomName  string `json:"custom_name,omitempty"`
	SandboxTool string `json:"sandbox_tool,omitempty"`
	SandboxNet  bool   `json:"sandbox_net,omitempty"`
}

type Manifest struct {
	ToolName   string     `json:"tool_name"`
	Version    string     `json:"version"`
	ExportedAt string     `json:"exported_at"`
	Apps       []AppEntry `json:"apps"`
}

type ImportPlan struct {
	WillInstall    []AppEntry
	AlreadyPresent []AppEntry
	Skipped        []AppEntry
}

func Export(pathsInstance *paths.Paths, destinationFilePath string) error {
	apps, err := registry.LoadAll(pathsInstance)
	if err != nil {
		return err
	}

	var entries []AppEntry
	for _, app := range apps {
		sourceAddress := ""
		if app.SourceURL != nil && *app.SourceURL != "" {
			sourceAddress = *app.SourceURL
		} else if app.UpdateInfo.SourceURL != "" {
			sourceAddress = app.UpdateInfo.SourceURL
		} else {
			sourceAddress = app.OriginalFilename
		}

		entries = append(entries, AppEntry{
			ID:          app.ID,
			Name:        app.Name,
			Source:      sourceAddress,
			Version:     app.Version,
			Checksum:    app.Checksum,
			CustomName:  app.Name,
			SandboxTool: app.Sandbox.Tool,
			SandboxNet:  app.Sandbox.Network,
		})
	}

	var tomlContent strings.Builder
	tomlContent.WriteString(fmt.Sprintf("tool = \"plumcake\"\nversion = \"1.0.1\"\nexported_at = \"%s\"\n\n",
		time.Now().Format(time.RFC3339)))

	for _, entry := range entries {
		tomlContent.WriteString("[[apps]]\n")
		tomlContent.WriteString(fmt.Sprintf("id = \"%s\"\n", entry.ID))
		tomlContent.WriteString(fmt.Sprintf("name = \"%s\"\n", entry.Name))
		tomlContent.WriteString(fmt.Sprintf("source = \"%s\"\n", entry.Source))
		tomlContent.WriteString(fmt.Sprintf("version = \"%s\"\n", entry.Version))
		tomlContent.WriteString(fmt.Sprintf("checksum = \"%s\"\n", entry.Checksum))
		if entry.SandboxTool != "" && entry.SandboxTool != "off" {
			tomlContent.WriteString(fmt.Sprintf("sandbox_tool = \"%s\"\n", entry.SandboxTool))
			tomlContent.WriteString(fmt.Sprintf("sandbox_net = %t\n", entry.SandboxNet))
		}
		tomlContent.WriteString("\n")
	}

	return os.WriteFile(destinationFilePath, []byte(tomlContent.String()), 0644)
}

func Parse(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	manifest := &Manifest{
		ToolName:   "plumcake",
		Version:    "1.0.1",
		ExportedAt: time.Now().Format(time.RFC3339),
	}

	var current *AppEntry
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[[apps]]" {
			if current != nil {
				manifest.Apps = append(manifest.Apps, *current)
			}
			current = &AppEntry{}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		if current == nil {
			switch key {
			case "tool", "tool_name":
				manifest.ToolName = val
			case "version", "adopt_version", "plumcake_version":
				manifest.Version = val
			}
			continue
		}

		switch key {
		case "id":
			current.ID = val
		case "name":
			current.Name = val
		case "source":
			current.Source = val
		case "version":
			current.Version = val
		case "checksum":
			current.Checksum = val
		case "sandbox_tool":
			current.SandboxTool = val
		case "sandbox_net":
			current.SandboxNet = val == "true" || val == "1"
		}
	}

	if current != nil && current.ID != "" {
		manifest.Apps = append(manifest.Apps, *current)
	}

	return manifest, nil
}

func Plan(pathsInstance *paths.Paths, manifestFilePath string) (*ImportPlan, error) {
	man, err := Parse(manifestFilePath)
	if err != nil {
		return nil, err
	}

	apps, _ := registry.LoadAll(pathsInstance)
	installedMap := make(map[string]*registry.App)
	for i := range apps {
		installedMap[apps[i].ID] = &apps[i]
	}

	plan := &ImportPlan{}
	for _, entry := range man.Apps {
		if entry.Source == "" {
			plan.Skipped = append(plan.Skipped, entry)
			continue
		}

		if existing, ok := installedMap[entry.ID]; ok {
			if existing.Version == entry.Version {
				plan.AlreadyPresent = append(plan.AlreadyPresent, entry)
			} else {
				plan.WillInstall = append(plan.WillInstall, entry)
			}
		} else {
			plan.WillInstall = append(plan.WillInstall, entry)
		}
	}

	return plan, nil
}
