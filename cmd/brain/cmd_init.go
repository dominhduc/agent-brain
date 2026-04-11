package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/hook"
	"github.com/dominhduc/agent-brain/internal/preflight"
	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/skill"
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

	fmt.Println("\nInstalling agent skills...")
	skillResults := skill.InstallProject(cwd)
	for _, r := range skillResults {
		if r.Skipped {
			fmt.Printf("  ✓ %s (already exists, skipped)\n", r.Path)
		} else if r.Written {
			fmt.Printf("  ✓ %s\n", r.Path)
		} else if r.Error != nil {
			fmt.Printf("  ⚠ %s: %v\n", r.Path, r.Error)
		}
	}

	gitignorePath := filepath.Join(cwd, ".gitignore")
	entries := []string{".brain/archived/", ".brain/.queue/", ".brain/pending/", ".brain/index.json", ".brain/buffer/", ".brain/handoffs/", ".brain/.session/"}
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

	configChoice := promptConfigChoice(brainDir)
	if configChoice == "project" {
		cfg := config.DefaultConfig()
		if apiKey != "" {
			cfg.LLM.APIKey = apiKey
		}
		if err := config.SaveToProject(cfg, brainDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create project config file: %v\nWhat to do: check permissions on .brain/.\n", err)
		} else {
			fmt.Printf("Project config created at %s\n", config.ProjectConfigPath(brainDir))
			fmt.Println("This project will use its own isolated configuration.")
		}
	} else {
		if !config.GlobalConfigExists() {
			cfg := config.DefaultConfig()
			if apiKey != "" {
				cfg.LLM.APIKey = apiKey
			}
			if err := config.Save(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not create config file: %v\nWhat to do: check permissions on ~/.config/brain/.\n", err)
			}
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
	"### Session Workflow\n\n" +
	"1. **Start:** Run `brain get all` to load accumulated knowledge\n" +
	"2. **Work:** Run `brain search \"<topic>\"` before writing unfamiliar code\n" +
	"3. **Learn:** Run `brain add <topic> \"<insight>\"` when you discover something\n" +
	"4. **End:** Run `brain eval` to save your work and create a handoff\n\n" +
	"That's it. Working memory, handoffs, and outcome feedback happen automatically.\n\n" +
	"### Topics\n" +
	"Use these topic names when adding entries:\n" +
	"- `ui` — frontend, styling, UX\n" +
	"- `backend` — API, business logic, services\n" +
	"- `infrastructure` — deployment, VPS, CI/CD, Docker\n" +
	"- `database` — schemas, migrations, queries\n" +
	"- `security` — auth, secrets, permissions\n" +
	"- `testing` — tests, mocks, fixtures\n" +
	"- `architecture` — module structure, patterns\n" +
	"- `general` — cross-cutting knowledge\n\n" +
	"### Commands\n" +
	"- `brain get all --focus \"<topic>\"` — Load knowledge filtered by topic\n" +
	"- `brain add <topic> \"<message>\"` — Add entry with topic tag\n" +
	"- `brain add --wm \"<message>\"` — Add to working memory\n" +
	"- `brain eval [--good|--bad]` — End session, create handoff, apply feedback\n" +
	"- `brain search \"<query>\" --topic \"<topic>\"` — Search within a topic\n\n" +
	"### Self-Evolution\n" +
	"When corrected, add the learning: `brain add gotcha|pattern|decision \"<message>\"`\n\n" +
	"### At Session End\n" +
		"Run `brain eval` to write a self-evaluation and create a handoff for the next session.\n"

func promptConfigChoice(brainDir string) string {
	reader := bufio.NewReader(os.Stdin)

	hasGlobal := config.GlobalConfigExists()
	hasProject := config.ProjectConfigExists(brainDir)

	if hasProject {
		return "project"
	}

	if !hasGlobal {
		fmt.Println("\nNo global configuration found.")
		fmt.Println("You can create a global config (shared across all projects) or a project-specific config.")
		fmt.Println()
	} else {
		fmt.Println("\nGlobal configuration found.")
		fmt.Println("You can use the global config (shared across all projects) or create a project-specific config.")
		fmt.Println()
	}

	fmt.Println("Config scope options:")
	fmt.Println("  1. Global config  - Share LLM settings across all projects (default)")
	fmt.Println("  2. Project config - Isolate settings to this project only")
	fmt.Println()
	fmt.Print("Choose config scope (1-2, or press Enter for default): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "2" {
		fmt.Println("\nProject-specific config selected.")
		fmt.Println("Config will be stored in .brain/config.yaml")
		fmt.Println("You can change this later with 'brain config set ...'")
		return "project"
	}

	fmt.Println("\nUsing global config.")
	fmt.Println("You can change this later with 'brain config set ...'")
	return "global"
}
