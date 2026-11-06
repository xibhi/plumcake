package installer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Nuza", "nuza"},
		{"My Cool App", "my-cool-app"},
		{"app_name_test", "app-name-test"},
		{"  spaced-out  ", "spaced-out"},
		{"", ""},
	}
	for _, tc := range cases {
		got := slugify(tc.in)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestExtractVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"nuza_2.2.0_amd64.AppImage", "2.2.0"},
		{"obsidian-1.5.3.AppImage", "1.5.3"},
		{"sample-v4.1.2.appimage", "4.1.2"},
		{"plain.AppImage", "1.0.0"},
	}
	for _, tc := range cases {
		got := extractVersion(tc.in)
		if got != tc.want {
			t.Errorf("extractVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseExtractedDesktop(t *testing.T) {
	tmp := t.TempDir()
	squashRoot := filepath.Join(tmp, "squashfs-root")
	_ = os.MkdirAll(squashRoot, 0755)

	content := "[Desktop Entry]\nName=Demo App\nExec=demoapp --flag\nIcon=demo\nComment=Test description\nCategories=Utility;Office;\nStartupWMClass=demo-wm\nX-AppImage-Version=3.4.1\nTerminal=true\n"
	_ = os.WriteFile(filepath.Join(squashRoot, "demo.desktop"), []byte(content), 0644)

	meta := parseExtractedDesktop(squashRoot, "demo-3.4.1.AppImage")
	if meta.Name != "Demo App" {
		t.Errorf("expected Name 'Demo App', got %q", meta.Name)
	}
	if meta.Exec != "demoapp --flag" {
		t.Errorf("expected Exec 'demoapp --flag', got %q", meta.Exec)
	}
	if meta.Icon != "demo" {
		t.Errorf("expected Icon 'demo', got %q", meta.Icon)
	}
	if meta.Comment != "Test description" {
		t.Errorf("expected Comment 'Test description', got %q", meta.Comment)
	}
	if meta.WMClass != "demo-wm" {
		t.Errorf("expected WMClass 'demo-wm', got %q", meta.WMClass)
	}
	if meta.Version != "3.4.1" {
		t.Errorf("expected Version '3.4.1', got %q", meta.Version)
	}
	if !meta.Terminal {
		t.Errorf("expected Terminal true")
	}
	if len(meta.Categories) != 2 || meta.Categories[0] != "Utility" {
		t.Errorf("unexpected Categories: %v", meta.Categories)
	}
}

func TestWriteDesktopContent(t *testing.T) {
	meta := ExtractedMetadata{
		Name:       "Test App",
		Comment:    "Test comment",
		Categories: []string{"Graphics"},
		WMClass:    "test-app",
		Version:    "1.2.3",
	}
	content := writeDesktopContent(meta, "test-app", "/usr/bin/wrapper")
	if !strings.Contains(content, "Name=Test App") {
		t.Errorf("missing Name: %s", content)
	}
	if !strings.Contains(content, "Exec=/usr/bin/wrapper %U") {
		t.Errorf("missing Exec: %s", content)
	}
	if !strings.Contains(content, "Icon=plumcake-test-app") {
		t.Errorf("missing Icon: %s", content)
	}
	if !strings.Contains(content, "X-Plumcake-Managed=true") {
		t.Errorf("missing X-Plumcake-Managed: %s", content)
	}
	if !strings.Contains(content, "X-Plumcake-ID=test-app") {
		t.Errorf("missing X-Plumcake-ID: %s", content)
	}
	if !strings.Contains(content, "X-Plumcake-Version=1.2.3") {
		t.Errorf("missing X-Plumcake-Version: %s", content)
	}
}

func TestRunnerPipeline(t *testing.T) {
	tmp := t.TempDir()
	pathsInstance := &paths.Paths{
		AppsDir:         filepath.Join(tmp, "apps"),
		BinDir:          filepath.Join(tmp, "bin"),
		ApplicationsDir: filepath.Join(tmp, "applications"),
		IconsDir:        filepath.Join(tmp, "icons"),
		PixmapsDir:      filepath.Join(tmp, "pixmaps"),
		StateHome:       tmp,
		Home:            tmp,
	}

	appImagePath := filepath.Join(tmp, "test-1.0.0.AppImage")
	header := make([]byte, 64)
	copy(header[0:4], []byte("\x7fELF"))
	header[8] = 0x41
	header[9] = 0x49
	header[10] = 0x02
	header[18] = 0x3E
	_ = os.WriteFile(appImagePath, header, 0755)

	runner := NewRunner(pathsInstance)
	ctx := context.Background()
	out := make(chan Event, 16)

	go runner.Run(ctx, appImagePath, false, false, "", out)

	var lastEvent Event
	for ev := range out {
		lastEvent = ev
		if ev.Err != nil {
			t.Fatalf("unexpected step error on %s: %v", ev.Step, ev.Err)
		}
	}

	if !lastEvent.Done {
		t.Fatalf("pipeline did not complete")
	}
	if lastEvent.Manifest == nil {
		t.Fatalf("expected Manifest on completion event")
	}

	apps, err := registry.LoadAll(pathsInstance)
	if err != nil {
		t.Fatalf("failed loading apps: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 installed app, got %d", len(apps))
	}

	app := apps[0]
	if !registry.FileExists(app.AppImagePath) {
		t.Errorf("app binary does not exist: %s", app.AppImagePath)
	}
	if !registry.FileExists(app.WrapperFile) {
		t.Errorf("wrapper script does not exist: %s", app.WrapperFile)
	}
	if !registry.FileExists(app.DesktopFile) {
		t.Errorf("desktop file does not exist: %s", app.DesktopFile)
	}
}

func TestRealAppImageExtractionSmoke(t *testing.T) {
	realAppImage := "/home/xibhi/Downloads/nuza_2.2.0_amd64.AppImage"
	if _, err := os.Stat(realAppImage); os.IsNotExist(err) {
		t.Skip("local sample AppImage not found, skipping real extraction smoke test")
	}

	tmp := t.TempDir()
	pathsInstance := &paths.Paths{
		AppsDir:         filepath.Join(tmp, "apps"),
		BinDir:          filepath.Join(tmp, "bin"),
		ApplicationsDir: filepath.Join(tmp, "applications"),
		IconsDir:        filepath.Join(tmp, "icons"),
		PixmapsDir:      filepath.Join(tmp, "pixmaps"),
		StateHome:       tmp,
		Home:            tmp,
	}

	runner := NewRunner(pathsInstance)
	ctx := context.Background()
	out := make(chan Event, 32)

	go runner.Run(ctx, realAppImage, false, false, "", out)

	var completedManifest *registry.App
	for ev := range out {
		if ev.Err != nil {
			t.Fatalf("step %s failed: %v", ev.Step, ev.Err)
		}
		if ev.Done {
			completedManifest = ev.Manifest
		}
	}

	if completedManifest == nil {
		t.Fatalf("pipeline finished without manifest")
	}
	if completedManifest.ID != "nuza" {
		t.Errorf("expected ID 'nuza', got %q", completedManifest.ID)
	}
	if completedManifest.Name != "nuza" {
		t.Errorf("expected Name 'nuza', got %q", completedManifest.Name)
	}
	if completedManifest.ExecName != "nuza" {
		t.Errorf("expected ExecName 'nuza', got %q", completedManifest.ExecName)
	}
	if len(completedManifest.IconFiles) == 0 {
		t.Errorf("expected extracted icon files, got none")
	}
	if !registry.FileExists(completedManifest.DesktopFile) {
		t.Errorf("desktop file missing: %s", completedManifest.DesktopFile)
	}
	if !registry.FileExists(completedManifest.WrapperFile) {
		t.Errorf("wrapper script missing: %s", completedManifest.WrapperFile)
	}

	deskData, err := os.ReadFile(completedManifest.DesktopFile)
	if err != nil {
		t.Fatalf("failed reading desktop file: %v", err)
	}
	if !strings.Contains(string(deskData), "X-Plumcake-Managed=true") {
		t.Errorf("missing X-Plumcake-Managed in desktop: %s", string(deskData))
	}
	if !strings.Contains(string(deskData), "Icon=plumcake-nuza") {
		t.Errorf("missing Icon=plumcake-nuza in desktop: %s", string(deskData))
	}
}
