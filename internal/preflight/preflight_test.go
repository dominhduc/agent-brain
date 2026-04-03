package preflight

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckGitInstalled(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed, skipping")
	}
	if err := CheckGitInstalled(); err != nil {
		t.Errorf("expected no error when git is installed: %v", err)
	}
}

func TestCheckGitRepo_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)

	if err := CheckGitRepo(tmpDir); err != nil {
		t.Errorf("expected no error for valid git repo: %v", err)
	}
}

func TestCheckGitRepo_Invalid(t *testing.T) {
	tmpDir := t.TempDir()

	if err := CheckGitRepo(tmpDir); err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestCheckHasCommits_WithCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	tmpDir := t.TempDir()
	exec.Command("git", "init", tmpDir).Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "test").Run()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	if err := CheckHasCommits(tmpDir); err != nil {
		t.Errorf("expected no error with commits: %v", err)
	}
}

func TestCheckHasCommits_NoCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	tmpDir := t.TempDir()
	exec.Command("git", "init", tmpDir).Run()

	if err := CheckHasCommits(tmpDir); err == nil {
		t.Error("expected error for repo with no commits")
	}
}

func TestCheckLocalBinInPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local/bin")
	path := os.Getenv("PATH")

	result := CheckLocalBinInPath()
	expected := false
	for _, dir := range filepath.SplitList(path) {
		if dir == localBin {
			expected = true
			break
		}
	}
	if result != expected {
		t.Errorf("CheckLocalBinInPath() = %v, want %v", result, expected)
	}
}

func TestCheckSafeDirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	tmpDir := t.TempDir()
	exec.Command("git", "init", tmpDir).Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "test").Run()

	if err := CheckSafeDirectory(tmpDir); err != nil {
		t.Errorf("expected no error for safe directory: %v", err)
	}
}
