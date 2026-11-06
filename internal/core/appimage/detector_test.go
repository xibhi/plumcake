package appimage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	tempDirectory := t.TempDir()

	nonElfFile := filepath.Join(tempDirectory, "fake.AppImage")
	_ = os.WriteFile(nonElfFile, []byte("plain text file"), 0755)
	valid, _, _, _ := Validate(nonElfFile)
	if valid {
		t.Fatalf("expected non-ELF file to fail validation")
	}

	type2File := filepath.Join(tempDirectory, "valid2.AppImage")
	type2Header := make([]byte, 64)
	copy(type2Header[0:4], []byte("\x7fELF"))
	type2Header[8] = 0x41
	type2Header[9] = 0x49
	type2Header[10] = 0x02
	type2Header[18] = 0x3E
	_ = os.WriteFile(type2File, type2Header, 0755)

	valid2, architecture, appImageType, err := Validate(type2File)
	if err != nil || !valid2 {
		t.Fatalf("expected Type 2 AppImage valid, err=%v", err)
	}
	if architecture != "x86_64" {
		t.Fatalf("expected architecture x86_64, got %s", architecture)
	}
	if appImageType != "Type 2" {
		t.Fatalf("expected Type 2, got %s", appImageType)
	}

	nonExtensionFile := filepath.Join(tempDirectory, "valid.bin")
	_ = os.WriteFile(nonExtensionFile, type2Header, 0755)
	validExtension, _, _, _ := Validate(nonExtensionFile)
	if validExtension {
		t.Fatalf("expected non-.AppImage extension to fail validation")
	}
}
