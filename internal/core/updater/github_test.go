package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseGitHubRepoInput(t *testing.T) {
	testCases := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantIs    bool
	}{
		{"github.com/mockuser/mockrepo", "mockuser", "mockrepo", true},
		{"https://github.com/mockuser/mockrepo/", "mockuser", "mockrepo", true},
		{"mockuser/mockrepo", "mockuser", "mockrepo", true},
		{"http://example.com/app.AppImage", "", "", false},
		{"/home/user/app.AppImage", "", "", false},
		{"https://github.com/mockuser/mockrepo/releases/download/v1.0/app.AppImage", "", "", false},
	}

	for _, tc := range testCases {
		owner, repo, isRepo := ParseGitHubRepoInput(tc.input)
		if isRepo != tc.wantIs || owner != tc.wantOwner || repo != tc.wantRepo {
			t.Errorf("input %s: got (%s, %s, %t), want (%s, %s, %t)",
				tc.input, owner, repo, isRepo, tc.wantOwner, tc.wantRepo, tc.wantIs)
		}
	}
}

func TestGitHubAssetPickerWithMock(t *testing.T) {
	releases := []GitHubRelease{
		{
			TagName:    "v2.0.0-beta",
			Prerelease: true,
			Assets: []GitHubAsset{
				{
					Name:               "myapp-2.0.0-beta-x86_64.AppImage",
					Size:               50000000,
					BrowserDownloadURL: "https://mock.com/download/beta.AppImage",
				},
			},
		},
		{
			TagName:    "v1.0.0",
			Prerelease: false,
			Assets: []GitHubAsset{
				{
					Name:               "myapp-1.0.0-x86_64.AppImage",
					Size:               40000000,
					BrowserDownloadURL: "https://mock.com/download/1.0.0.AppImage",
				},
				{
					Name:               "myapp-1.0.0-aarch64.AppImage",
					Size:               40000000,
					BrowserDownloadURL: "https://mock.com/download/1.0.0-arm.AppImage",
				},
				{
					Name:               "myapp-1.0.0.AppImage.sha256",
					Size:               64,
					BrowserDownloadURL: "https://mock.com/download/1.0.0.sha256",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releases)
	}))
	defer server.Close()

	var matchedWithoutPre []MatchedAsset
	for _, rel := range releases {
		if rel.Prerelease {
			continue
		}
		for _, asset := range rel.Assets {
			if asset.Name == "myapp-1.0.0-x86_64.AppImage" {
				matchedWithoutPre = append(matchedWithoutPre, MatchedAsset{
					TagName:     rel.TagName,
					AssetName:   asset.Name,
					DownloadURL: asset.BrowserDownloadURL,
					Arch:        "x86_64",
				})
			}
		}
	}

	if len(matchedWithoutPre) != 1 || matchedWithoutPre[0].TagName != "v1.0.0" {
		t.Fatalf("expected v1.0.0 stable asset, got: %+v", matchedWithoutPre)
	}
}
