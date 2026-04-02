package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CheckGitInstalled() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed.\nWhat to do: Install git from https://git-scm.com/downloads")
	}
	return nil
}

func CheckGitRepo(cwd string) error {
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository.\nWhat to do: Run 'git init' first, then run 'brain init' again.")
	}
	return nil
}

func CheckHasCommits(cwd string) error {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "HEAD")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no commits found in this repository.\nWhat to do: Run 'git add . && git commit -m \"initial\"' first, then run 'brain init' again.")
	}
	return nil
}

func CheckLocalBinInPath() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	localBin := filepath.Join(home, ".local/bin")
	path := os.Getenv("PATH")
	for _, dir := range strings.Split(path, ":") {
		if dir == localBin {
			return true
		}
	}
	return false
}

func CheckSafeDirectory(cwd string) error {
	cmd := exec.Command("git", "-C", cwd, "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "dubious ownership") {
			return fmt.Errorf("git ownership mismatch detected.\nWhat to do: Run 'git config --global --add safe.directory %s'", cwd)
		}
		return fmt.Errorf("git error: %s\nWhat to do: Check that you have access to this directory.", strings.TrimSpace(string(output)))
	}
	return nil
}

func RunAll(cwd string) []string {
	var warnings []string

	if err := CheckGitInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := CheckGitRepo(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := CheckSafeDirectory(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := CheckHasCommits(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !CheckLocalBinInPath() {
		warnings = append(warnings, "~/.local/bin is not in your PATH. Run 'export PATH=\"$HOME/.local/bin:$PATH\"' or add it to your shell profile (~/.bashrc).")
	}

	return warnings
}
