package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt string        `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

type MatchedAsset struct {
	Repo        string
	TagName     string
	AssetName   string
	DownloadURL string
	Size        int64
	Arch        string
	DigestURL   string
}

func ParseGitHubRepoInput(input string) (string, string, bool) {
	clean := strings.TrimSpace(input)
	if strings.Contains(clean, "://") && !strings.Contains(clean, "github.com/") {
		return "", "", false
	}
	clean = strings.TrimPrefix(clean, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "github.com/")
	clean = strings.TrimSuffix(clean, "/")

	if strings.HasSuffix(strings.ToLower(clean), ".appimage") {
		return "", "", false
	}

	if strings.Contains(clean, "/releases/download/") {
		return "", "", false
	}

	parts := strings.Split(clean, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+/[a-zA-Z0-9_\-\.]+$`, clean)
		if matched {
			return parts[0], parts[1], true
		}
	}
	return "", "", false
}

func FetchGitHubReleaseAssets(ctx context.Context, owner, repo string, includePrerelease bool) ([]MatchedAsset, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	request, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "plumcake/1.0.1 (Linux; x86_64)")
	request.Header.Set("Accept", "application/vnd.github.v3+json")

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("GitHub API connection error: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("GitHub API rate limit exceeded. Set GITHUB_TOKEN environment variable")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub repository %s/%s returned HTTP %d", owner, repo, response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	var releases []GitHubRelease
	if err := json.Unmarshal(bodyBytes, &releases); err != nil {
		return nil, fmt.Errorf("error parsing GitHub releases JSON: %v", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found for repository %s/%s", owner, repo)
	}

	machineArch := runtime.GOARCH
	var matchedAssets []MatchedAsset

	for _, rel := range releases {
		if rel.Prerelease && !includePrerelease {
			continue
		}

		var candidateAssets []GitHubAsset
		var sha256AssetURL string

		for _, asset := range rel.Assets {
			assetLower := strings.ToLower(asset.Name)
			if strings.HasSuffix(assetLower, ".appimage") {
				candidateAssets = append(candidateAssets, asset)
			} else if strings.HasSuffix(assetLower, ".sha256") || strings.HasSuffix(assetLower, ".sha256sum") || assetLower == "sha256sums" {
				sha256AssetURL = asset.BrowserDownloadURL
			}
		}

		if len(candidateAssets) == 0 {
			continue
		}

		for _, asset := range candidateAssets {
			nameLower := strings.ToLower(asset.Name)
			assetArch := "other"
			archMatches := false

			if strings.Contains(nameLower, "x86_64") || strings.Contains(nameLower, "amd64") {
				assetArch = "x86_64"
				if machineArch == "amd64" {
					archMatches = true
				}
			} else if strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "aarch64") {
				assetArch = "aarch64"
				if machineArch == "arm64" {
					archMatches = true
				}
			} else if strings.Contains(nameLower, "i386") || strings.Contains(nameLower, "i686") {
				assetArch = "i386"
			} else if strings.Contains(nameLower, "arm") {
				assetArch = "arm"
			} else {
				archMatches = true
				assetArch = "universal"
			}

			if archMatches {
				matchedAssets = append(matchedAssets, MatchedAsset{
					Repo:        fmt.Sprintf("%s/%s", owner, repo),
					TagName:     rel.TagName,
					AssetName:   asset.Name,
					DownloadURL: asset.BrowserDownloadURL,
					Size:        asset.Size,
					Arch:        assetArch,
					DigestURL:   sha256AssetURL,
				})
			}
		}

		if len(matchedAssets) > 0 {
			break
		}
	}

	if len(matchedAssets) == 0 {
		return nil, fmt.Errorf("no AppImage assets matching %s architecture found in %s/%s", machineArch, owner, repo)
	}

	return matchedAssets, nil
}
