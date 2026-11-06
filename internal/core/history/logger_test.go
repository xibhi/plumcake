package history

import (
	"os"
	"strings"
	"testing"

	"github.com/xibhi/plumcake/internal/core/paths"
)

func TestLogRotationAndRedaction(t *testing.T) {
	tempDirectory := t.TempDir()
	originalState := os.Getenv("XDG_STATE_HOME")
	os.Setenv("XDG_STATE_HOME", tempDirectory)
	defer os.Setenv("XDG_STATE_HOME", originalState)

	pathsInstance := &paths.Paths{Home: tempDirectory}

	Record(pathsInstance, "install", "secretapp", "success", "Fetched with token=supersecret123 parameter")
	entries, err := ReadRecent(pathsInstance, 10)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected logged entries, got err=%v", err)
	}

	if strings.Contains(entries[0].Message, "supersecret123") {
		t.Fatalf("sensitive token was not redacted from log: %s", entries[0].Message)
	}

	logFile := FilePath(pathsInstance)
	largeChunk := strings.Repeat("A", 1024*1024+10)
	_ = os.WriteFile(logFile, []byte(largeChunk), 0644)

	Record(pathsInstance, "rotate_check", "testapp", "success", "new entry after rotation")

	if _, err := os.Stat(logFile + ".1"); os.IsNotExist(err) {
		t.Fatalf("expected rotated logfile .1 to exist")
	}

	fileInfo, err := os.Stat(logFile)
	if err != nil || fileInfo.Size() > 1024*100 {
		t.Fatalf("expected new log file to be fresh and small, size=%d", fileInfo.Size())
	}
}
