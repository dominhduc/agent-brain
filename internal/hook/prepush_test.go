package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPrePushHook_CreatesHook(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")

	err := InstallPrePushHook(dir)
	if err != nil {
		t.Fatalf("InstallPrePushHook() error = %v", err)
	}

	hookPath := filepath.Join(gitDir, "pre-push")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("hook file not found: %v", err)
	}
	if info.Mode().Perm()&0700 != 0700 {
		t.Errorf("hook not executable, got mode %v", info.Mode().Perm())
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("cannot read hook: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("hook file is empty")
	}
}

func TestInstallPrePushHook_SkipsExistingAgentBrain(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(hooksDir, "pre-push")
	if err := os.WriteFile(hookPath, []byte("# agent-brain pre-push\nexit 0"), 0700); err != nil {
		t.Fatal(err)
	}

	err := InstallPrePushHook(dir)
	if err != nil {
		t.Fatalf("InstallPrePushHook() error = %v", err)
	}

	content, _ := os.ReadFile(hookPath)
	if string(content) != "# agent-brain pre-push\nexit 0" {
		t.Errorf("hook should not be overwritten, got %q", string(content))
	}
}

func TestInstallPrePushHook_BackupsNonAgentBrain(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	original := "#!/bin/bash\n# some other hook\necho 'hello'\n"
	hookPath := filepath.Join(hooksDir, "pre-push")
	if err := os.WriteFile(hookPath, []byte(original), 0700); err != nil {
		t.Fatal(err)
	}

	err := InstallPrePushHook(dir)
	if err != nil {
		t.Fatalf("InstallPrePushHook() error = %v", err)
	}

	backupPath := hookPath + ".bak"
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup not found: %v", err)
	}
	if string(backupContent) != original {
		t.Errorf("backup content mismatch, got %q", string(backupContent))
	}

	currentContent, _ := os.ReadFile(hookPath)
	if len(currentContent) == 0 {
		t.Error("hook should be replaced after backup")
	}
}

func TestInstallPrePushHook_ContentContainsKeyPhrases(t *testing.T) {
	dir := t.TempDir()

	if err := InstallPrePushHook(dir); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "pre-push"))
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)

	checks := []string{
		"Pre-push",
		"agent-brain",
		"BRAIN_DIR",
		"QUEUE_DIR",
		"while read local_ref",
		"remote_sha..$local_sha",
		"commit-",
		"timestamp",
		"diff_stat",
		"pending",
	}

	for _, phrase := range checks {
		if !contains(s, phrase) {
			t.Errorf("hook content missing %q", phrase)
		}
	}
}

func TestInstallPrePushHook_NewBranchDiff(t *testing.T) {
	content := PrePushHookContent
	if !contains(content, "4b825dc642cb6eb9a060e54bf899d69f8272690f") {
		t.Error("hook should diff against empty tree hash for new branches")
	}
	if !contains(content, "0000000000000000000000000000000000000000") {
		t.Error("hook should check for zero sha (new branch)")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
