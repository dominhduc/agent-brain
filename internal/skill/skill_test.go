package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallProject(t *testing.T) {
	cwd := t.TempDir()

	results := InstallProject(cwd)
	if len(results) != 3 {
		t.Fatalf("expected 3 results (one per platform), got %d", len(results))
	}

	var written int
	for _, r := range results {
		if r.Written {
			written++
		}
		if r.Skipped {
			t.Error("expected not skipped on first install")
		}
		if r.Error != nil {
			t.Errorf("unexpected error: %v", r.Error)
		}
	}
	if written != 3 {
		t.Errorf("expected 3 written, got %d", written)
	}

	skillPath := filepath.Join(cwd, ".opencode", "skills", "agent-brain", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatal("expected SKILL.md to be installed")
	}
}

func TestInstallProjectSkipsExisting(t *testing.T) {
	cwd := t.TempDir()

	InstallProject(cwd)

	results := InstallProject(cwd)
	for _, r := range results {
		if !r.Skipped {
			t.Errorf("expected skipped on second install")
		}
		if r.Written {
			t.Error("expected not written on second install")
		}
	}
}

func TestInstallGlobal(t *testing.T) {
	results, err := InstallGlobal()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results (one per platform), got %d", len(results))
	}
}

func TestListInstalled(t *testing.T) {
	cwd := t.TempDir()

	infos := ListInstalled(cwd)
	if len(infos) != 6 {
		t.Fatalf("expected 6 skill infos (3 project + 3 global), got %d", len(infos))
	}

	projectInfos := 0
	globalInfos := 0
	for _, info := range infos {
		if info.Global {
			globalInfos++
		} else {
			projectInfos++
		}
	}
	if projectInfos != 3 {
		t.Errorf("expected 3 project infos, got %d", projectInfos)
	}
	if globalInfos != 3 {
		t.Errorf("expected 3 global infos, got %d", globalInfos)
	}
}

func TestListInstalledDetectsInstalled(t *testing.T) {
	cwd := t.TempDir()

	InstallProject(cwd)

	infos := ListInstalled(cwd)
	projectInstalled := 0
	for _, info := range infos {
		if !info.Global && info.Installed {
			projectInstalled++
		}
	}
	if projectInstalled != 3 {
		t.Errorf("expected 3 installed project skills, got %d", projectInstalled)
	}
}

func TestListInstalledDetectsModified(t *testing.T) {
	cwd := t.TempDir()

	InstallProject(cwd)

	skillPath := filepath.Join(cwd, ".opencode", "skills", "agent-brain", "SKILL.md")
	f, err := os.OpenFile(skillPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n# Custom adaptation\n")
	f.Close()

	infos := ListInstalled(cwd)
	for _, info := range infos {
		if info.Path == skillPath {
			if !info.Modified {
				t.Error("expected skill to be detected as modified")
			}
			break
		}
	}
}

func TestExtractAdaptations(t *testing.T) {
	content := `# Skill

Some content.

<!-- brain:adaptations start -->
# Custom section
Custom content.
<!-- brain:adaptations end -->
`
	result := extractAdaptations(content)
	if result == "" {
		t.Fatal("expected non-empty adaptations")
	}
	if !contains(result, "Custom section") {
		t.Error("expected adaptations to include custom section")
	}
}

func TestExtractAdaptationsNoMarkers(t *testing.T) {
	content := "# Skill\n\nSome content.\n"
	result := extractAdaptations(content)
	if result != "" {
		t.Errorf("expected empty adaptations, got %s", result)
	}
}

func TestGenerateDiff(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"

	diff := generateDiff(old, new)
	if !contains(diff, "-line2") {
		t.Error("expected diff to show removed line")
	}
	if !contains(diff, "+modified") {
		t.Error("expected diff to show added line")
	}
}

func TestHasUncommittedChangesNoGit(t *testing.T) {
	dir := t.TempDir()
	result := HasUncommittedChanges(dir)
	if result {
		t.Error("expected false when no git repo")
	}
}

func TestPlatformDirs(t *testing.T) {
	dirs := platformDirs("/some/path")
	if len(dirs) != 3 {
		t.Fatalf("expected 3 platform dirs, got %d", len(dirs))
	}
	expected := []string{
		"/some/path/.opencode/skills/agent-brain",
		"/some/path/.claude/skills/agent-brain",
		"/some/path/.agents/skills/agent-brain",
	}
	for i, d := range dirs {
		if d != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], d)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
