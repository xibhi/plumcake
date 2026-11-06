package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func TestManifestExportImportRoundTrip(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		AppsDir: filepath.Join(tempDirectory, "apps"),
	}

	appDirectory := filepath.Join(pathsInstance.AppsDir, "cooltool")
	_ = os.MkdirAll(appDirectory, 0755)

	originalURL := "https://github.com/cool/tool/releases/download/v1.2.0/tool.AppImage"
	app := registry.App{
		SchemaVersion:    2,
		ID:               "cooltool",
		Name:             "Cool Tool",
		Version:          "1.2.0",
		SourceURL:        &originalURL,
		Checksum:         "abc123sha256",
		OriginalFilename: "tool.AppImage",
		Sandbox: registry.SandboxConfig{
			Enabled: true,
			Tool:    "bwrap",
			Network: false,
		},
	}
	appData, _ := json.Marshal(app)
	_ = os.WriteFile(filepath.Join(appDirectory, "app.json"), appData, 0644)

	exportFile := filepath.Join(tempDirectory, "exported-manifest.toml")
	if err := Export(pathsInstance, exportFile); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	parsedManifest, err := Parse(exportFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsedManifest.Apps) != 1 {
		t.Fatalf("expected 1 exported app, got %d", len(parsedManifest.Apps))
	}

	exported := parsedManifest.Apps[0]
	if exported.ID != "cooltool" || exported.Name != "Cool Tool" || exported.Version != "1.2.0" {
		t.Fatalf("mismatched exported fields: %+v", exported)
	}
	if exported.SandboxTool != "bwrap" || exported.SandboxNet != false {
		t.Fatalf("mismatched sandbox fields: %+v", exported)
	}

	planSame, err := Plan(pathsInstance, exportFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(planSame.AlreadyPresent) != 1 {
		t.Fatalf("expected AlreadyPresent == 1, got %d", len(planSame.AlreadyPresent))
	}
	if len(planSame.WillInstall) != 0 {
		t.Fatalf("expected WillInstall == 0, got %d", len(planSame.WillInstall))
	}

	app.Version = "1.1.0"
	appData2, _ := json.Marshal(app)
	_ = os.WriteFile(filepath.Join(appDirectory, "app.json"), appData2, 0644)

	planUpgrade, err := Plan(pathsInstance, exportFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(planUpgrade.WillInstall) != 1 {
		t.Fatalf("expected WillInstall == 1 for upgrade, got %d", len(planUpgrade.WillInstall))
	}
}
