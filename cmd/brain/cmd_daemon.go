package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/analyzer"
	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/daemon"
	"github.com/dominhduc/agent-brain/internal/service"
)

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

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	workDir := filepath.Dir(brainDir)
	if err := service.Register(execPath, workDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error registering daemon: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Daemon registered. Polling queue every 5s.")
}

func cmdDaemonStop() {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	workDir := filepath.Dir(brainDir)
	if err := service.Stop(workDir); err != nil {
		fmt.Println("Daemon stop not supported on this OS.")
		return
	}

	fmt.Println("Daemon stopped.")
}

func cmdDaemonStatus() {
	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Println("Daemon Status")
		fmt.Println("=============")
		fmt.Println("Status:          not running")
		fmt.Println("What to do: run 'brain init' in a project directory first.")
		return
	}

	workDir := filepath.Dir(brainDir)
	running := service.IsRunning(workDir)

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
		fmt.Fprintln(os.Stderr, "Warning: OpenRouter API key not configured yet.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain config set llm.api_key <your-openrouter-key>'")
		fmt.Fprintln(os.Stderr, "Daemon will start processing once the key is set.")
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

		if cycleCount%10 == 0 {
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

		if apiKey == "" {
			time.Sleep(pollInterval)
			continue
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

		limit := maxPerCycle
		if len(pending) < limit {
			limit = len(pending)
		}

		for i := 0; i < limit; i++ {
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
					Diff:     req.Diff,
					APIKey:   apiKey,
					Model:    cfg.LLM.Model,
					Provider: cfg.LLM.Provider,
					BaseURL:  cfg.LLM.BaseURL,
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
