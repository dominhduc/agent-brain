package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
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
		fmt.Fprintf(os.Stderr, "Error: cannot determine current directory: %v\n", err)
		os.Exit(1)
	}

	if brain.BrainDirExists(cwd) {
		fmt.Println("Knowledge hub already exists in this project.")
		fmt.Println("To reinitialize, remove the .brain/ directory first.")
		return
	}

	if !brain.IsGitRepo(cwd) {
		fmt.Println("Error: This doesn't appear to be a git repository.")
		fmt.Println("brain needs git to track changes. Run 'git init' first.")
		os.Exit(1)
	}

	fmt.Println("Initializing knowledge hub...")

	// 1. Create .brain/ structure
	if err := brain.EnsureBrainDir(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .brain/ directory: %v\n", err)
		os.Exit(1)
	}

	// 2. Create skeleton topic files
	skeletons := map[string]string{
		"MEMORY.md": `# Project Memory Index

## Project
[Auto-populated by daemon analysis or first agent session]

## Stack
[Auto-populated by daemon analysis or first agent session]

## Commands
[Auto-populated by daemon analysis or first agent session]

## Key Patterns
[Auto-populated by daemon analysis]

## Active Gotchas
[Auto-populated by daemon analysis]

## Topic Files
- ` + "`gotchas.md`" + ` — Error patterns and fixes
- ` + "`patterns.md`" + ` — Discovered conventions
- ` + "`architecture.md`" + ` — Module structure and relationships
- ` + "`decisions.md`" + ` — Architecture decisions and rationale

## Last Updated
[Auto-updated]
`,
		"gotchas.md":       "# Gotchas\n<!-- Entries added by brain add gotcha or daemon analysis -->\n",
		"patterns.md":      "# Patterns\n<!-- Entries added by brain add pattern or daemon analysis -->\n",
		"architecture.md":  "# Architecture\n<!-- Entries added by brain add architecture or daemon analysis -->\n",
		"decisions.md":     "# Decisions\n<!-- Entries added by brain add decision or daemon analysis -->\n",
	}

	brainDir := filepath.Join(cwd, ".brain")
	for name, content := range skeletons {
		path := filepath.Join(brainDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", name, err)
			os.Exit(1)
		}
	}

	// 3. Create .gitkeep in sessions
	sessionsDir := filepath.Join(brainDir, "sessions")
	if err := os.WriteFile(filepath.Join(sessionsDir, ".gitkeep"), []byte{}, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .gitkeep: %v\n", err)
		os.Exit(1)
	}

	// 4. Copy AGENTS.md
	agentsPath := filepath.Join(cwd, "AGENTS.md")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		if err := os.WriteFile(agentsPath, []byte(agentsTemplate), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating AGENTS.md: %v\n", err)
			os.Exit(1)
		}
	}

	// 5. Copy wrapper files
	wrappers := map[string]string{
		"CLAUDE.md":      "See AGENTS.md for complete project instructions and knowledge base.\n",
		".cursorrules":   "See AGENTS.md for complete project instructions and knowledge base.\n",
		".windsurfrules": "See AGENTS.md for complete project instructions and knowledge base.\n",
	}
	for name, content := range wrappers {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", name, err)
				os.Exit(1)
			}
		}
	}

	// 6. Update .gitignore
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
			fmt.Fprintf(os.Stderr, "Error updating .gitignore: %v\n", err)
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

	// 7. Install git post-commit hook
	if err := installGitHook(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not install git hook: %v\n", err)
		fmt.Println("The daemon will still work, but commits won't be auto-queued.")
	}

	// 8. Check API key (no interactive prompt — user sets via config command)
	apiKey := config.GetAPIKey()
	if apiKey == "" {
		fmt.Println("\nOpenRouter API key not configured.")
		fmt.Println("Set it with: brain config set llm.api_key sk-or-v1-...")
		fmt.Println("Or set the BRAIN_API_KEY environment variable.")
		fmt.Println("The daemon will start once the key is configured.")
	}

	// 9. Ensure config file exists
	if _, err := os.Stat(config.ConfigPath()); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		if apiKey := config.GetAPIKey(); apiKey != "" {
			cfg.LLM.APIKey = apiKey
		}
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not create config file: %v\n", err)
		}
	}

	// 10. Register and start daemon
	fmt.Println("\nRegistering daemon service...")
	if err := registerDaemonService(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not register daemon: %v\n", err)
		fmt.Println("You can start it manually with: brain daemon start")
	} else {
		fmt.Println("Daemon registered and started.")
	}

	fmt.Println("\nKnowledge hub initialized successfully!")
	fmt.Println("First commit will auto-populate knowledge via the daemon.")
	fmt.Println("Run 'brain status' to see hub statistics.")
}

func installGitHook(cwd string) error {
	hookContent := `#!/bin/bash
# Post-commit hook for agent-brain
# Installed by: brain init
# Purpose: Capture commit changes and queue for daemon analysis

BRAIN_DIR=".brain"
QUEUE_DIR="$BRAIN_DIR/.queue"

if [ ! -d "$QUEUE_DIR" ]; then
    exit 0
fi

TIMESTAMP=$(date +%Y%m%dT%H%M%S)
DIFF_STAT=$(git diff --stat HEAD~1 2>/dev/null || echo "No previous commit")
FILES=$(git diff --name-status HEAD~1 2>/dev/null || echo "No previous commit")
REPO=$(pwd)

# Simple JSON escaping
escape_json() {
    echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | tr '\n' ' '
}

DIFF_ESCAPED=$(escape_json "$DIFF_STAT")
FILES_ESCAPED=$(escape_json "$FILES")

cat > "$QUEUE_DIR/commit-${TIMESTAMP}.json" << EOF
{
  "timestamp": "${TIMESTAMP}",
  "repo": "${REPO}",
  "diff_stat": "${DIFF_ESCAPED}",
  "files": "${FILES_ESCAPED}",
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
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return err
	}

	return nil
}

func registerDaemonService() error {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "brain"
	}

	switch runtime.GOOS {
	case "darwin":
		return registerLaunchd(execPath)
	case "linux":
		return registerSystemd(execPath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func registerLaunchd(execPath string) error {
	home, _ := os.UserHomeDir()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(plistDir, 0755); err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dominhduc.brain-daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/brain-daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/brain-daemon.err</string>
</dict>
</plist>`, execPath)

	plistPath := filepath.Join(plistDir, "com.dominhduc.brain-daemon.plist")
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	// Load the service
	cmd := exec.Command("launchctl", "load", plistPath)
	return cmd.Run()
}

func registerSystemd(execPath string) error {
	home, _ := os.UserHomeDir()
	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return err
	}

	service := fmt.Sprintf(`[Unit]
Description=agent-brain Daemon
After=network.target

[Service]
Type=simple
ExecStart=%s daemon run
Restart=always
RestartSec=5
Environment=BRAIN_API_KEY=

[Install]
WantedBy=default.target`, execPath)

	servicePath := filepath.Join(serviceDir, "brain-daemon.service")
	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		return err
	}

	// Enable and start the service
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	cmd.Run()
	cmd = exec.Command("systemctl", "--user", "enable", "brain-daemon.service")
	cmd.Run()
	cmd = exec.Command("systemctl", "--user", "start", "brain-daemon.service")
	return cmd.Run()
}

func cmdGet(jsonFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain get <topic>")
		fmt.Println("Topics: memory, gotchas, patterns, decisions, architecture, all")
		os.Exit(1)
	}

	topic := os.Args[2]

	if topic == "all" {
		content, err := brain.GetAllTopics()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if jsonFlag {
			topics := map[string]string{
				"memory":       "",
				"gotchas":      "",
				"patterns":     "",
				"decisions":    "",
				"architecture": "",
			}
			for t := range topics {
				c, _ := brain.GetTopic(t)
				topics[t] = c
			}
			data, _ := json.MarshalIndent(topics, "", "  ")
			fmt.Println(string(data))
		} else {
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
		os.Exit(1)
	}

	query := os.Args[2]

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	files := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(query))

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

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if pattern.MatchString(line) {
				matches = append(matches, Match{
					File:    f,
					Line:    i + 1,
					Content: strings.TrimSpace(line),
				})
			}
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
		fmt.Println(`Usage: brain add <topic> "<message>"`)
		fmt.Println("Topics: gotcha, pattern, decision, architecture, memory")
		os.Exit(1)
	}

	topic := os.Args[2]
	message := strings.Join(os.Args[3:], " ")

	// Normalize topic names
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
		os.Exit(1)
	}

	if err := brain.AddEntry(normalized, message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added to %s\n", normalized)
}

func cmdEval() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !brain.IsGitRepo(cwd) {
		fmt.Println("Error: This doesn't appear to be a git repository.")
		os.Exit(1)
	}

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get git info
	diffStat := runGit(cwd, "diff", "--stat", "HEAD~1")
	nameStatus := runGit(cwd, "diff", "--name-status", "HEAD~1")
	shortstat := runGit(cwd, "diff", "--shortstat", "HEAD~1")
	log := runGit(cwd, "log", "--oneline", "-3")

	if strings.TrimSpace(diffStat) == "" {
		fmt.Println("No file changes detected since last session.")
		return
	}

	// Parse changes
	var created, modified, deleted []string
	lines := strings.Split(nameStatus, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		status := parts[0]
		file := parts[1]
		switch status {
		case "A":
			created = append(created, file)
		case "M":
			modified = append(modified, file)
		case "D":
			deleted = append(deleted, file)
		}
	}

	// Create session file
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	sessionsDir := filepath.Join(brainDir, "sessions")
	sessionPath := filepath.Join(sessionsDir, timestamp+".md")

	content := fmt.Sprintf(`# Session — %s

## Git Summary
- Files created: %s
- Files modified: %s
- Files deleted: %s
- Total changes: %s

## Recent Commits
%s

---
<!-- Agent: append your evaluation below this line -->
`,
		time.Now().Format("2006-01-02 15:04:05"),
		formatList(created),
		formatList(modified),
		formatList(deleted),
		strings.TrimSpace(shortstat),
		strings.TrimSpace(log),
	)

	if err := os.WriteFile(sessionPath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session file: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	prunePath := filepath.Join(filepath.Dir(brainDir), ".brainprune")
	data, err := os.ReadFile(prunePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No .brainprune file found. Nothing to prune.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error reading .brainprune: %v\n", err)
		os.Exit(1)
	}

	patterns := strings.Split(string(data), "\n")
	var activePatterns []string
	for _, p := range patterns {
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
	if err := os.MkdirAll(archivedDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating archived directory: %v\n", err)
		os.Exit(1)
	}

	var pruned []string

	for _, tf := range topicFiles {
		path := filepath.Join(brainDir, tf)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		var kept, removed []string

		for _, line := range lines {
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
				// Write kept lines back
				if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", tf, err)
					continue
				}

				// Write removed lines to archived
				archivePath := filepath.Join(archivedDir, fmt.Sprintf("%s-%s.md", tf[:len(tf)-3], time.Now().Format("2006-01-02")))
				archiveContent := fmt.Sprintf("# Archived from %s — %s\n\n%s\n", tf, time.Now().Format("2006-01-02"), strings.Join(removed, "\n"))
				os.WriteFile(archivePath, []byte(archiveContent), 0644)
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
		fmt.Printf("\nArchived entries saved to .brain/archived/\n")
	}
}

func cmdStatus(jsonFlag bool) {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Count files
	topicFiles := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	var topicCount int
	for _, f := range topicFiles {
		if _, err := os.Stat(filepath.Join(brainDir, f)); err == nil {
			topicCount++
		}
	}

	// Count sessions
	sessionsDir := filepath.Join(brainDir, "sessions")
	sessionCount := 0
	if entries, err := os.ReadDir(sessionsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				sessionCount++
			}
		}
	}

	// MEMORY.md line count
	lineCount, _ := brain.MemoryLineCount()
	lineStatus := "OK"
	if lineCount > 200 {
		lineStatus = "OVER LIMIT"
	}

	// Total size
	var totalSize int64
	filepath.Walk(brainDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Queue depth
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

	// Last updated
	var lastUpdated string
	entries, _ := os.ReadDir(brainDir)
	for _, e := range entries {
		if !e.IsDir() {
			info, _ := e.Info()
			if info != nil {
				lastUpdated = info.ModTime().Format("2006-01-02 15:04:05")
			}
		}
	}

	if jsonFlag {
		status := map[string]interface{}{
			"memory_lines":   lineCount,
			"memory_status":  lineStatus,
			"topic_files":    topicCount,
			"session_files":  sessionCount,
			"total_size_kb":  totalSize / 1024,
			"last_updated":   lastUpdated,
			"queue_pending":  pendingCount,
			"queue_done":     doneCount,
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("Knowledge Hub Status")
		fmt.Println("====================")
		fmt.Printf("MEMORY.md:       %d lines (%s, under 200)\n", lineCount, lineStatus)
		fmt.Printf("Topic files:     %d files\n", topicCount)
		fmt.Printf("Session files:   %d sessions\n", sessionCount)
		fmt.Printf("Total size:      %d KB\n", totalSize/1024)
		fmt.Printf("Last updated:    %s\n", lastUpdated)
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
		// Internal: run daemon in foreground (used by service managers)
		runDaemon()
	default:
		fmt.Printf("Unknown daemon command: %s\n", os.Args[2])
		fmt.Println("Usage: brain daemon <start|stop|status|run>")
		os.Exit(1)
	}
}

func cmdDaemonStart() {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "brain"
	}

	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.dominhduc.brain-daemon.plist")
		if _, err := os.Stat(plistPath); os.IsNotExist(err) {
			if err := registerLaunchd(execPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error registering daemon: %v\n", err)
				os.Exit(1)
			}
		}
		cmd := exec.Command("launchctl", "load", plistPath)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting daemon: %v\n", err)
			os.Exit(1)
		}
	case "linux":
		cmd := exec.Command("systemctl", "--user", "start", "brain-daemon.service")
		if err := cmd.Run(); err != nil {
			// Try registering first
			if err := registerSystemd(execPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error registering daemon: %v\n", err)
				os.Exit(1)
			}
			cmd = exec.Command("systemctl", "--user", "start", "brain-daemon.service")
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting daemon: %v\n", err)
				os.Exit(1)
			}
		}
	default:
		fmt.Println("Daemon auto-start not supported on this OS.")
		fmt.Println("Run 'brain daemon run' to start in foreground.")
		return
	}

	fmt.Println("Daemon started. Polling queue every 5s.")
}

func cmdDaemonStop() {
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.dominhduc.brain-daemon.plist")
		cmd := exec.Command("launchctl", "unload", plistPath)
		cmd.Run()
	case "linux":
		cmd := exec.Command("systemctl", "--user", "stop", "brain-daemon.service")
		cmd.Run()
	default:
		fmt.Println("Daemon stop not supported on this OS.")
		return
	}

	fmt.Println("Daemon stopped.")
}

func cmdDaemonStatus() {
	running := false
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("launchctl", "list", "com.dominhduc.brain-daemon")
		if cmd.Run() == nil {
			running = true
		}
	case "linux":
		cmd := exec.Command("systemctl", "--user", "is-active", "brain-daemon.service")
		if cmd.Run() == nil {
			running = true
		}
	}

	brainDir, err := brain.FindBrainDir()
	queueDir := filepath.Join(brainDir, ".queue")
	pendingCount := 0
	doneCount := 0
	failedCount := 0

	if err == nil {
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
		if entries, err := os.ReadDir(filepath.Join(queueDir, "failed")); err == nil {
			failedCount = len(entries)
		}
	}

	fmt.Println("Daemon Status")
	fmt.Println("=============")
	if running {
		fmt.Println("Status:          running")
	} else {
		fmt.Println("Status:          not running")
	}
	fmt.Printf("Queue:           %d pending, %d done, %d failed\n", pendingCount, doneCount, failedCount)

	if err == nil {
		// Find last processed item
		doneDir := filepath.Join(queueDir, "done")
		if entries, err := os.ReadDir(doneDir); err == nil && len(entries) > 0 {
			last := entries[len(entries)-1]
			fmt.Printf("Last processed:  %s\n", last.Name())
		}
	}
}

func runDaemon() {
	fmt.Println("brain-daemon starting...")

	pollInterval := config.PollInterval()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load config: %v\n", err)
	}

	apiKey := config.GetAPIKey()
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OpenRouter API key not configured.")
		fmt.Fprintln(os.Stderr, "Run: brain config set llm.api_key sk-or-v1-...")
		os.Exit(1)
	}

	fmt.Printf("Polling interval: %s\n", pollInterval)
	fmt.Printf("Model: %s\n", cfg.LLM.Model)
	fmt.Println("Watching for queue items...")

	for {
		brainDir, err := brain.FindBrainDir()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		queueDir := filepath.Join(brainDir, ".queue")
		entries, err := os.ReadDir(queueDir)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Find pending items (sorted by name = sorted by timestamp)
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

		// Process first pending item
		itemPath := pending[0]
		fmt.Printf("Processing: %s\n", filepath.Base(itemPath))

		// Read queue item
		data, err := os.ReadFile(itemPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading queue item: %v\n", err)
			moveToFailed(itemPath, queueDir)
			continue
		}

		var item struct {
			Timestamp string `json:"timestamp"`
			Repo      string `json:"repo"`
			DiffStat  string `json:"diff_stat"`
			Files     string `json:"files"`
			Attempts  int    `json:"attempts"`
			Status    string `json:"status"`
		}
		if err := json.Unmarshal(data, &item); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing queue item: %v\n", err)
			moveToFailed(itemPath, queueDir)
			continue
		}

		// Get full diff
		cmd := exec.Command("git", "-C", item.Repo, "diff", "HEAD~1")
		diffOutput, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting diff: %v\n", err)
			item.Attempts++
			if item.Attempts >= cfg.Daemon.MaxRetries {
				moveToFailed(itemPath, queueDir)
			} else {
				// Update attempts
				itemData, _ := json.Marshal(item)
				os.WriteFile(itemPath, itemData, 0644)
			}
			continue
		}

		diff := string(diffOutput)
		if len(diff) > cfg.Analysis.MaxDiffLines*100 {
			diff = diff[:cfg.Analysis.MaxDiffLines*100] + "\n... [diff truncated]"
		}

		// Call LLM
		findings, err := callOpenRouter(diff, cfg, apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling LLM: %v\n", err)
			item.Attempts++
			if item.Attempts >= cfg.Daemon.MaxRetries {
				moveToFailed(itemPath, queueDir)
			} else {
				itemData, _ := json.Marshal(item)
				os.WriteFile(itemPath, itemData, 0644)
			}
			continue
		}

		// Write findings
		if err := writeFindings(findings, brainDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing findings: %v\n", err)
		} else {
			fmt.Println("Findings written successfully.")
		}

		// Mark done
		moveToDone(itemPath, queueDir)
	}
}

func moveToFailed(itemPath, queueDir string) {
	doneDir := filepath.Join(queueDir, "failed")
	os.MkdirAll(doneDir, 0755)
	os.Rename(itemPath, filepath.Join(doneDir, filepath.Base(itemPath)))
}

func moveToDone(itemPath, queueDir string) {
	doneDir := filepath.Join(queueDir, "done")
	os.MkdirAll(doneDir, 0755)
	os.Rename(itemPath, filepath.Join(doneDir, filepath.Base(itemPath)))
}

func cmdConfig() {
	if len(os.Args) < 3 {
		// Show current config
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Current Configuration")
		fmt.Println("=====================")
		fmt.Printf("LLM Provider:    %s\n", cfg.LLM.Provider)
		if cfg.LLM.APIKey != "" {
			masked := cfg.LLM.APIKey[:8] + "..." + cfg.LLM.APIKey[len(cfg.LLM.APIKey)-4:]
			fmt.Printf("API Key:         %s\n", masked)
		} else {
			fmt.Printf("API Key:         not set\n")
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

		fmt.Printf("Set %s = %s\n", key, value)
		return
	}

	fmt.Println("Usage: brain config [set <key> <value>]")
}

// LLM integration types
type LLMFinding struct {
	Gotchas      []string `json:"gotchas"`
	Patterns     []string `json:"patterns"`
	Decisions    []string `json:"decisions"`
	Architecture []string `json:"architecture"`
	Confidence   string   `json:"confidence"`
}

func callOpenRouter(diff string, cfg config.Config, apiKey string) (LLMFinding, error) {
	var findings LLMFinding

	prompt := fmt.Sprintf(`You are analyzing a git commit to extract knowledge for a coding agent's memory system.

The agent works on this codebase over time. Your job is to identify patterns, gotchas,
decisions, and architectural insights from the code changes.

## Rules
- Only extract knowledge that is NOT obvious from reading the code
- Focus on things that would save time or prevent mistakes in future sessions
- Be specific: mention file paths, function names, exact patterns
- If nothing noteworthy was found, return empty arrays
- Do NOT hallucinate — only report what the diff actually shows
- Output ONLY valid JSON, no markdown formatting, no explanation

## Categories
- **gotchas**: Things that could trip up the agent (error patterns, edge cases, quirks)
- **patterns**: Conventions the code follows (naming, structure, tool choices)
- **decisions**: Why certain choices were made (trade-offs, rejected alternatives visible in diff)
- **architecture**: Module relationships, key abstractions, data flow

## Input
Full diff:
%s

## Output Format (JSON only)
{
  "gotchas": ["Finding 1", "Finding 2"],
  "patterns": ["Finding 1"],
  "decisions": ["Finding 1"],
  "architecture": [],
  "confidence": "HIGH|MEDIUM|LOW"
}`, diff)

	// Build request
	reqBody := map[string]interface{}{
		"model": cfg.LLM.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	reqData, _ := json.Marshal(reqBody)

	req, err := exec.Command("curl", "-s",
		"-X", "POST",
		"https://openrouter.ai/api/v1/chat/completions",
		"-H", "Authorization: Bearer "+apiKey,
		"-H", "HTTP-Referer: https://github.com/dominhduc/agent-brain",
		"-H", "X-Title: agent-brain",
		"-H", "Content-Type: application/json",
		"-d", string(reqData),
	).CombinedOutput()

	if err != nil {
		return findings, fmt.Errorf("curl failed: %w", err)
	}

	// Parse response
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(req, &resp); err != nil {
		return findings, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Error.Message != "" {
		return findings, fmt.Errorf("API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return findings, fmt.Errorf("no choices in response")
	}

	content := resp.Choices[0].Message.Content

	// Extract JSON from content (in case model wraps in markdown code blocks)
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		content = content[jsonStart : jsonEnd+1]
	}

	if err := json.Unmarshal([]byte(content), &findings); err != nil {
		return findings, fmt.Errorf("failed to parse findings JSON: %w\nContent: %s", err, content)
	}

	return findings, nil
}

func writeFindings(findings LLMFinding, brainDir string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	writeEntries := func(filename string, entries []string) error {
		if len(entries) == 0 {
			return nil
		}
		path := filepath.Join(brainDir, filename)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		for _, entry := range entries {
			entryText := fmt.Sprintf("\n### [%s] %s\n\n", timestamp, entry)
			if _, err := f.WriteString(entryText); err != nil {
				return err
			}
		}
		return nil
	}

	if err := writeEntries("gotchas.md", findings.Gotchas); err != nil {
		return err
	}
	if err := writeEntries("patterns.md", findings.Patterns); err != nil {
		return err
	}
	if err := writeEntries("decisions.md", findings.Decisions); err != nil {
		return err
	}
	if err := writeEntries("architecture.md", findings.Architecture); err != nil {
		return err
	}

	return nil
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
