package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

type FetchOptions struct {
	APIBaseURL string
	Owner      string
	Repo       string
}

func FetchLatestRelease(opts FetchOptions) (GitHubRelease, error) {
	var release GitHubRelease

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", opts.APIBaseURL, opts.Owner, opts.Repo)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return release, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return release, fmt.Errorf("fetching release info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return release, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return release, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return release, fmt.Errorf("parsing release JSON: %w", err)
	}

	return release, nil
}

func FindAssetForPlatform(release GitHubRelease, goos, goarch string) (string, error) {
	osName := strings.Title(goos)
	if goos == "darwin" {
		osName = "Darwin"
	}
	if goos == "linux" {
		osName = "Linux"
	}
	if goos == "windows" {
		osName = "Windows"
	}

	archName := goarch
	if goarch == "amd64" {
		archName = "x86_64"
	}

	expectedSuffix := fmt.Sprintf("_%s_%s.", osName, archName)

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, expectedSuffix) {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no asset found for %s/%s in release %s. Available: %v",
		goos, goarch, release.TagName, assetNames(release.Assets))
}

func assetNames(assets []GitHubAsset) []string {
	names := make([]string, len(assets))
	for i, a := range assets {
		names[i] = a.Name
	}
	return names
}

func IsNewerVersion(current, latest string) bool {
	c := strings.TrimPrefix(current, "v")
	l := strings.TrimPrefix(latest, "v")

	cParts := versionParts(c)
	lParts := versionParts(l)

	maxLen := len(cParts)
	if len(lParts) > maxLen {
		maxLen = len(lParts)
	}

	for i := 0; i < maxLen; i++ {
		cv := 0
		lv := 0
		if i < len(cParts) {
			cv = cParts[i]
		}
		if i < len(lParts) {
			lv = lParts[i]
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}
	return false
}

func versionParts(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%d", &result[i])
	}
	return result
}

func DownloadAndReplace(downloadURL, binPath string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return fmt.Errorf("reading download: %w", err)
	}

	var binaryData []byte

	switch {
	case strings.HasSuffix(downloadURL, ".zip"):
		binaryData, err = extractFromZip(body)
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}
	case strings.HasSuffix(downloadURL, ".tar.gz") || strings.HasSuffix(downloadURL, ".tgz"):
		binaryData, err = extractFromTarGz(body)
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}
	default:
		binaryData = body
	}

	if len(binaryData) == 0 {
		return fmt.Errorf("extracted binary is empty")
	}

	backupPath := binPath + ".bak"
	if err := os.Rename(binPath, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	tmpPath := binPath + ".tmp"
	if err := os.WriteFile(tmpPath, binaryData, 0755); err != nil {
		os.Rename(backupPath, binPath)
		return fmt.Errorf("writing new binary: %w", err)
	}

	if err := os.Rename(tmpPath, binPath); err != nil {
		os.Rename(backupPath, binPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && (filepath.Base(hdr.Name) == "brain" || filepath.Base(hdr.Name) == "brain.exe") {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf("brain binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("creating zip reader: %w", err)
	}

	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == "brain" || base == "brain.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening file in zip: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("brain binary not found in zip archive")
}
