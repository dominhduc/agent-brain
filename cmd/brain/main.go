package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/analyzer"
	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/daemon"
	"github.com/dominhduc/agent-brain/internal/preflight"
	"github.com/dominhduc/agent-brain/internal/secrets"
	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/updater"
)

const version = "v0.2"

var (
	commit string
	date   string
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	jsonFlag := hasFlag("--json")
	dryRun := hasFlag("--dry-run")

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "get":
		cmdGet(jsonFlag)
	case "search":
		cmdSearch(jsonFlag)
	case "add":
		cmdAdd()
	case "eval":
		cmdEval()
	case "prune":
		cmdPrune(dryRun)
	case "status":
		cmdStatus(jsonFlag)
	case "daemon":
		cmdDaemon()
	case "config":
		cmdConfig()
	case "version", "--version", "-v":
		cmdVersion()
	case "update":
		cmdUpdate()
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[2:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Println(`brain - AI Agent Knowledge Hub CLI

Usage:
  brain init                          Initialize knowledge hub in current project
  brain get <topic> [--json]          Get topic content (memory, gotchas, patterns, decisions, architecture, all)
  brain search <query> [--json]       Search across all knowledge files
  brain add <topic> "<message>"       Add knowledge entry to a topic
  brain eval                          Create session evaluation file
  brain prune [--dry-run]             Archive stale knowledge entries
  brain status [--json]               Show knowledge hub statistics
  brain daemon start|stop|status      Manage background daemon
  brain config [set <key> <value>]    View or set configuration
  brain version                       Show version and build info
  brain update                       Self-update to latest release

Topics: memory, gotchas, patterns, decisions, architecture, all

Examples:
  brain init
  brain get gotchas
  brain search "auth"
  brain add gotcha "Project uses argon2, NOT bcrypt"
  brain eval
  brain status
  brain daemon status
  brain config set llm.api_key sk-or-v1-...`)
}

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
	entries := []string{".brain/archived/", ".brain/.queue/"}
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

	apiKey := config.GetAPIKey()
	if apiKey == "" {
		fmt.Println("\nOpenRouter API key not configured.")
		fmt.Println("What to do: set it with 'brain config set llm.api_key sk-or-v1-...'")
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
	if err := service.Register(execPath); err != nil {
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
DIFF_STAT=$(git diff --stat HEAD~1 2>/dev/null || echo "No previous commit")
FILES=$(git diff --name-status HEAD~1 2>/dev/null || echo "No previous commit")
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

func cmdGet(jsonFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain get <topic>")
		fmt.Println("Topics: memory, gotchas, patterns, decisions, architecture, all")
		fmt.Println("What to do: specify a topic name to retrieve.")
		os.Exit(1)
	}

	topic := os.Args[2]

	if topic == "all" {
		if jsonFlag {
			topics := map[string]string{}
			for _, t := range brain.AvailableTopics() {
				c, err := brain.GetTopic(t)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", t, err)
					os.Exit(1)
				}
				topics[t] = c
			}
			data, _ := json.MarshalIndent(topics, "", "  ")
			fmt.Println(string(data))
		} else {
			content, err := brain.GetAllTopics()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
				os.Exit(1)
			}
			fmt.Println(content)
		}
		return
	}

	content, err := brain.GetTopic(topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonFlag {
		data, _ := json.MarshalIndent(map[string]string{topic: content}, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println(content)
	}
}

func cmdSearch(jsonFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain search <query>")
		fmt.Println("What to do: provide a search term to look for across knowledge files.")
		os.Exit(1)
	}

	query := os.Args[2]

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' in your project directory first.\n", err)
		os.Exit(1)
	}

	files := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))

	type Match struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}

	var matches []Match

	for _, f := range files {
		path := filepath.Join(brainDir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			if pattern.MatchString(line) {
				matches = append(matches, Match{
					File:    f,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
				})
			}
			lineNum++
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No matches found for '%s'\n", query)
		return
	}

	if jsonFlag {
		data, _ := json.MarshalIndent(matches, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Found %d match(es) for '%s':\n\n", len(matches), query)
		for _, m := range matches {
			fmt.Printf("  %s:%d  %s\n", m.File, m.Line, m.Content)
		}
	}
}

func cmdAdd() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: brain add <topic> \"<message>\"")
		fmt.Println("Topics: gotcha, pattern, decision, architecture, memory")
		fmt.Println("What to do: provide a topic and a message to add.")
		os.Exit(1)
	}

	topic := os.Args[2]
	message := strings.Join(os.Args[3:], " ")

	if len(message) > 10240 {
		fmt.Fprintf(os.Stderr, "Error: message too long (%d bytes, max 10240).\nWhat to do: shorten your message or split it into multiple entries.\n", len(message))
		os.Exit(1)
	}

	if secrets.HasSecrets(message) {
		findings := secrets.Scan(message)
		fmt.Fprintf(os.Stderr, "Error: your message may contain a secret (detected: %s).\n", findings[0].Type)
		fmt.Fprintln(os.Stderr, "What to do: redact the sensitive value and try again.")
		os.Exit(1)
	}

	topicMap := map[string]string{
		"gotcha":       "gotchas",
		"pattern":      "patterns",
		"decision":     "decisions",
		"architecture": "architecture",
		"memory":       "memory",
	}

	normalized, ok := topicMap[strings.ToLower(topic)]
	if !ok {
		fmt.Printf("Unknown topic '%s'. Available topics: gotcha, pattern, decision, architecture, memory\n", topic)
		fmt.Println("What to do: use one of the listed topic names.")
		os.Exit(1)
	}

	if err := brain.AddEntry(normalized, message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: make sure you are in a project with .brain/ initialized.\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added to %s\n", normalized)
}

func cmdEval() {
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

	diffStat := runGit(cwd, "diff", "--stat", "HEAD~1")
	nameStatus := runGit(cwd, "diff", "--name-status", "HEAD~1")
	shortstat := runGit(cwd, "diff", "--shortstat", "HEAD~1")
	log := runGit(cwd, "log", "--oneline", "-3")

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
	fmt.Println("Append your evaluation (objective, work completed, self-evaluation, lessons learned).")
}

func runGit(cwd string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	output, _ := cmd.CombinedOutput()
	return string(output)
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return "`" + strings.Join(items, "`, `") + "`"
}

func cmdPrune(dryRun bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	prunePath := filepath.Join(filepath.Dir(brainDir), ".brainprune")
	data, err := os.ReadFile(prunePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No .brainprune file found. Nothing to prune.")
			fmt.Println("What to do: create a .brainprune file with patterns to match for removal.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error reading .brainprune: %v\nWhat to do: check file permissions.\n", err)
		os.Exit(1)
	}

	var activePatterns []string
	for _, p := range strings.Split(string(data), "\n") {
		p = strings.TrimSpace(p)
		if p != "" && !strings.HasPrefix(p, "#") {
			activePatterns = append(activePatterns, p)
		}
	}

	if len(activePatterns) == 0 {
		fmt.Println("No prune patterns defined in .brainprune. Nothing to prune.")
		return
	}

	topicFiles := []string{"gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	archivedDir := filepath.Join(brainDir, "archived")
	os.MkdirAll(archivedDir, 0755)

	var pruned []string

	for _, tf := range topicFiles {
		path := filepath.Join(brainDir, tf)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		var kept, removed []string
		for scanner.Scan() {
			line := scanner.Text()
			matched := false
			for _, pattern := range activePatterns {
				if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
					matched = true
					break
				}
			}
			if matched {
				removed = append(removed, line)
			} else {
				kept = append(kept, line)
			}
		}

		if len(removed) > 0 {
			pruned = append(pruned, fmt.Sprintf("%s: %d entries", tf, len(removed)))
			if !dryRun {
				os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0600)
				archivePath := filepath.Join(archivedDir, fmt.Sprintf("%s-%s.md", tf[:len(tf)-3], time.Now().Format("2006-01-02")))
				os.WriteFile(archivePath, []byte(fmt.Sprintf("# Archived from %s — %s\n\n%s\n", tf, time.Now().Format("2006-01-02"), strings.Join(removed, "\n"))), 0600)
			}
		}
	}

	if len(pruned) == 0 {
		fmt.Println("No entries matched prune patterns. Nothing to prune.")
		return
	}

	if dryRun {
		fmt.Println("Dry run — would prune:")
		for _, p := range pruned {
			fmt.Printf("  %s\n", p)
		}
	} else {
		fmt.Println("Pruned:")
		for _, p := range pruned {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println("\nArchived entries saved to .brain/archived/")
	}
}

func cmdStatus(jsonFlag bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	topicFiles := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	var topicCount int
	var totalSize int64
	for _, f := range topicFiles {
		info, err := os.Stat(filepath.Join(brainDir, f))
		if err == nil {
			topicCount++
			totalSize += info.Size()
		}
	}

	sessionsDir := filepath.Join(brainDir, "sessions")
	sessionCount := 0
	if entries, err := os.ReadDir(sessionsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				sessionCount++
			}
		}
	}

	lineCount, _ := brain.MemoryLineCount()
	lineStatus := "OK"
	if lineCount > 200 {
		lineStatus = "OVER LIMIT"
	}

	queueDir := filepath.Join(brainDir, ".queue")
	pendingCount := 0
	doneCount := 0
	if entries, err := os.ReadDir(queueDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				pendingCount++
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(queueDir, "done")); err == nil {
		doneCount = len(entries)
	}

	if jsonFlag {
		status := map[string]interface{}{
			"memory_lines":  lineCount,
			"memory_status": lineStatus,
			"topic_files":   topicCount,
			"session_files": sessionCount,
			"total_size_kb": totalSize / 1024,
			"queue_pending": pendingCount,
			"queue_done":    doneCount,
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("Knowledge Hub Status")
		fmt.Println("====================")
		limitHint := "OK"
		if lineCount > 200 {
			limitHint = "OVER LIMIT — run 'brain prune' or move entries to topic files"
		}
		fmt.Printf("MEMORY.md:       %d lines (%s)\n", lineCount, limitHint)
		fmt.Printf("Topic files:     %d files\n", topicCount)
		fmt.Printf("Session files:   %d sessions\n", sessionCount)
		fmt.Printf("Total size:      %d KB\n", totalSize/1024)
		fmt.Printf("Queue depth:     %d pending, %d done\n", pendingCount, doneCount)
	}
}

func cmdDaemon() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain daemon <start|stop|status|run>")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "start":
		cmdDaemonStart()
	case "stop":
		cmdDaemonStop()
	case "status":
		cmdDaemonStatus()
	case "run":
		runDaemon()
	default:
		fmt.Printf("Unknown daemon command: %s\nWhat to do: use start, stop, status, or run.\n", os.Args[2])
		os.Exit(1)
	}
}

func cmdDaemonStart() {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "brain"
	}

	if err := service.Start(); err != nil {
		if err := service.Register(execPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error registering daemon: %v\n", err)
			os.Exit(1)
		}
		if err := service.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting daemon: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Daemon started. Polling queue every 5s.")
}

func cmdDaemonStop() {
	if err := service.Stop(); err != nil {
		fmt.Println("Daemon stop not supported on this OS.")
		return
	}

	fmt.Println("Daemon stopped.")
}

func cmdDaemonStatus() {
	running := service.IsRunning()

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Println("Daemon Status")
		fmt.Println("=============")
		fmt.Println("Status:          not running")
		fmt.Println("What to do: run 'brain init' in a project directory first.")
		return
	}

	queueDir := filepath.Join(brainDir, ".queue")
	pendingCount := 0
	doneCount := 0
	failedCount := 0

	if entries, e := os.ReadDir(queueDir); e == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				pendingCount++
			}
		}
	}
	if entries, e := os.ReadDir(filepath.Join(queueDir, "done")); e == nil {
		doneCount = len(entries)
	}
	if entries, e := os.ReadDir(filepath.Join(queueDir, "failed")); e == nil {
		failedCount = len(entries)
	}

	fmt.Println("Daemon Status")
	fmt.Println("=============")
	if running {
		fmt.Println("Status:          running")
	} else {
		fmt.Println("Status:          not running")
		fmt.Println("What to do: run 'brain daemon start' to start it.")
	}
	fmt.Printf("Queue:           %d pending, %d done, %d failed\n", pendingCount, doneCount, failedCount)

	doneDir := filepath.Join(queueDir, "done")
	if entries, e := os.ReadDir(doneDir); e == nil && len(entries) > 0 {
		fmt.Printf("Last processed:  %s\n", entries[len(entries)-1].Name())
	}
}

func lockFilePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	lockDir := filepath.Join(cacheDir, "brain")
	if err := os.MkdirAll(lockDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(lockDir, "brain-daemon.pid"), nil
}

func acquireLock() (*os.File, error) {
	path, err := lockFilePath()
	if err != nil {
		return nil, fmt.Errorf("cannot determine lock file path: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot open lock file: %w", err)
	}

	if err := tryLockFile(f); err != nil {
		content, _ := os.ReadFile(path)
		f.Close()
		return nil, fmt.Errorf("another daemon is already running (PID: %s).\nWhat to do: run 'brain daemon stop' first, or remove the lock file at %s", strings.TrimSpace(string(content)), path)
	}

	f.Truncate(0)
	f.Seek(0, 0)
	fmt.Fprintf(f, "%d\n", os.Getpid())
	f.Sync()

	return f, nil
}

func releaseLock(f *os.File) {
	if f == nil {
		return
	}
	unlockFile(f)
	f.Close()
	os.Remove(f.Name())
}

func runDaemon() {
	fmt.Println("brain-daemon starting...")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\nWhat to do: check ~/.config/brain/config.yaml\n", err)
		os.Exit(1)
	}

	pollInterval := daemon.ParsePollInterval(cfg.Daemon.PollInterval)

	apiKey := config.GetAPIKey()
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OpenRouter API key not configured.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain config set llm.api_key sk-or-v1-...'")
		os.Exit(1)
	}

	lockFile, err := acquireLock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer releaseLock(lockFile)

	fmt.Printf("Version:         %s\n", version)
	fmt.Printf("Poll interval:   %s\n", pollInterval)
	fmt.Printf("Model:           %s\n", cfg.LLM.Model)
	fmt.Println("Watching for queue items...")

	ctx, stop := setupSignalContext()
	defer stop()

	cycleCount := 0
	startupRecoveryDone := false

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down gracefully...")
			fmt.Println("Daemon stopped.")
			return
		default:
		}

		cycleCount++

		if cycleCount%100 == 0 {
			newCfg, err := config.Load()
			if err == nil {
				cfg = newCfg
				pollInterval = daemon.ParsePollInterval(cfg.Daemon.PollInterval)
			}
			apiKey = config.GetAPIKey()
		}

		brainDir, err := brain.FindBrainDir()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		if !startupRecoveryDone {
			daemon.RecoverStaleProcessing(brainDir)
			startupRecoveryDone = true
		}

		queueDir := filepath.Join(brainDir, ".queue")
		entries, err := os.ReadDir(queueDir)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		var pending []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "commit-") && strings.HasSuffix(e.Name(), ".json") {
				pending = append(pending, filepath.Join(queueDir, e.Name()))
			}
		}

		if len(pending) == 0 {
			time.Sleep(pollInterval)
			continue
		}

		maxPerCycle := 5
		if len(pending) < maxPerCycle {
			maxPerCycle = len(pending)
		}

		for i := 0; i < maxPerCycle; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("\nShutting down gracefully...")
				daemon.RecoverStaleProcessing("")
				fmt.Println("Daemon stopped.")
				return
			default:
			}

			itemPath := pending[i]
			processingPath := itemPath + ".processing"

			if err := os.Rename(itemPath, processingPath); err != nil {
				continue
			}

			fmt.Printf("Processing: %s\n", filepath.Base(processingPath))

			getDiff := func(repo string) (string, error) {
				out, err := exec.Command("git", "-C", repo, "diff", "HEAD~1").CombinedOutput()
				if err != nil {
					return "", err
				}
				return string(out), nil
			}

			analyzeFn := func(req analyzer.AnalyzeRequest) (analyzer.Finding, error) {
				return analyzer.Analyze(analyzer.AnalyzeRequest{
					Diff:       req.Diff,
					APIKey:     apiKey,
					Model:      cfg.LLM.Model,
					APIBaseURL: "",
				})
			}

			processed, err := daemon.ProcessItemWithDeps(
				processingPath, queueDir, brainDir,
				filepath.Dir(brainDir), cfg.Daemon.MaxRetries,
				getDiff, analyzeFn,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			if processed {
				fmt.Println("Findings written successfully.")
			}
		}
	}
}

func cmdVersion() {
	fmt.Printf("brain %s", version)
	if commit != "" {
		fmt.Printf("  commit: %s", commit)
	}
	if date != "" {
		fmt.Printf("  built: %s", date)
	}
	fmt.Println()
	fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func cmdConfig() {
	if len(os.Args) < 3 {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\nWhat to do: check ~/.config/brain/config.yaml\n", err)
			os.Exit(1)
		}

		fmt.Println("Current Configuration")
		fmt.Println("=====================")
		fmt.Printf("LLM Provider:    %s\n", cfg.LLM.Provider)
		if cfg.LLM.APIKey != "" {
			fmt.Printf("API Key:         %s\n", maskKey(cfg.LLM.APIKey))
		} else {
			fmt.Println("API Key:         not set")
		}
		fmt.Printf("Model:           %s\n", cfg.LLM.Model)
		fmt.Printf("Max Diff Lines:  %d\n", cfg.Analysis.MaxDiffLines)
		fmt.Printf("Categories:      %s\n", strings.Join(cfg.Analysis.Categories, ", "))
		fmt.Printf("Poll Interval:   %s\n", cfg.Daemon.PollInterval)
		fmt.Printf("Max Retries:     %d\n", cfg.Daemon.MaxRetries)
		fmt.Printf("Retry Backoff:   %s\n", cfg.Daemon.RetryBackoff)
		fmt.Printf("\nConfig file:     %s\n", config.ConfigPath())
		return
	}

	if os.Args[2] == "set" {
		if len(os.Args) < 5 {
			fmt.Println("Usage: brain config set <key> <value>")
			fmt.Println("Example: brain config set llm.api_key sk-or-v1-...")
			os.Exit(1)
		}

		key := os.Args[3]
		value := os.Args[4]

		if err := config.SetValue(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		displayValue := value
		if strings.Contains(key, "api_key") || strings.Contains(key, "apikey") {
			displayValue = maskKey(value)
		}
		fmt.Printf("Set %s = %s\n", key, displayValue)
		return
	}

	fmt.Println("Usage: brain config [set <key> <value>]")
}

func maskKey(key string) string {
	if len(key) <= 6 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-2:]
}



func cmdUpdate() {
	fmt.Printf("Current version: %s\n", version)

	fmt.Println("Checking for updates...")
	release, err := updater.FetchLatestRelease(updater.FetchOptions{
		APIBaseURL: "https://api.github.com",
		Owner:      "dominhduc",
		Repo:       "agent-brain",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\nWhat to do: check your internet connection or try again later.\n", err)
		os.Exit(1)
	}

	if !updater.IsNewerVersion(version, release.TagName) {
		fmt.Printf("Already up to date (%s).\n", version)
		return
	}

	fmt.Printf("New version available: %s → %s\n", version, release.TagName)

	downloadURL, err := updater.FindAssetForPlatform(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine binary path: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	resolvedPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		resolvedPath = execPath
	}

	fmt.Printf("Downloading from %s...\n", filepath.Base(downloadURL))
	if err := updater.DownloadAndReplace(downloadURL, resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\nWhat to do: download manually from https://github.com/dominhduc/agent-brain/releases/latest\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to %s successfully!\n", release.TagName)

	service.Stop()
	fmt.Println("Restart the daemon with: brain daemon start")
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
	"Run `brain eval` to write a self-evaluation to the current session file.\n" +
	"Include: what you did, what worked, what failed, confidence scores, knowledge persisted.\n\n" +
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


