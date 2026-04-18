package knowledge

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectWorkContext_NoGit(t *testing.T) {
	tmpDir := t.TempDir()
	topics, err := DetectWorkContext(tmpDir)
	if err != nil {
		t.Errorf("expected nil error outside git repo, got %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("expected no topics outside git repo, got %v", topics)
	}
}

func TestDetectWorkContext_CleanRepo(t *testing.T) {
	tmpDir := t.TempDir()
	if err := execCommand(tmpDir, "git", "init"); err != nil {
		t.Skip("git not available")
	}
	if err := execCommand(tmpDir, "git", "config", "user.email", "test@test.com"); err != nil {
		t.Skip("git not available")
	}
	if err := execCommand(tmpDir, "git", "config", "user.name", "test"); err != nil {
		t.Skip("git not available")
	}

	topics, err := DetectWorkContext(tmpDir)
	if err != nil {
		t.Errorf("expected nil error in clean repo, got %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("expected no topics in clean repo, got %v", topics)
	}
}

func TestDetectWorkContext_WithChanges(t *testing.T) {
	tmpDir := t.TempDir()
	if err := execCommand(tmpDir, "git", "init"); err != nil {
		t.Skip("git not available")
	}
	if err := execCommand(tmpDir, "git", "config", "user.email", "test@test.com"); err != nil {
		t.Skip("git not available")
	}
	if err := execCommand(tmpDir, "git", "config", "user.name", "test"); err != nil {
		t.Skip("git not available")
	}

	file := filepath.Join(tmpDir, "auth.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := execCommand(tmpDir, "git", "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := execCommand(tmpDir, "git", "commit", "-m", "init"); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "auth.go"), []byte("package main\n\nfunc auth() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	topics, err := DetectWorkContext(tmpDir)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	hasSecurity := false
	for _, t := range topics {
		if t == "security" || t == "backend" {
			hasSecurity = true
		}
	}
	if !hasSecurity {
		t.Errorf("expected security or backend topic for auth.go, got %v", topics)
	}
}

func execCommand(dir string, name string, args ...string) error {
	cmd := execCommandImpl(dir, name, args...)
	return cmd.Run()
}

func execCommandImpl(dir string, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd
}
