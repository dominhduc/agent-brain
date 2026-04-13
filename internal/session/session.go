package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Handoff struct {
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	Next      string    `json:"next"`
	Session   string    `json:"session"`
	Timestamp time.Time `json:"timestamp"`
	Topic     string    `json:"topic"`
}

type Session struct {
	brainDir string
}

func Open(brainDir string) *Session {
	return &Session{brainDir: brainDir}
}

func (s *Session) Dir() string {
	return s.brainDir
}

func (s *Session) CreateHandoff(summary, next, sessionID, topic string) (*Handoff, error) {
	dir := filepath.Join(s.brainDir, "handoffs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating handoff dir: %w", err)
	}

	h := &Handoff{
		ID:        fmt.Sprintf("handoff-%s", time.Now().Format("20060102-150405")),
		Summary:   summary,
		Next:      next,
		Session:   sessionID,
		Timestamp: time.Now(),
		Topic:     topic,
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling handoff: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "latest.json"), data, 0600); err != nil {
		return nil, fmt.Errorf("writing handoff: %w", err)
	}

	return h, nil
}

func (s *Session) LatestHandoff() (*Handoff, error) {
	path := filepath.Join(s.brainDir, "handoffs", "latest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading handoff: %w", err)
	}

	var h Handoff
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, nil
	}
	return &h, nil
}

func (s *Session) ResumeHandoff() (*Handoff, error) {
	return s.LatestHandoff()
}

type GitStats struct {
	DiffStat   string
	ShortStat  string
	Log        string
	Diff       string
	Created    []string
	Modified   []string
	Deleted    []string
}

func CollectGitStats(cwd string) (*GitStats, error) {
	if !isGitRepo(cwd) {
		return nil, fmt.Errorf("not a git repository.\nWhat to do: run 'git init' first.")
	}

	hasParent := runGit(cwd, "rev-parse", "HEAD~1") != ""
	diffArgs := []string{"diff", "--stat", "HEAD~1"}
	nameArgs := []string{"diff", "--name-status", "HEAD~1"}
	shortArgs := []string{"diff", "--shortstat", "HEAD~1"}
	if !hasParent {
		emptyTree := "4b825dc642cb6eb9a060e54bf899d69f8272690f"
		diffArgs = []string{"diff", "--stat", emptyTree, "HEAD"}
		nameArgs = []string{"diff", "--name-status", emptyTree, "HEAD"}
		shortArgs = []string{"diff", "--shortstat", emptyTree, "HEAD"}
	}

	stats := &GitStats{
		DiffStat:  runGit(cwd, diffArgs...),
		ShortStat: strings.TrimSpace(runGit(cwd, shortArgs...)),
		Log:       strings.TrimSpace(runGit(cwd, "log", "--oneline", "-3")),
		Diff:      runGit(cwd, "diff", "HEAD~1"),
	}

	nameStatus := runGit(cwd, nameArgs...)
	for _, line := range strings.Split(nameStatus, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "A":
			stats.Created = append(stats.Created, parts[1])
		case "M":
			stats.Modified = append(stats.Modified, parts[1])
		case "D":
			stats.Deleted = append(stats.Deleted, parts[1])
		}
	}

	return stats, nil
}

func (s *Session) CreateSessionFile(stats *GitStats) (string, error) {
	sessionsDir := filepath.Join(s.brainDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return "", fmt.Errorf("creating sessions dir: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	sessionPath := filepath.Join(sessionsDir, timestamp+".md")

	content := fmt.Sprintf("# Session — %s\n\n## Git Summary\n- Files created: %s\n- Files modified: %s\n- Files deleted: %s\n- Total changes: %s\n\n## Recent Commits\n%s\n\n---\n<!-- Agent: append your evaluation below this line -->\n",
		time.Now().Format("2006-01-02 15:04:05"),
		formatList(stats.Created),
		formatList(stats.Modified),
		formatList(stats.Deleted),
		stats.ShortStat,
		stats.Log,
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("creating session file: %w", err)
	}

	return sessionPath, nil
}

func isGitRepo(cwd string) bool {
	return exec.Command("git", "-C", cwd, "rev-parse", "--git-dir").Run() == nil
}

func runGit(cwd string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return string(output)
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return "`" + strings.Join(items, "`, `") + "`"
}
