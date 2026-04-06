package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/review"
	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/updater"
)

func cmdStatus(jsonFlag bool) {
	brainDir, err := brain.FindBrainDir()

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
		lineCount, _ = brain.MemoryLineCount()
		lineStatus = "OK"
		if lineCount > 200 {
			lineStatus = "OVER LIMIT"
		}
	}

	// Config
	cfg, cfgErr := config.Load()
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
		entries, err := review.LoadPendingEntries(pendingDir)
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
		warnings = append(warnings, "MEMORY.md over limit — run 'brain prune'")
	}
	if queueFailed > 0 {
		warnings = append(warnings, fmt.Sprintf("%d failed items — run 'brain daemon failed' to inspect", queueFailed))
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

	// Text output
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
		fmt.Printf("  .brain/      found\n")
		fmt.Printf("  Topics       %d files (%d KB)\n", topicCount, totalSize/1024)
		fmt.Printf("  Sessions     %d\n", sessionCount)
		fmt.Printf("  MEMORY.md    %d lines (%s)\n", lineCount, lineStatus)
	} else {
		fmt.Println("  .brain/      not found")
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
		fmt.Println("  Error loading config")
	}

	fmt.Println()
	fmt.Println("Daemon")
	if daemonRunning {
		fmt.Println("  Status       running")
	} else {
		fmt.Println("  Status       not running")
	}
	fmt.Printf("  Queue        %d pending, %d done, %d failed\n", queuePending, queueDone, queueFailed)
	if pendingEntries > 0 {
		fmt.Printf("  Review       %d pending entries\n", pendingEntries)
	}

	if len(warnings) > 0 {
		fmt.Println()
		fmt.Println("Health")
		for _, w := range warnings {
			fmt.Printf("  ✗ %s\n", w)
		}
		if hubFound && apiKeySet && lineStatus != "OVER LIMIT" && queueFailed == 0 {
			fmt.Println("  ✓ All clear")
		}
	}
}

func keyStatus(set bool) string {
	if set {
		return "configured"
	}
	return "not set"
}
