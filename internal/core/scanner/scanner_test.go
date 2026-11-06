package scanner

import (
	"testing"
)

func TestScannerExclusions(t *testing.T) {
	home := "/home/user"
	appsDir := "/home/user/.local/share/plumcake/apps"

	skips := []string{
		"/proc",
		"/proc/1/cwd",
		"/sys/devices",
		"/dev/null",
		"/run/user/1000",
		"/var/log",
		"/home/user/.git",
		"/home/user/project/node_modules",
		"/home/user/.local/share/plumcake/apps/nuza",
		"/home/user/.local/share/flatpak/app",
	}

	for _, s := range skips {
		if !ShouldSkip(s, home, appsDir) {
			t.Errorf("expected ShouldSkip(%s) == true", s)
		}
	}

	keeps := []string{
		"/home/user/Downloads",
		"/home/user/Desktop",
		"/home/user/Applications",
		"/opt",
	}

	for _, k := range keeps {
		if ShouldSkip(k, home, appsDir) {
			t.Errorf("expected ShouldSkip(%s) == false", k)
		}
	}
}

func TestFuzzyMatch(t *testing.T) {
	if !FuzzyMatch("nuza", "nuza_2.1.0_amd64.AppImage") {
		t.Fatalf("expected fuzzy match nuza in nuza_2.1.0_amd64.AppImage")
	}
	if !FuzzyMatch("mst", "Maestro-1.0.2.AppImage") {
		t.Fatalf("expected fuzzy match mst in Maestro")
	}
	if FuzzyMatch("xyz", "Maestro-1.0.2.AppImage") {
		t.Fatalf("did not expect fuzzy match xyz in Maestro")
	}
}
