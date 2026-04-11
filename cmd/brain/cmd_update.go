package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/updater"
)

func cmdUpdate() {
	fmt.Printf("Current version: %s\n", version)

	fmt.Println("Checking for updates...")
	release, err := updater.FetchLatestRelease(updater.FetchOptions{
		APIBaseURL: "https://api.github.com",
		Owner:      "dominhduc",
		Repo:       "agent-brain",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\nWhat to do: check your internet connection or try again later.\n", err)
		os.Exit(1)
	}

	if !updater.IsNewerVersion(version, release.TagName) {
		fmt.Printf("Already up to date (%s).\n", version)
		return
	}

	fmt.Printf("New version available: %s → %s\n", version, release.TagName)

	asset, err := updater.FindAssetForPlatform(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine binary path: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	resolvedPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		resolvedPath = execPath
	}

	fmt.Printf("Downloading %s...\n", asset.Name)

	var archiveData []byte
	if asset.ID > 0 && os.Getenv("GITHUB_TOKEN") != "" {
		archiveData, err = updater.DownloadAsset("https://api.github.com", asset.ID)
	} else {
		archiveData, err = updater.DownloadFile(asset.BrowserDownloadURL)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	if err := updater.ReplaceBinary(archiveData, asset.Name, resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to %s successfully!\n", release.TagName)

	fmt.Println("\nSkill updates available. Run 'brain skill diff' to preview changes.")

	service.StopCurrentProject()
	fmt.Println("Restart the daemon with: brain daemon start")
}
