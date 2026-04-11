package skill

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var SkillFS embed.FS

type InstallResult struct {
	Path    string
	Skipped bool
	Written bool
	Error   error
}

type SkillInfo struct {
	Path      string
	Installed bool
	Modified  bool
	Global    bool
}

func platformDirs(cwd string) []string {
	return []string{
		filepath.Join(cwd, ".opencode", "skills", "agent-brain"),
		filepath.Join(cwd, ".claude", "skills", "agent-brain"),
		filepath.Join(cwd, ".agents", "skills", "agent-brain"),
	}
}

func globalDirs() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(home, ".config", "opencode", "skills", "agent-brain"),
		filepath.Join(home, ".claude", "skills", "agent-brain"),
		filepath.Join(home, ".agents", "skills", "agent-brain"),
	}, nil
}

func InstallProject(cwd string) []InstallResult {
	dirs := platformDirs(cwd)
	var results []InstallResult

	for _, dir := range dirs {
		result := installToDir(dir)
		results = append(results, result)
	}

	return results
}

func InstallGlobal() ([]InstallResult, error) {
	dirs, err := globalDirs()
	if err != nil {
		return nil, err
	}

	var results []InstallResult
	for _, dir := range dirs {
		result := installToDir(dir)
		results = append(results, result)
	}

	return results, nil
}

func installToDir(dir string) InstallResult {
	result := InstallResult{Path: dir}

	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); err == nil {
		result.Skipped = true
		return result
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		result.Error = fmt.Errorf("cannot create directory: %w", err)
		return result
	}

	err := fs.WalkDir(SkillFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "templates" {
			return nil
		}

		relPath := strings.TrimPrefix(path, "templates/")
		if relPath == "" || relPath == path {
			return nil
		}

		targetPath := filepath.Join(dir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := SkillFS.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return err
		}

		result.Written = true
		return nil
	})

	if err != nil {
		result.Error = err
	}

	return result
}

func ListInstalled(cwd string) []SkillInfo {
	var infos []SkillInfo

	for _, dir := range platformDirs(cwd) {
		infos = append(infos, checkSkillDir(dir, false))
	}

	gDirs, err := globalDirs()
	if err == nil {
		for _, dir := range gDirs {
			infos = append(infos, checkSkillDir(dir, true))
		}
	}

	return infos
}

func checkSkillDir(dir string, global bool) SkillInfo {
	info := SkillInfo{Path: dir, Global: global}
	skillPath := filepath.Join(dir, "SKILL.md")

	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return info
	}

	info.Installed = true

	data, err := os.ReadFile(skillPath)
	if err != nil {
		return info
	}

	templateData, err := SkillFS.ReadFile("templates/SKILL.md")
	if err != nil {
		return info
	}

	info.Modified = !bytes.Equal(data, templateData)

	return info
}

func ShowDiff() ([]byte, error) {
	var output bytes.Buffer

	templateData, err := SkillFS.ReadFile("templates/SKILL.md")
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()
	checkDirs := []struct {
		path  string
		label string
	}{
		{filepath.Join(cwd, ".opencode", "skills", "agent-brain", "SKILL.md"), "Project (.opencode)"},
		{filepath.Join(cwd, ".claude", "skills", "agent-brain", "SKILL.md"), "Project (.claude)"},
		{filepath.Join(cwd, ".agents", "skills", "agent-brain", "SKILL.md"), "Project (.agents)"},
		{filepath.Join(home, ".config", "opencode", "skills", "agent-brain", "SKILL.md"), "Global (.config/opencode)"},
		{filepath.Join(home, ".claude", "skills", "agent-brain", "SKILL.md"), "Global (.claude)"},
		{filepath.Join(home, ".agents", "skills", "agent-brain", "SKILL.md"), "Global (.agents)"},
	}

	hasDiff := false

	for _, check := range checkDirs {
		installedData, err := os.ReadFile(check.path)
		if err != nil {
			continue
		}

		if bytes.Equal(installedData, templateData) {
			continue
		}

		hasDiff = true
		output.WriteString(fmt.Sprintf("\n--- %s\n+++ templates/SKILL.md (latest)\n", check.label))
		output.WriteString(generateDiff(string(installedData), string(templateData)))
	}

	if !hasDiff {
		output.WriteString("No differences found. Skill files are up to date.\n")
	}

	return output.Bytes(), nil
}

func generateDiff(old, new string) string {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	var output strings.Builder

	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		if oldLines[i] == newLines[j] {
			i++
			j++
		} else {
			output.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
			output.WriteString(fmt.Sprintf("+%s\n", newLines[j]))
			i++
			j++
		}
	}

	for i < len(oldLines) {
		output.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
		i++
	}

	for j < len(newLines) {
		output.WriteString(fmt.Sprintf("+%s\n", newLines[j]))
		j++
	}

	return output.String()
}

func UpdateSkills(cwd string) error {
	dirs := platformDirs(cwd)

	for _, dir := range dirs {
		skillPath := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			continue
		}

		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("cannot remove %s: %w", dir, err)
		}

		result := installToDir(dir)
		if result.Error != nil {
			return fmt.Errorf("cannot install to %s: %w", dir, result.Error)
		}
	}

	return nil
}

func HasUncommittedChanges(cwd string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "/skills/agent-brain/") ||
			strings.Contains(line, "/.opencode/skills/") ||
			strings.Contains(line, "/.claude/skills/") ||
			strings.Contains(line, "/.agents/skills/") {
			return true
		}
	}

	return false
}
