package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repos/dominhduc/agent-brain/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if accept := r.Header.Get("Accept"); accept != "application/vnd.github+json" {
			t.Errorf("expected Accept header, got %s", accept)
		}

		resp := GitHubRelease{
			TagName: "v0.3.0",
			Assets: []GitHubAsset{
				{
					Name:               "brain_Linux_x86_64.tar.gz",
					BrowserDownloadURL: "https://github.com/dominhduc/agent-brain/releases/download/v0.3.0/brain_Linux_x86_64.tar.gz",
				},
				{
					Name:               "brain_Darwin_arm64.tar.gz",
					BrowserDownloadURL: "https://github.com/dominhduc/agent-brain/releases/download/v0.3.0/brain_Darwin_arm64.tar.gz",
				},
				{
					Name:               "brain_Windows_x86_64.zip",
					BrowserDownloadURL: "https://github.com/dominhduc/agent-brain/releases/download/v0.3.0/brain_Windows_x86_64.zip",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	release, err := FetchLatestRelease(FetchOptions{
		APIBaseURL: server.URL,
		Owner:      "dominhduc",
		Repo:       "agent-brain",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release.TagName != "v0.3.0" {
		t.Errorf("expected tag v0.3.0, got %s", release.TagName)
	}
	if len(release.Assets) != 3 {
		t.Errorf("expected 3 assets, got %d", len(release.Assets))
	}
}

func TestFetchLatestRelease_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer server.Close()

	_, err := FetchLatestRelease(FetchOptions{
		APIBaseURL: server.URL,
		Owner:      "dominhduc",
		Repo:       "agent-brain",
	})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestFindAssetForPlatform(t *testing.T) {
	release := GitHubRelease{
		TagName: "v0.3.0",
		Assets: []GitHubAsset{
			{Name: "brain_Linux_x86_64.tar.gz", BrowserDownloadURL: "url1"},
			{Name: "brain_Darwin_arm64.tar.gz", BrowserDownloadURL: "url2"},
			{Name: "brain_Windows_x86_64.zip", BrowserDownloadURL: "url3"},
			{Name: "brain_Linux_arm64.tar.gz", BrowserDownloadURL: "url4"},
		},
	}

	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"linux", "amd64", "url1"},
		{"darwin", "arm64", "url2"},
		{"windows", "amd64", "url3"},
		{"linux", "arm64", "url4"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			asset, err := FindAssetForPlatform(release, tt.goos, tt.goarch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if asset.BrowserDownloadURL != tt.want {
				t.Errorf("FindAssetForPlatform(%s, %s) = %q, want %q", tt.goos, tt.goarch, asset.BrowserDownloadURL, tt.want)
			}
		})
	}
}

func TestFindAssetForPlatform_NotFound(t *testing.T) {
	release := GitHubRelease{
		TagName: "v0.3.0",
		Assets:  []GitHubAsset{},
	}

	_, err := FindAssetForPlatform(release, "linux", "amd64")
	if err == nil {
		t.Fatal("expected error when no matching asset")
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v0.2", "v0.3.0", true},
		{"v0.3.0", "v0.3.0", false},
		{"v0.3.0", "v0.2", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.1.0", "v1.0.9", false},
		{"v0.2", "v0.2.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.latest, func(t *testing.T) {
			got := IsNewerVersion(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestDownloadAndReplace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("fake-binary-content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "brain")
	os.WriteFile(binPath, []byte("old-binary"), 0755)

	err := DownloadAndReplace(server.URL, binPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-binary-content" {
		t.Errorf("binary not replaced, got: %q", string(data))
	}

	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("binary should be executable")
	}
}

func TestDownloadAndReplace_Backup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("new-binary"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "brain")
	os.WriteFile(binPath, []byte("old-binary"), 0755)

	err := DownloadAndReplace(server.URL, binPath)
	if err != nil {
		t.Fatal(err)
	}

	backupPath := binPath + ".bak"
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup file should exist: %v", err)
	}
	if string(backup) != "old-binary" {
		t.Errorf("backup should contain old binary, got: %q", string(backup))
	}
}
