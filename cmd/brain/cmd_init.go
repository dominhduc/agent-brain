package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/hook"
	"github.com/dominhduc/agent-brain/internal/preflight"
	"github.com/dominhduc/agent-brain/internal/service"
)

func cmdInit() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine current directory.\nWhat to do: make sure you are in a valid directory and try again.\n")
		os.Exit(1)
	}

	warnings := preflight.RunAll(cwd)

	if brain.BrainDirExists(cwd) {
		fmt.Println("Knowledge hub already exists in this project.")
		fmt.Println("What to do: to reinitialize, remove the .brain/ directory first: rm -rf .brain/")
		return
	}

	fmt.Println("Initializing knowledge hub...")

	if err := brain.EnsureBrainDir(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .brain/ directory: %v\nWhat to do: check directory permissions.\n", err)
		os.Exit(1)
	}

	skeletons := map[string]string{
		"MEMORY.md": "# Project Memory Index\n\n" +
			"## Project\n[Auto-populated by daemon analysis or first agent session]\n\n" +
			"## Stack\n[Auto-populated by daemon analysis or first agent session]\n\n" +
			"## Commands\n[Auto-populated by daemon analysis or first agent session]\n\n" +
			"## Key Patterns\n[Auto-populated by daemon analysis]\n\n" +
			"## Active Gotchas\n[Auto-populated by daemon analysis]\n\n" +
			"## Topic Files\n- gotchas.md — Error patterns and fixes\n- patterns.md — Discovered conventions\n- architecture.md — Module structure and relationships\n- decisions.md — Architecture decisions and rationale\n\n" +
			"## Last Updated\n[Auto-updated]\n",
		"gotchas.md":      "# Gotchas\n<!-- Entries added by brain add gotcha or daemon analysis -->\n",
		"patterns.md":     "# Patterns\n<!-- Entries added by brain add pattern or daemon analysis -->\n",
		"architecture.md": "# Architecture\n<!-- Entries added by brain add architecture or daemon analysis -->\n",
		"decisions.md":    "# Decisions\n<!-- Entries added by brain add decision or daemon analysis -->\n",
	}

	brainDir := filepath.Join(cwd, ".brain")
	for name, content := range skeletons {
		path := filepath.Join(brainDir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\nWhat to do: check file permissions.\n", name, err)
			os.Exit(1)
		}
	}

	sessionsDir := filepath.Join(brainDir, "sessions")
	if err := os.WriteFile(filepath.Join(sessionsDir, ".gitkeep"), []byte{}, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .gitkeep: %v\nWhat to do: check permissions on .brain/sessions/.\n", err)
		os.Exit(1)
	}

	agentsPath := filepath.Join(cwd, "AGENTS.md")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		if err := os.WriteFile(agentsPath, []byte(agentsTemplate), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating AGENTS.md: %v\nWhat to do: check file permissions.\n", err)
			os.Exit(1)
		}
	}

	wrappers := map[string]string{
		"CLAUDE.md":      "See AGENTS.md for complete project instructions and knowledge base.\n",
		".cursorrules":   "See AGENTS.md for complete project instructions and knowledge base.\n",
		".windsurfrules": "See AGENTS.md for complete project instructions and knowledge base.\n",
	}
	for name, content := range wrappers {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating %s: %v\nWhat to do: check file permissions.\n", name, err)
				os.Exit(1)
			}
		}
	}

	gitignorePath := filepath.Join(cwd, ".gitignore")
	entries := []string{".brain/archived/", ".brain/.queue/", ".brain/pending/"}
	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	}
	var newEntries []string
	for _, entry := range entries {
		if !strings.Contains(existing, entry) {
			newEntries = append(newEntries, entry)
		}
	}
	if len(newEntries) > 0 {
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating .gitignore: %v\nWhat to do: check file permissions.\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if !strings.HasSuffix(existing, "\n") && existing != "" {
			f.WriteString("\n")
		}
		f.WriteString("# agent-brain\n")
		for _, entry := range newEntries {
			f.WriteString(entry + "\n")
		}
	}

	if err := installGitHook(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not install git hook: %v\nWhat to do: the daemon will still work, but commits won't be auto-queued.\n", err)
	}

	if err := hook.InstallPrePushHook(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not install pre-push hook: %v\nWhat to do: the daemon will still work with the post-commit hook, but pre-push is recommended.\n", err)
	} else {
		fmt.Println("Pre-push hook installed (analyzes only stable shifted code).")
	}

	apiKey := config.GetAPIKey()
	if apiKey == "" {
		fmt.Println("\nOpenRouter API key not configured.")
		fmt.Println("What to do: run 'brain config set api-key <your-openrouter-key>'")
		fmt.Println("The daemon is registered but won't process commits until you set a key.")
	}

	if _, err := os.Stat(config.ConfigPath()); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		if apiKey != "" {
			cfg.LLM.APIKey = apiKey
		}
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create config file: %v\nWhat to do: check permissions on ~/.config/brain/.\n", err)
		}
	}

	fmt.Println("\nWarning: Diffs are sent to OpenRouter for analysis.")
	fmt.Println("brain will scan for secrets before sending. If secrets are detected, the commit is skipped for safety.")

	fmt.Println("\nRegistering daemon service...")
	execPath, err := os.Executable()
	if err != nil {
		execPath = "brain"
	}
	if err := service.Register(execPath, cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not register daemon: %v\nWhat to do: start manually with 'brain daemon start'.\n", err)
	} else {
		fmt.Println("Daemon registered and started.")
	}

	brain.ResetCache()

	fmt.Println("\nKnowledge hub initialized successfully!")
	fmt.Println("First commit will auto-populate knowledge via the daemon.")
	fmt.Println("Run 'brain status' to see hub statistics.")

	for _, w := range warnings {
		fmt.Printf("\nWarning: %s\n", w)
	}
}

func installGitHook(cwd string) error {
	hookContent := `#!/bin/bash
# Post-commit hook for agent-brain
# Installed by: brain init

BRAIN_DIR=".brain"
QUEUE_DIR="$BRAIN_DIR/.queue"

if [ ! -d "$QUEUE_DIR" ]; then
    exit 0
fi

TIMESTAMP=$(date +%Y%m%dT%H%M%S)

if git rev-parse HEAD~1 >/dev/null 2>&1; then
    DIFF_STAT=$(git diff --stat HEAD~1)
    FILES=$(git diff --name-status HEAD~1)
else
    EMPTY_TREE=$(git hash-object -t tree /dev/null)
    DIFF_STAT=$(git diff --stat $EMPTY_TREE HEAD)
    FILES=$(git diff --name-status $EMPTY_TREE HEAD)
fi
REPO=$(pwd)

escape_json() {
    printf '%s' "$1" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))' 2>/dev/null || printf '"%s"' "$(echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | tr '\n' ' ')"
}

DIFF_ESCAPED=$(escape_json "$DIFF_STAT")
FILES_ESCAPED=$(escape_json "$FILES")

cat > "$QUEUE_DIR/commit-${TIMESTAMP}.json" << EOF
{
  "timestamp": "${TIMESTAMP}",
  "repo": "${REPO}",
  "diff_stat": ${DIFF_ESCAPED},
  "files": ${FILES_ESCAPED},
  "attempts": 0,
  "status": "pending"
}
EOF
`

	hooksDir := filepath.Join(cwd, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	hookPath := filepath.Join(hooksDir, "post-commit")

	if _, err := os.Stat(hookPath); err == nil {
		existing, err := os.ReadFile(hookPath)
		if err != nil {
			return fmt.Errorf("cannot read existing hook: %w", err)
		}
		if strings.Contains(string(existing), "agent-brain") {
			return nil
		}
		backupPath := hookPath + ".bak"
		if err := os.Rename(hookPath, backupPath); err != nil {
			return fmt.Errorf("cannot back up existing hook: %w", err)
		}
		fmt.Printf("Existing post-commit hook backed up to %s\n", backupPath)
	}

	return os.WriteFile(hookPath, []byte(hookContent), 0700)
}

const agentsTemplate = "# Project Instructions\n\n" +
	"## Knowledge Base\n" +
	"This project uses a `.brain/` knowledge hub managed by the `brain` CLI.\n\n" +
	"### At Session Start\n" +
	"Run `brain get all` to load accumulated project knowledge before starting work.\n\n" +
	"### During Work\n" +
	"- Run `brain search <topic>` before writing code against unfamiliar patterns\n" +
	"- Run `brain get gotchas` before debugging to avoid known pitfalls\n\n" +
	"### Self-Evolution\n" +
	"When I correct you, express frustration about a repeated mistake, or point out a pattern:\n" +
	"1. Add the learning: `brain add gotcha \"...\"` or `brain add pattern \"...\"`\n" +
	"2. Update MEMORY.md if the index needs refreshing\n" +
	"3. Treat every correction as permanent — don't repeat mistakes\n\n" +
	"### At Session End\n" +
	"1. Run `brain eval` to write a self-evaluation to the current session file.\n" +
	"   Include: what you did, what worked, what failed, confidence scores, knowledge persisted.\n" +
	"2. Run `brain status` to check MEMORY.md line count.\n" +
	"   If over 200 lines, run `brain prune --dry-run` to preview stale entries.\n" +
	"   Ask the user before pruning. Only prune with approval.\n\n" +
	"### Maintenance\n" +
	"Run periodically (or when MEMORY.md exceeds 200 lines):\n\n" +
	"- `brain status` — check health, queue state, and line counts\n" +
	"- `brain prune --dry-run` — preview what would be removed from topic files\n" +
	"- `brain prune` — archive matching entries to `.brain/archived/`\n" +
	"- `brain review` — approve/reject pending daemon-analyzed entries\n\n" +
	"**Prune patterns (.brainprune):** Create a `.brainprune` file at project root\n" +
	"with one pattern per line. Topic file lines matching any pattern get archived.\n" +
	"Use patterns for outdated entries: old version references, resolved issues,\n" +
	"or entries no longer relevant to the current codebase.\n\n" +
	"Example .brainprune:\n" +
	"```\n" +
	"# Old patterns no longer relevant\n" +
	"v0.3.0\n" +
	"old-api-endpoint\n" +
	"deprecated function name\n" +
	"```\n\n" +
	"### Confidence Reporting\n" +
	"Always report confidence on technical decisions:\n" +
	"- HIGH: documented best practice, matches codebase patterns\n" +
	"- MEDIUM: reasonable approach, alternatives exist\n" +
	"- LOW: best guess, recommend verification\n" +
	"When confidence is below HIGH, state what would increase it and the risks.\n\n" +
	"### Clarifying Questions\n" +
	"If requirements are ambiguous, ask BEFORE coding. Present 2-3 options with tradeoffs.\n\n" +
	"### Safety Rules\n" +
	"- NEVER delete files or run destructive commands without explicit approval\n" +
	"- NEVER read or expose `.env` files or secrets\n" +
	"- Flag risky changes (auth, payments, data mutations) and wait for my review\n\n" +
	"## Project Overview\n" +
	"[Auto-populated by daemon analysis or first agent session]\n\n" +
	"## Stack\n" +
	"[Auto-populated by daemon analysis or first agent session]\n\n" +
	"## Commands\n" +
	"[Auto-populated by daemon analysis or first agent session]\n"
