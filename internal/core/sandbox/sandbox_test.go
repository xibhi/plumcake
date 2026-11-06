package sandbox

import (
	"strings"
	"testing"
)

func TestGenerateExec(t *testing.T) {
	binaryPath := "/home/user/.local/share/plumcake/apps/test/test.AppImage"
	originalExec := "/home/user/.local/bin/test %U"

	offExec := GenerateExec(originalExec, binaryPath, "off", true)
	if offExec != originalExec {
		t.Fatalf("expected original exec when off, got %s", offExec)
	}

	firejailNet := GenerateExec(originalExec, binaryPath, "firejail", true)
	if !strings.HasPrefix(firejailNet, "firejail") || strings.Contains(firejailNet, "--net=none") {
		t.Fatalf("unexpected firejail exec with network: %s", firejailNet)
	}

	firejailNoNet := GenerateExec(originalExec, binaryPath, "firejail", false)
	if !strings.Contains(firejailNoNet, "--net=none") {
		t.Fatalf("expected --net=none in firejail without network: %s", firejailNoNet)
	}

	bwrapNoNet := GenerateExec(originalExec, binaryPath, "bwrap", false)
	if !strings.HasPrefix(bwrapNoNet, "bwrap") || !strings.Contains(bwrapNoNet, "--unshare-net") {
		t.Fatalf("expected bwrap with --unshare-net: %s", bwrapNoNet)
	}
}
