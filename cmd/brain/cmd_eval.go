package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/handoff"
	"github.com/dominhduc/agent-brain/internal/index"
	"github.com/dominhduc/agent-brain/internal/outcome"
	"github.com/dominhduc/agent-brain/internal/wm"
)

func cmdEval() {
	good := hasFlag("--good")
	bad := hasFlag("--bad")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine current directory.\n")
		os.Exit(1)
	}

	if !brain.IsGitRepo(cwd) {
		fmt.Fprintln(os.Stderr, "Error: this doesn't appear to be a git repository.")
		fmt.Fprintln(os.Stderr, "What to do: run 'git init' first.")
		os.Exit(1)
	}

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	diffArgs := []string{"diff", "--stat", "HEAD~1"}
	nameArgs := []string{"diff", "--name-status", "HEAD~1"}
	shortArgs := []string{"diff", "--shortstat", "HEAD~1"}
	if runGit(cwd, "rev-parse", "HEAD~1") == "" {
		emptyTree := "4b825dc642cb6eb9a060e54bf899d69f8272690f"
		diffArgs = []string{"diff", "--stat", emptyTree, "HEAD"}
		nameArgs = []string{"diff", "--name-status", emptyTree, "HEAD"}
		shortArgs = []string{"diff", "--shortstat", emptyTree, "HEAD"}
	}
	diffStat := runGit(cwd, diffArgs...)
	nameStatus := runGit(cwd, nameArgs...)
	shortstat := runGit(cwd, shortArgs...)
	log := runGit(cwd, "log", "--oneline", "-3")
	diffContent := runGit(cwd, "diff", "HEAD~1")

	if strings.TrimSpace(diffStat) == "" {
		fmt.Println("No file changes detected since last session.")
		fmt.Println("What to do: make a commit first, then run 'brain eval'.")
		return
	}

	var created, modified, deleted []string
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
			created = append(created, parts[1])
		case "M":
			modified = append(modified, parts[1])
		case "D":
			deleted = append(deleted, parts[1])
		}
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	sessionsDir := filepath.Join(brainDir, "sessions")
	sessionPath := filepath.Join(sessionsDir, timestamp+".md")

	content := fmt.Sprintf("# Session — %s\n\n## Git Summary\n- Files created: %s\n- Files modified: %s\n- Files deleted: %s\n- Total changes: %s\n\n## Recent Commits\n%s\n\n---\n<!-- Agent: append your evaluation below this line -->\n",
		time.Now().Format("2006-01-02 15:04:05"),
		formatList(created),
		formatList(modified),
		formatList(deleted),
		strings.TrimSpace(shortstat),
		strings.TrimSpace(log),
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session file: %v\nWhat to do: check permissions on .brain/sessions/.\n", err)
		os.Exit(1)
	}

	relPath, _ := filepath.Rel(cwd, sessionPath)
	fmt.Printf("Session file created: %s\n", relPath)

	topic := detectTopicFromDiff(diffContent)
	summary := fmt.Sprintf("Modified %d files (%d created, %d modified, %d deleted)", len(created)+len(modified)+len(deleted), len(created), len(modified), len(deleted))
	next := "Review session evaluation and continue work."

	h, err := handoff.Create(brainDir, summary, next, timestamp, topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create handoff: %v\n", err)
	} else {
		fmt.Printf("Handoff created: %s (topic: %s)\n", h.ID, topic)
	}

	if good || bad {
		keys, _ := outcome.LoadKeys(brainDir)
		idx, err := index.Load(brainDir)
		if err == nil {
			var adjusted int
			for _, key := range keys {
				entry, ok := idx.GetByRawKey(key)
				if !ok {
					continue
				}
				if good {
					entry.HalfLifeDays += 5
				} else if bad {
					entry.HalfLifeDays = max(1, entry.HalfLifeDays-3)
				}
				idx.SetByRawKey(key, entry)
				adjusted++
			}
			if err := idx.Save(brainDir); err == nil {
				fmt.Printf("Applied %s outcome to %d entries\n", map[bool]string{true: "positive", false: "negative"}[good], adjusted)
			}
			outcome.Clear(brainDir)
		}
	}

	if err := wm.Clear(brainDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not clear working memory: %v\n", err)
	} else {
		fmt.Println("Working memory flushed.")
	}

	fmt.Println("Append your evaluation (objective, work completed, self-evaluation, lessons learned).")
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

func detectTopicFromDiff(diff string) string {
	topics := index.DetectTopics(diff)
	if len(topics) > 0 {
		return topics[0]
	}
	return "general"
}
