package appimage

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"
)

func Validate(filePath string) (bool, string, string, error) {
	if !strings.HasSuffix(strings.ToLower(filePath), ".appimage") {
		return false, "", "", nil
	}

	fileInformation, err := os.Stat(filePath)
	if err != nil || fileInformation.IsDir() {
		return false, "", "", nil
	}

	fileDescriptor, err := os.Open(filePath)
	if err != nil {
		return false, "", "", err
	}
	defer fileDescriptor.Close()

	header := make([]byte, 64)
	bytesRead, err := io.ReadFull(fileDescriptor, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return false, "", "", err
	}
	if bytesRead < 20 {
		return false, "", "", nil
	}

	if !bytes.HasPrefix(header, []byte("\x7fELF")) {
		return false, "", "", nil
	}

	machineCode := binary.LittleEndian.Uint16(header[18:20])
	architecture := "other"
	switch machineCode {
	case 0x3E:
		architecture = "x86_64"
	case 0xB7:
		architecture = "aarch64"
	case 0x03:
		architecture = "i386"
	case 0x28:
		architecture = "arm"
	}

	appImageType := "Type 2"
	if len(header) >= 11 && header[8] == 0x41 && header[9] == 0x49 && header[10] == 0x02 {
		appImageType = "Type 2"
	} else if len(header) >= 11 && header[8] == 0x41 && header[9] == 0x49 && header[10] == 0x01 {
		appImageType = "Type 1"
	}

	return true, architecture, appImageType, nil
}

func IsValid(filePath string) bool {
	valid, _, _, err := Validate(filePath)
	return err == nil && valid
}
