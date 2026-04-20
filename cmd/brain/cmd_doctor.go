package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/updater"
	"golang.org/x/term"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func colorize(condition bool, color, text string) string {
	if condition {
		return color + text + colorReset
	}
	return text
}

func cmdDoctor(jsonFlag, fixFlag bool) {
	conflictsFlag := hasFlag("--conflicts")

	brainDir, err := knowledge.FindBrainDir()

	// Hub stats
	var hubFound bool
	var topicCount, sessionCount, lineCount int
	var totalSize int64
	var lineStatus string

	if err == nil {
		hubFound = true
		topicFiles := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
		for _, f := range topicFiles {
			info, err := os.Stat(filepath.Join(brainDir, f))
			if err == nil {
				topicCount++
				totalSize += info.Size()
			}
		}
		sessionsDir := filepath.Join(brainDir, "sessions")
		if entries, e := os.ReadDir(sessionsDir); e == nil {
			for _, e := range entries {
				if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					sessionCount++
				}
			}
		}
		lineCount, _ = knowledge.MemoryLineCount()
		lineStatus = "OK"
		if lineCount > 200 {
			lineStatus = "OVER LIMIT"
		}
	}

	// Config
	var cfg config.Config
	var cfgErr error
	if brainDir != "" && config.ProjectConfigExists(brainDir) {
		cfg, cfgErr = config.LoadForProject(brainDir)
	} else {
		cfg, cfgErr = config.Load()
	}
	provider := "unknown"
	model := ""
	apiKeySet := false
	profile := ""
	if cfgErr == nil {
		provider = cfg.LLM.Provider
		model = cfg.LLM.Model
		apiKeySet = cfg.LLM.APIKey != ""
		profile = cfg.Review.Profile
		if cp, ok := config.GetCustomProvider(cfg.LLM.Provider); ok {
			model = cp.Model
			apiKeySet = cp.APIKey != ""
		}
	}

	// Daemon
	var daemonRunning bool
	var queuePending, queueDone, queueFailed int
	if brainDir != "" {
		workDir := filepath.Dir(brainDir)
		daemonRunning = service.IsRunning(workDir)
		queueDir := filepath.Join(brainDir, ".queue")
		if entries, e := os.ReadDir(queueDir); e == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasPrefix(e.Name(), "commit-") && strings.HasSuffix(e.Name(), ".json") {
					queuePending++
				}
			}
		}
		if entries, e := os.ReadDir(filepath.Join(queueDir, "done")); e == nil {
			queueDone = len(entries)
		}
		if entries, e := os.ReadDir(filepath.Join(queueDir, "failed")); e == nil {
			queueFailed = len(entries)
		}
	}

	// Pending entries (for review)
	var pendingEntries int
	if brainDir != "" {
		pendingDir := filepath.Join(brainDir, "pending")
		entries, err := knowledge.LoadPendingEntries(pendingDir)
		if err == nil {
			pendingEntries = len(entries)
		}
	}

	// Warnings
	var warnings []string
	if !hubFound {
		warnings = append(warnings, "No .brain/ directory — run 'brain init'")
	}
	if cfgErr != nil {
		warnings = append(warnings, fmt.Sprintf("Config error: %v", cfgErr))
	}
	if !apiKeySet && cfgErr == nil {
		warnings = append(warnings, "API key not set — run 'brain config set api-key <key>'")
	}
	if lineStatus == "OVER LIMIT" {
		warnings = append(warnings, "MEMORY.md over limit — run 'brain clean'")
	}
	if queueFailed > 0 {
		warnings = append(warnings, fmt.Sprintf("%d failed items — run 'brain daemon failed' to inspect", queueFailed))
	}
	if brainDir != "" {
		memPath := filepath.Join(brainDir, "MEMORY.md")
		if info, err := os.Stat(memPath); err == nil {
			daysSinceUpdate := time.Since(info.ModTime()).Hours() / 24
			if daysSinceUpdate > 7 {
				warnings = append(warnings, fmt.Sprintf("MEMORY.md not updated in %.0f days — run 'brain get all --summary' to check index", daysSinceUpdate))
			}
		}
	}
	if pendingEntries > 0 {
		warnings = append(warnings, fmt.Sprintf("%d pending entries awaiting review — run 'brain daemon review'", pendingEntries))
	}

	if fixFlag {
		fmt.Println("Running auto-repair...")
		if brainDir != "" {
			idx, err := knowledge.RebuildIndex(brainDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error rebuilding index: %v\n", err)
			} else {
				if err := idx.Save(brainDir); err == nil {
					fmt.Printf("  Index rebuilt: %d entries\n", len(idx.Entries))
				}
			}
		}
		if brainDir != "" && queueFailed > 0 {
			failedDir := filepath.Join(brainDir, ".queue", "failed")
			queueDir := filepath.Join(brainDir, ".queue")
			entries, _ := os.ReadDir(failedDir)
			retried := 0
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				srcPath := filepath.Join(failedDir, e.Name())
				name := strings.TrimSuffix(e.Name(), ".processing")
				data, err := os.ReadFile(srcPath)
				if err != nil {
					continue
				}
				var item struct {
					Attempts    int    `json:"attempts"`
					ErrorReason string `json:"error_reason"`
				}
				if err := json.Unmarshal(data, &item); err != nil {
					continue
				}
				item.Attempts = 0
				item.ErrorReason = ""
				newData, _ := json.Marshal(item)
				destPath := filepath.Join(queueDir, name)
				if os.WriteFile(destPath, newData, 0600) == nil {
					os.Remove(srcPath)
					retried++
				}
			}
			if retried > 0 {
				fmt.Printf("  Requeued %d failed items\n", retried)
			}
		}
		fmt.Println("Auto-repair complete.")
		return
	}

	if jsonFlag {
		status := map[string]interface{}{
			"version":         version,
			"os":              fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			"hub_found":       hubFound,
			"topic_files":     topicCount,
			"session_files":   sessionCount,
			"total_size_kb":   totalSize / 1024,
			"memory_lines":    lineCount,
			"memory_status":   lineStatus,
			"provider":        provider,
			"model":           model,
			"api_key_set":     apiKeySet,
			"profile":         profile,
			"daemon_running":  daemonRunning,
			"queue_pending":   queuePending,
			"queue_done":      queueDone,
			"queue_failed":    queueFailed,
			"pending_entries": pendingEntries,
			"warnings":        warnings,
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
		return
	}

	useColor := isTTY()

	fmt.Printf("brain %s  %s/%s", version, runtime.GOOS, runtime.GOARCH)
	if commit != "" {
		short := commit
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Printf("  %s", short)
	}
	fmt.Println()
	if latest, err := updater.CheckLatest(version); err == nil && latest != "" {
		if latest != version {
			fmt.Printf("  (update available: %s)\n", latest)
		}
	}
	fmt.Println()

	fmt.Println("Hub")
	if hubFound {
		fmt.Printf("  .brain/      %s\n", colorize(useColor, colorGreen, "found ✓"))
		fmt.Printf("  Topics       %d files (%d KB)\n", topicCount, totalSize/1024)
		fmt.Printf("  Sessions     %d\n", sessionCount)
		lineColor := colorGreen
		if lineStatus == "OVER LIMIT" {
			lineColor = colorRed
		}
		fmt.Printf("  MEMORY.md    %d lines (%s)\n", lineCount, colorize(useColor, lineColor, lineStatus))
	} else {
		fmt.Printf("  .brain/      %s\n", colorize(useColor, colorRed, "not found"))
	}

	fmt.Println()
	fmt.Println("Config")
	if cfgErr == nil {
		if cp, ok := config.GetCustomProvider(cfg.LLM.Provider); ok {
			fmt.Printf("  Provider     %s (custom)\n", provider)
			fmt.Printf("  Model        %s\n", cp.Model)
			fmt.Printf("  API Key      %s\n", keyStatus(apiKeySet))
		} else {
			fmt.Printf("  Provider     %s\n", provider)
			fmt.Printf("  Model        %s\n", model)
			fmt.Printf("  API Key      %s\n", keyStatus(apiKeySet))
		}
		fmt.Printf("  Profile      %s\n", profile)
	} else {
		fmt.Printf("  %s\n", colorize(useColor, colorRed, "Error loading config"))
	}

	fmt.Println()
	fmt.Println("Daemon")
	daemonColor := colorGreen
	if !daemonRunning {
		daemonColor = colorYellow
	}
	fmt.Printf("  Status       %s\n", colorize(useColor, daemonColor, func() string {
		if daemonRunning {
			return "running ●"
		}
		return "not running"
	}()))
	if queueFailed > 0 {
		fmt.Printf("  Queue        %d pending, %d done, %d %s\n", queuePending, queueDone, queueFailed, colorize(useColor, colorRed, "failed"))
	} else {
		fmt.Printf("  Queue        %d pending, %d done, %d failed\n", queuePending, queueDone, queueFailed)
	}
	if pendingEntries > 0 {
		fmt.Printf("  Review       %d pending entries\n", pendingEntries)
	}

	fmt.Println()

	if len(warnings) > 0 {
		fmt.Println("Warnings")
		for _, w := range warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
		fmt.Println()
	}

	if conflictsFlag {
		hub, hubErr := knowledge.Open(brainDir)
		if hubErr == nil {
			conflicts, err := hub.FindConflicts()
			if err != nil {
				fmt.Printf("Error finding conflicts: %v\n", err)
			} else if len(conflicts) > 0 {
				fmt.Printf("Potential Conflicts (%d)\n", len(conflicts))
				for _, c := range conflicts {
					fmt.Printf("  ⚠ %s vs %s\n", c.Key1, c.Key2)
				}
				fmt.Println()
			} else {
				fmt.Println("No conflicts detected.")
				fmt.Println()
			}
		}
	}

	// Health checks
	checks := []func() (string, bool, string){
		checkDoctorVersion,
		checkDoctorGit,
		checkDoctorBrainDir,
		checkDoctorConfig,
		checkDoctorDaemon,
		checkDoctorPreflight,
		checkDoctorGitignore,
		checkDoctorTrackedBrain,
	}

	fmt.Println("Health")
	allPassed := true
	for _, check := range checks {
		name, passed, detail := check()
		status := "✓"
		if !passed {
			status = "✗"
			allPassed = false
		}
		fmt.Printf("  %s %s", status, name)
		if detail != "" {
			fmt.Printf(" — %s", detail)
		}
		fmt.Println()
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed!")
	} else {
		fmt.Println("Some checks failed. See details above.")
		os.Exit(1)
	}
}

func keyStatus(set bool) string {
	if set {
		return "configured"
	}
	return "not set"
}

func checkDoctorVersion() (string, bool, string) {
	return fmt.Sprintf("Version: %s", version), true, fmt.Sprintf("OS: %s/%s", runtime.GOOS, runtime.GOARCH)
}

func checkDoctorGit() (string, bool, string) {
	_, err := exec.LookPath("git")
	if err != nil {
		return "Git", false, "not found in PATH"
	}
	out, _ := exec.Command("git", "--version").Output()
	return "Git", true, strings.TrimSpace(string(out))
}

func checkDoctorBrainDir() (string, bool, string) {
	cwd, _ := os.Getwd()
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		return "Project .brain/", false, fmt.Sprintf("not found (cwd: %s)", cwd)
	}
	entries, _ := os.ReadDir(brainDir)
	return "Project .brain/", true, fmt.Sprintf("found at %s (%d entries)", brainDir, len(entries))
}

func checkDoctorConfig() (string, bool, string) {
	brainDir, _ := knowledge.FindBrainDir()
	var cfg config.Config
	var err error
	if brainDir != "" && config.ProjectConfigExists(brainDir) {
		cfg, err = config.LoadForProject(brainDir)
	} else {
		cfg, err = config.Load()
	}
	if err != nil {
		return "Config", false, fmt.Sprintf("error: %v", err)
	}
	provider := cfg.LLM.Provider
	model := cfg.LLM.Model
	if cfg.LLM.APIKey == "" {
		return "Config", false, fmt.Sprintf("provider: %s, model: %s, API key: not set", provider, model)
	}
	return "Config", true, fmt.Sprintf("provider: %s, model: %s", provider, model)
}

func checkDoctorPreflight() (string, bool, string) {
	_, err := exec.LookPath("git")
	if err != nil {
		return "Preflight", false, "git required but not found"
	}
	return "Preflight", true, "all dependencies available"
}

func checkDoctorDaemon() (string, bool, string) {
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		return "Daemon", false, "no project .brain/ found"
	}
	workDir := filepath.Dir(brainDir)
	running := service.IsRunning(workDir)
	if running {
		return "Daemon", true, "running"
	}
	return "Daemon", false, "not running (run 'brain daemon start')"
}

func checkDoctorGitignore() (string, bool, string) {
	cwd, _ := os.Getwd()
	gitignorePath := filepath.Join(cwd, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return ".gitignore", true, "not found (optional)"
	}
	content := string(data)
	if strings.Contains(content, ".brain/") {
		return ".gitignore", true, ".brain/ is gitignored"
	}
	oldEntries := []string{
		".brain/archived/",
		".brain/.queue/",
		".brain/sessions/",
	}
	for _, e := range oldEntries {
		if strings.Contains(content, e) {
			return ".gitignore", false, "uses legacy selective entries — run 'brain init' to update"
		}
	}
	return ".gitignore", false, ".brain/ not in .gitignore — run 'brain init' to add"
}

func checkDoctorTrackedBrain() (string, bool, string) {
	cmd := exec.Command("git", "ls-files", ".brain/")
	out, err := cmd.Output()
	if err != nil {
		return "Git tracking", true, "cannot check (not a git repo or git unavailable)"
	}
	files := strings.TrimSpace(string(out))
	if files == "" {
		return "Git tracking", true, ".brain/ not tracked in git"
	}
	count := len(strings.Split(files, "\n"))
	return "Git tracking", false, fmt.Sprintf(".brain/ has %d tracked file(s) — run 'git rm -r --cached .brain/' to untrack", count)
}
