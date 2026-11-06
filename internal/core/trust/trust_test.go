package trust

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseChecksumFile(t *testing.T) {
	content := `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  myapp-1.0.0.AppImage
a1b2c3d4e5f67890123456789012345678901234567890123456789012345678 *otherapp.AppImage
`
	hash1 := ParseChecksumFile(content, "myapp-1.0.0.AppImage")
	if hash1 != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("expected hash1 match, got: %s", hash1)
	}

	hash2 := ParseChecksumFile(content, "otherapp.AppImage")
	if hash2 != "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678" {
		t.Fatalf("expected hash2 match, got: %s", hash2)
	}

	bareContent := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n"
	hashBare := ParseChecksumFile(bareContent, "anything.AppImage")
	if hashBare != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("expected bare hash match, got: %s", hashBare)
	}
}

func TestTrustStateDecisions(t *testing.T) {
	tempDirectory := t.TempDir()
	dummyFile := filepath.Join(tempDirectory, "dummy.AppImage")
	content := []byte("dummy appimage content for testing")
	if err := os.WriteFile(dummyFile, content, 0755); err != nil {
		t.Fatal(err)
	}

	actualHash, err := ComputeSHA256(dummyFile)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	resVerified := VerifyBinary(ctx, dummyFile, "", actualHash)
	if resVerified.State != Verified {
		t.Fatalf("expected Verified, got %s", resVerified.State)
	}
	if resVerified.IsHardFail {
		t.Fatalf("expected IsHardFail false")
	}

	badHash := "0000000000000000000000000000000000000000000000000000000000000000"
	resMismatch := VerifyBinary(ctx, dummyFile, "", badHash)
	if resMismatch.State != Mismatch {
		t.Fatalf("expected Mismatch, got %s", resMismatch.State)
	}
	if !resMismatch.IsHardFail {
		t.Fatalf("expected IsHardFail true on mismatch")
	}

	resUnsigned := VerifyBinary(ctx, dummyFile, "", "")
	if resUnsigned.State != Unsigned {
		t.Fatalf("expected Unsigned, got %s", resUnsigned.State)
	}
}
