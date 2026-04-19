package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type GitHubAsset struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName   string        `json:"tag_name"`
	Assets    []GitHubAsset `json:"assets"`
	Body      string        `json:"body"`
	Checksums map[string]string
}

var updateClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          5,
		IdleConnTimeout:       60 * time.Second,
		DisableCompression:    false,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

type FetchOptions struct {
	APIBaseURL string
	Owner      string
	Repo       string
}

func FetchLatestRelease(opts FetchOptions) (GitHubRelease, error) {
	var release GitHubRelease

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", opts.APIBaseURL, opts.Owner, opts.Repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return release, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := updateClient.Do(req)
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

func FindAssetForPlatform(release GitHubRelease, goos, goarch string) (GitHubAsset, error) {
	osNameMap := map[string]string{
		"darwin":  "Darwin",
		"linux":   "Linux",
		"windows": "Windows",
	}
	osName := osNameMap[goos]
	if osName == "" {
		if len(goos) > 0 {
			osName = strings.ToUpper(goos[:1]) + goos[1:]
		}
	}

	archName := goarch
	if goarch == "amd64" {
		archName = "x86_64"
	}

	expectedSuffix := fmt.Sprintf("_%s_%s.", osName, archName)

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, expectedSuffix) {
			return asset, nil
		}
	}

	return GitHubAsset{}, fmt.Errorf("no asset found for %s/%s in release %s. Available: %v",
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
		n, err := strconv.Atoi(p)
		if err != nil {
			n = 0
		}
		result[i] = n
	}
	return result
}

func downloadFile(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := updateClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return nil, fmt.Errorf("reading download: %w", err)
	}
	return body, nil
}

func DownloadAsset(apiBaseURL string, assetID int) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/dominhduc/agent-brain/releases/assets/%d", apiBaseURL, assetID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating asset request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := updateClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("asset download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return nil, fmt.Errorf("reading asset: %w", err)
	}
	return body, nil
}

func DownloadFile(url string) ([]byte, error) {
	return downloadFile(url)
}

func ReplaceBinary(archiveData []byte, filename, binPath string, releaseChecksums map[string]string) error {
	var binaryData []byte
	var err error

	switch {
	case strings.HasSuffix(filename, ".zip"):
		binaryData, err = extractFromZip(archiveData)
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}
	case strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz"):
		binaryData, err = extractFromTarGz(archiveData)
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}
	default:
		binaryData = archiveData
	}

	if len(binaryData) == 0 {
		return fmt.Errorf("extracted binary is empty")
	}

	if releaseChecksums != nil {
		if expected, ok := releaseChecksums[filename]; ok {
			if !VerifyChecksum(binaryData, expected) {
				return fmt.Errorf("CHECKSUM MISMATCH: binary integrity verification failed — refusing to install. Expected %s for %s", expected, filename)
			}
		}
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

func DownloadAndReplace(downloadURL, binPath string) error {
	body, err := downloadFile(downloadURL)
	if err != nil {
		return err
	}

	filename := downloadURL
	if idx := strings.LastIndex(downloadURL, "/"); idx >= 0 {
		filename = downloadURL[idx+1:]
	}
	return ReplaceBinary(body, filename, binPath, nil)
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
		if hdr.Typeflag == tar.TypeReg && !strings.Contains(hdr.Name, "..") && isBrainBinary(filepath.Base(hdr.Name)) {
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
		if isBrainBinary(base) {
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

func isBrainBinary(name string) bool {
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	return name == "brain" || name == "brain.exe" || strings.HasPrefix(name, "brain_")
}

func CheckLatest(currentVersion string) (string, error) {
	release, err := FetchLatestRelease(FetchOptions{
		APIBaseURL: "https://api.github.com",
		Owner:      "dominhduc",
		Repo:       "agent-brain",
	})
	if err != nil {
		return "", err
	}
	if IsNewerVersion(currentVersion, release.TagName) {
		return release.TagName, nil
	}
	return "", nil
}

func ParseChecksums(body string) map[string]string {
	checksums := make(map[string]string)
	re := regexp.MustCompile(`(?m)(?:SHA256\(([^)]+)\)|sha256\s+)?([a-fA-F0-9]{64})\s+(?:\*\s*)?(\S+)`)
	matches := re.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		filename := m[1]
		if filename == "" {
			filename = m[3]
		}
		checksums[filepath.Base(filename)] = m[2]
	}
	return checksums
}

func VerifyChecksum(data []byte, expectedHex string) bool {
	h := sha256.Sum256(data)
	actual := hex.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHex)) == 1
}
