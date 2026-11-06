package trust

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type State string

const (
	Verified         State = "verified"
	SignedUnverified State = "signed-unverified"
	Unsigned         State = "unsigned"
	Mismatch         State = "mismatch"
)

func (s State) BadgeGlyph() string {
	switch s {
	case Verified:
		return "✓"
	case SignedUnverified:
		return "◐"
	case Unsigned:
		return "○"
	case Mismatch:
		return "✗"
	default:
		return "○"
	}
}

type Result struct {
	State        State
	ExpectedHash string
	ActualHash   string
	SignatureKey string
	Message      string
	IsHardFail   bool
}

func ComputeSHA256(filePath string) (string, error) {
	fileDescriptor, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fileDescriptor.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, fileDescriptor); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func ParseChecksumFile(content string, targetFilename string) string {
	targetBase := strings.ToLower(filepath.Base(targetFilename))
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 1 && len(fields[0]) == 64 {
			return strings.ToLower(fields[0])
		}

		if len(fields) >= 2 && len(fields[0]) == 64 {
			hashValue := strings.ToLower(fields[0])
			filename := strings.TrimPrefix(fields[1], "*")
			filenameBase := strings.ToLower(filepath.Base(filename))

			if targetBase == "" || filenameBase == targetBase {
				return hashValue
			}
		}
	}
	return ""
}

func FetchSiblingChecksum(ctx context.Context, sourceURL string) string {
	if !strings.HasPrefix(sourceURL, "http://") && !strings.HasPrefix(sourceURL, "https://") {
		candidates := []string{
			sourceURL + ".sha256",
			sourceURL + ".sha256sum",
			filepath.Join(filepath.Dir(sourceURL), "SHA256SUMS"),
			filepath.Join(filepath.Dir(sourceURL), "sha256sums.txt"),
		}
		for _, candidate := range candidates {
			if content, err := os.ReadFile(candidate); err == nil {
				checksumValue := ParseChecksumFile(string(content), filepath.Base(sourceURL))
				if checksumValue != "" {
					return checksumValue
				}
			}
		}
		return ""
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	candidates := []string{
		sourceURL + ".sha256",
		sourceURL + ".sha256sum",
	}

	lastSlash := strings.LastIndex(sourceURL, "/")
	if lastSlash > 0 {
		candidates = append(candidates, sourceURL[:lastSlash+1]+"SHA256SUMS")
	}

	for _, urlAddress := range candidates {
		request, err := http.NewRequestWithContext(ctx, "GET", urlAddress, nil)
		if err != nil {
			continue
		}
		request.Header.Set("User-Agent", "plumcake/1.0.1 (Linux; x86_64)")

		response, err := httpClient.Do(request)
		if err != nil || response.StatusCode != http.StatusOK {
			if response != nil {
				response.Body.Close()
			}
			continue
		}

		bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, 512*1024))
		response.Body.Close()
		if err != nil {
			continue
		}

		checksumValue := ParseChecksumFile(string(bodyBytes), filepath.Base(sourceURL))
		if checksumValue != "" {
			return checksumValue
		}
	}

	return ""
}

func CheckSignature(ctx context.Context, appImagePath string) (bool, bool, string, error) {
	command := exec.CommandContext(ctx, appImagePath, "--appimage-signature")
	output, err := command.CombinedOutput()
	if err == nil && len(output) > 0 && bytes.Contains(output, []byte("-----BEGIN PGP SIGNATURE-----")) {
		hasSignature := true
		if _, gpgError := exec.LookPath("gpg"); gpgError == nil {
			temporarySignature, err := os.CreateTemp("", "plumcake_sig_*")
			if err == nil {
				defer os.Remove(temporarySignature.Name())
				_, _ = temporarySignature.Write(output)
				temporarySignature.Close()

				gpgCommand := exec.CommandContext(ctx, "gpg", "--status-fd", "1", "--verify", temporarySignature.Name(), appImagePath)
				gpgOutput, _ := gpgCommand.CombinedOutput()
				gpgString := string(gpgOutput)

				if strings.Contains(gpgString, "VALIDSIG") {
					for _, line := range strings.Split(gpgString, "\n") {
						if strings.HasPrefix(line, "[GNUPG:] VALIDSIG") {
							fields := strings.Fields(line)
							if len(fields) >= 3 {
								return true, true, fields[2], nil
							}
						}
					}
					return true, true, "", nil
				}
				if strings.Contains(gpgString, "NO_PUBKEY") {
					for _, line := range strings.Split(gpgString, "\n") {
						if strings.HasPrefix(line, "[GNUPG:] NO_PUBKEY") {
							fields := strings.Fields(line)
							if len(fields) >= 3 {
								return true, false, fields[2], nil
							}
						}
					}
					return true, false, "", nil
				}
			}
		}
		return hasSignature, false, "", nil
	}

	return false, false, "", nil
}

func VerifyBinary(ctx context.Context, filePath string, sourceURL string, expectedDigest string) Result {
	actualHash, err := ComputeSHA256(filePath)
	if err != nil {
		return Result{
			State:      Unsigned,
			Message:    fmt.Sprintf("Failed reading file: %v", err),
			IsHardFail: false,
		}
	}

	if expectedDigest == "" && sourceURL != "" {
		expectedDigest = FetchSiblingChecksum(ctx, sourceURL)
	}

	if expectedDigest != "" {
		expectedClean := strings.ToLower(strings.TrimSpace(expectedDigest))
		if actualHash != expectedClean {
			return Result{
				State:        Mismatch,
				ExpectedHash: expectedClean,
				ActualHash:   actualHash,
				Message: fmt.Sprintf(
					"CRITICAL: SHA256 checksum mismatch!\nExpected: %s\nActual:   %s",
					expectedClean, actualHash,
				),
				IsHardFail: true,
			}
		}
		return Result{
			State:        Verified,
			ExpectedHash: expectedClean,
			ActualHash:   actualHash,
			Message:      "SHA256 checksum verified against published release digest.",
			IsHardFail:   false,
		}
	}

	hasSignature, verified, keyIdentifier, _ := CheckSignature(ctx, filePath)
	if hasSignature {
		if verified {
			return Result{
				State:        Verified,
				ActualHash:   actualHash,
				SignatureKey: keyIdentifier,
				Message:      fmt.Sprintf("Embedded GPG signature verified (key %s).", keyIdentifier),
				IsHardFail:   false,
			}
		}
		return Result{
			State:        SignedUnverified,
			ActualHash:   actualHash,
			SignatureKey: keyIdentifier,
			Message:      fmt.Sprintf("Embedded GPG signature present from key %s (public key not imported in local keyring).", keyIdentifier),
			IsHardFail:   false,
		}
	}

	return Result{
		State:      Unsigned,
		ActualHash: actualHash,
		Message:    "No published checksum or digital signature found. Application is unsigned.",
		IsHardFail: false,
	}
}
