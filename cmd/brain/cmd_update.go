package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dominhduc/agent-brain/internal/knowledge"
	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/skill"
	"github.com/dominhduc/agent-brain/internal/updater"
)

func cmdUpdate() {
	skillsFlag := hasFlag("--skills")
	installFlag := hasFlag("--install")
	globalFlag := hasFlag("--global")
	listFlag := hasFlag("--list")
	diffFlag := hasFlag("--diff")
	reflectFlag := hasFlag("--reflect")
	dryRunFlag := hasFlag("--dry-run")

	if skillsFlag {
		cmdUpdateSkills(installFlag, globalFlag, listFlag, diffFlag, reflectFlag, dryRunFlag)
		return
	}

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

	checksums := updater.ParseChecksums(release.Body)
	release.Checksums = checksums

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

	if err := updater.ReplaceBinary(archiveData, asset.Name, resolvedPath, checksums); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to %s successfully!\n", release.TagName)

	brainDir, brainErr := knowledge.FindBrainDir()
	wasRunning := false
	if brainErr == nil {
		workDir := filepath.Dir(brainDir)
		wasRunning = service.IsRunning(workDir)
	}

	if wasRunning {
		service.StopCurrentProject()
		if brainErr == nil {
			workDir := filepath.Dir(brainDir)
			if err := service.Start(workDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not restart daemon: %v\n", err)
			} else {
				fmt.Println("Daemon restarted.")
			}
		}
	} else {
		service.StopCurrentProject()
		fmt.Println("Restart the daemon with: brain daemon start")
	}

	cwd, _ := os.Getwd()
	if brainErr == nil {
		if err := skill.UpdateSkills(cwd); err == nil {
			fmt.Println("Skill files updated to latest template.")
		}
	}
}

func cmdUpdateSkills(installFlag, globalFlag, listFlag, diffFlag, reflectFlag, dryRunFlag bool) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine current directory.\n")
		os.Exit(1)
	}

	if listFlag {
		infos := skill.ListInstalled(cwd)
		fmt.Println("Agent skill installation status:")
		fmt.Println()

		projectFound := false
		globalFound := false

		for _, info := range infos {
			label := "Project"
			if info.Global {
				label = "Global"
			}

			if info.Installed {
				if info.Global {
					globalFound = true
				} else {
					projectFound = true
				}
			}

			status := "not installed"
			if info.Installed {
				status = "installed"
				if info.Modified {
					status += " (modified)"
				}
			}

			fmt.Printf("  %-8s %-50s %s\n", label, info.Path, status)
		}

		fmt.Println()
		if !projectFound && !globalFound {
			fmt.Println("No skills installed. Run 'brain update --skills --install' to install.")
		}
		return
	}

	if diffFlag {
		diff, err := skill.ShowDiff()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error computing diff: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(diff))
		return
	}

	if installFlag {
		if globalFlag {
			results, err := skill.InstallGlobal()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error installing global skills: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Installing skills to global directories...")
			for _, r := range results {
				if r.Skipped {
					fmt.Printf("  ✓ %s (already exists, skipped)\n", r.Path)
				} else if r.Written {
					fmt.Printf("  ✓ %s\n", r.Path)
				} else if r.Error != nil {
					fmt.Printf("  ✗ %s: %v\n", r.Path, r.Error)
				}
			}
		} else {
			results := skill.InstallProject(cwd)

			fmt.Println("Installing skills to project directories...")
			for _, r := range results {
				if r.Skipped {
					fmt.Printf("  ✓ %s (already exists, skipped)\n", r.Path)
				} else if r.Written {
					fmt.Printf("  ✓ %s\n", r.Path)
				} else if r.Error != nil {
					fmt.Printf("  ✗ %s: %v\n", r.Path, r.Error)
				}
			}
		}
		return
	}

	if reflectFlag {
		brainDir, err := knowledge.FindBrainDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		hub, err := knowledge.Open(brainDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		adaptation, err := hub.GenerateAdaptation()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating adaptation: %v\n", err)
			os.Exit(1)
		}

		content := hub.FormatAdaptation(adaptation)
		if strings.TrimSpace(content) == "" {
			fmt.Println("No adaptations to generate. Use the tool more to accumulate behavior data.")
			return
		}

		if dryRunFlag {
			fmt.Println("=== Dry Run: Would append to SKILL.md ===")
			fmt.Println(content)
			return
		}

		updated, err := skill.WriteAdaptations(cwd, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing adaptations: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Skill adaptations updated (%d file%s).\n", updated, plural(updated))
		return
	}

	infos := skill.ListInstalled(cwd)
	var modified []skill.SkillInfo
	for _, info := range infos {
		if info.Installed && info.Modified {
			modified = append(modified, info)
		}
	}

	if len(modified) > 0 {
		fmt.Println("The following skill files have local modifications:")
		for _, m := range modified {
			label := "Project"
			if m.Global {
				label = "Global"
			}
			fmt.Printf("  [%s] %s\n", label, m.Path)
		}
		fmt.Println()
		fmt.Println("Updating will overwrite local changes (adaptations inside markers are preserved).")

		if skill.HasUncommittedChanges(cwd) {
			fmt.Println("Warning: You also have uncommitted git changes to skill files.")
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Continue? [y/N] ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice != "y" && choice != "Y" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	if err := skill.UpdateSkills(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating skills: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Skill files updated successfully.")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
