package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
)

func TestInspectPreInstall(t *testing.T) {
	tempDirectory := t.TempDir()
	pathsInstance := &paths.Paths{
		AppsDir: filepath.Join(tempDirectory, "apps"),
	}

	appImagePath := filepath.Join(tempDirectory, "valid.AppImage")
	header := make([]byte, 64)
	copy(header[0:4], []byte("\x7fELF"))
	header[8] = 0x41
	header[9] = 0x49
	header[10] = 0x02
	header[18] = 0x3E
	_ = os.WriteFile(appImagePath, header, 0755)

	ctx := context.Background()
	inspection, err := InspectPreInstall(ctx, pathsInstance, appImagePath, "", "")
	if err != nil {
		t.Fatalf("InspectPreInstall failed: %v", err)
	}

	if inspection.Arch != "x86_64" {
		t.Fatalf("expected x86_64 arch, got %s", inspection.Arch)
	}
	if !inspection.ArchMatches {
		t.Fatalf("expected arch to match machine")
	}
	if inspection.Size != 64 {
		t.Fatalf("expected size 64, got %d", inspection.Size)
	}
}
