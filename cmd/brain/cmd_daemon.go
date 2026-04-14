package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/daemon"
	"github.com/dominhduc/agent-brain/internal/otel"
	"github.com/dominhduc/agent-brain/internal/profile"
	"github.com/dominhduc/agent-brain/internal/service"
	"github.com/dominhduc/agent-brain/internal/tui"
	"go.opentelemetry.io/otel/trace"
)

func cmdDaemon() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain daemon <start|stop|restart|status|failed|retry|run|review>")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "start":
		cmdDaemonStart()
	case "stop":
		cmdDaemonStop()
	case "restart":
		cmdDaemonRestart()
	case "status":
		cmdDaemonStatus()
	case "run":
		runDaemon()
	case "failed":
		cmdDaemonFailed()
	case "retry":
		cmdDaemonRetry()
	case "review":
		allFlag := hasFlag("--all")
		yesFlag := hasFlag("--yes") || hasFlag("-y")
		ttyFlag := hasFlag("--tty")
		cmdDaemonReview(allFlag, yesFlag, ttyFlag)
	default:
		fmt.Printf("Unknown daemon command: %s\nWhat to do: use start, stop, restart, status, failed, retry, or review.\n", os.Args[2])
		os.Exit(1)
	}
}

func cmdDaemonStart() {
	execPath, err := os.Executable()
	if err != nil {
		execPath = "brain"
	}

	brainDir, err := knowledge.FindBrainDir()
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
	brainDir, err := knowledge.FindBrainDir()
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

func cmdDaemonRestart() {
	cmdDaemonStop()
	fmt.Println()
	cmdDaemonStart()
}

func cmdDaemonStatus() {
	brainDir, err := knowledge.FindBrainDir()
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
			if !e.IsDir() && strings.HasPrefix(e.Name(), "commit-") && strings.HasSuffix(e.Name(), ".json") {
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

func cmdDaemonFailed() {
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	failedDir := filepath.Join(brainDir, ".queue", "failed")
	entries, err := os.ReadDir(failedDir)
	if err != nil {
		fmt.Println("No failed items.")
		return
	}

	if len(entries) == 0 {
		fmt.Println("No failed items.")
		return
	}

	fmt.Printf("Failed Items (%d)\n", len(entries))
	fmt.Println("=================")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".processing")
		data, err := os.ReadFile(filepath.Join(failedDir, e.Name()))
		if err != nil {
			fmt.Printf("  %s (could not read)\n", name)
			continue
		}
		var item daemon.QueueItem
		if err := json.Unmarshal(data, &item); err != nil {
			fmt.Printf("  %s (could not parse)\n", name)
			continue
		}
		reason := item.ErrorReason
		if reason == "" {
			reason = "unknown (old format — reprocess or delete)"
		}
		fmt.Printf("  %s\n", name)
		fmt.Printf("    Error: %s\n", reason)
		fmt.Printf("    Attempts: %d\n", item.Attempts)
		fmt.Printf("    Files: %s\n", item.Files)
	}
}

func cmdDaemonRetry() {
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	failedDir := filepath.Join(brainDir, ".queue", "failed")
	entries, err := os.ReadDir(failedDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("No failed items to retry.")
		return
	}

	queueDir := filepath.Join(brainDir, ".queue")
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
		var item daemon.QueueItem
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		item.Attempts = 0
		item.ErrorReason = ""
		itemData, _ := json.Marshal(item)

		destPath := filepath.Join(queueDir, name)
		if err := os.WriteFile(destPath, itemData, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to requeue %s: %v\n", name, err)
			continue
		}
		os.Remove(srcPath)
		retried++
		fmt.Printf("  Requeued: %s\n", name)
	}

	if retried == 0 {
		fmt.Println("No failed items could be retried.")
	} else {
		fmt.Printf("\nRequeued %d item(s). Daemon will process them on next poll.\n", retried)
	}
}

func projectHash(workDir string) string {
	hash := sha256.Sum256([]byte(workDir))
	return hex.EncodeToString(hash[:4])
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

	brainDir, err := knowledge.FindBrainDir()
	if err == nil {
		workDir := filepath.Dir(brainDir)
		hash := projectHash(workDir)
		return filepath.Join(lockDir, fmt.Sprintf("brain-daemon-%s.pid", hash)), nil
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		hash := projectHash(cwd)
		return filepath.Join(lockDir, fmt.Sprintf("brain-daemon-%s.pid", hash)), nil
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
		fmt.Fprintln(os.Stderr, "What to do: run 'brain config set api-key <your-openrouter-key>'")
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

		brainDir, err := knowledge.FindBrainDir()
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
				emptyTree, _ := exec.Command("git", "-C", repo, "hash-object", "-t", "tree", "/dev/null").Output()
				emptyTreeStr := strings.TrimSpace(string(emptyTree))
				if emptyTreeStr != "" {
					out, err = exec.Command("git", "-C", repo, "diff", emptyTreeStr+"..HEAD").CombinedOutput()
					if err == nil {
						return string(out), nil
					}
				}
				return "", err
			}
			return string(out), nil
		}

			analyzeFn := func(req daemon.AnalyzeRequest) (daemon.Finding, error) {
				return daemon.Analyze(daemon.AnalyzeRequest{
					Diff:     req.Diff,
					APIKey:   apiKey,
					Model:    cfg.LLM.Model,
					Provider: cfg.LLM.Provider,
					BaseURL:  cfg.LLM.BaseURL,
				})
			}

			processed, err := daemon.ProcessItemWithDeps(
				context.Background(), processingPath, queueDir, brainDir,
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

func cmdDaemonReview(allFlag, yesFlag, ttyFlag bool) {
	ctx := context.Background()
	ctx, span := otel.StartSpan(ctx, "brain.review")
	defer otel.EndSpan(span, nil)

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	pendingDir := filepath.Join(brainDir, "pending")
	entries, err := knowledge.LoadPendingEntries(pendingDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading pending entries: %v\n", err)
		os.Exit(1)
	}

	if allFlag {
		topicFiles := map[string]string{
			"gotchas":      filepath.Join(brainDir, "gotchas.md"),
			"patterns":     filepath.Join(brainDir, "patterns.md"),
			"decisions":    filepath.Join(brainDir, "decisions.md"),
			"architecture": filepath.Join(brainDir, "architecture.md"),
		}
		totalImported := 0
		for topic, path := range topicFiles {
			count, err := knowledge.TopicEntriesToPending(topic, path, pendingDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not import from %s: %v\n", path, err)
				continue
			}
			totalImported += count
		}
		if totalImported > 0 {
			fmt.Printf("Imported %d existing entries into pending queue.\n", totalImported)
		}
		entries, _ = knowledge.LoadPendingEntries(pendingDir)
	}

	if len(entries) == 0 {
		fmt.Println("No pending entries to review.")
		fmt.Println("What to do: push some commits and let the daemon analyze them.")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	prof, err := profile.FromName(cfg.Review.Profile)
	if err != nil {
		prof = profile.DefaultProfile()
	}

	fmt.Printf("Reviewing %d pending entries (profile: %s)\n\n", len(entries), prof.Name)

	canUseTTY := tui.CanUseRawMode()
	useTUI := !yesFlag && !ttyFlag && canUseTTY

	var accepted []knowledge.PendingEntry
	var rejectedIDs []string
	var mode string

	if useTUI {
		mode = "tui"
		accepted, rejectedIDs, err = tui.RunReview(entries, prof.Name, os.Stdout)
		if err != nil {
			if accepted == nil && rejectedIDs == nil {
				fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered review.\n\n")
				mode = "line_buffered"
				accepted, rejectedIDs = doLineBufferedReviewDaemon(ctx, span, entries, prof, pendingDir)
			} else {
				fmt.Fprintf(os.Stderr, "Interactive review failed: %v\nFalling back to auto-accept.\n", err)
				mode = "auto"
				doAutoAcceptDaemon(ctx, span, entries, prof, pendingDir, true)
				return
			}
		} else if accepted == nil && rejectedIDs == nil {
			fmt.Fprintf(os.Stderr, "TUI not available, falling back to line-buffered review.\n\n")
			mode = "line_buffered"
			accepted, rejectedIDs = doLineBufferedReviewDaemon(ctx, span, entries, prof, pendingDir)
		} else {
			writeAcceptedDaemon(ctx, span, accepted, pendingDir)
			removeEntriesDaemon(accepted, rejectedIDs, pendingDir)
			fmt.Printf("\nApplied %d entries, rejected %d.\n", len(accepted), len(rejectedIDs))
		}
	} else if !yesFlag {
		mode = "line_buffered"
		accepted, rejectedIDs = doLineBufferedReviewDaemon(ctx, span, entries, prof, pendingDir)
	} else {
		mode = "auto"
		doAutoAcceptDaemon(ctx, span, entries, prof, pendingDir, true)
		return
	}

	otel.SetAttributes(span,
		otel.BrainProfile.String(prof.Name),
		otel.BrainReviewMode.String(mode),
		otel.BrainReviewTotal.Int(len(entries)),
		otel.BrainReviewAccepted.Int(len(accepted)),
		otel.BrainReviewRejected.Int(len(rejectedIDs)),
		otel.BrainDurationMs.Int64(0),
	)
}

func doAutoAcceptDaemon(ctx context.Context, span trace.Span, entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string, nonInteractive bool) {
	accepted, rejectedIDs := autoAcceptEntriesDaemon(entries, prof)
	writeAcceptedDaemon(ctx, span, accepted, pendingDir)
	removeEntriesDaemon(accepted, rejectedIDs, pendingDir)

	msg := "Auto-accepted %d entries (auto-dedup: %v)\n"
	if nonInteractive {
		msg = "Auto-accepted %d entries in non-interactive mode (auto-dedup: %v)\n"
	}
	fmt.Printf(msg, len(accepted), prof.AutoDedup)
}

func doLineBufferedReviewDaemon(ctx context.Context, span trace.Span, entries []knowledge.PendingEntry, prof profile.Profile, pendingDir string) ([]knowledge.PendingEntry, []string) {
	reader := bufio.NewReader(os.Stdin)

	var accepted []knowledge.PendingEntry
	var rejectedIDs []string

	for i, e := range entries {
		fmt.Printf("Entry %d/%d [%s]\n", i+1, len(entries), e.Topic)
		fmt.Printf("  %s\n", truncateForPromptDaemon(e.Content, 60))
		fmt.Print("Accept? (y/n/q/a): ")

		input, err := reader.ReadString('\n')
		isEOF := err != nil && strings.Contains(err.Error(), "EOF")

		if err != nil && !isEOF {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			input = "y"
		}

		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" || isEOF {
			input = "y"
		}

		decision := "accepted"
		switch input {
		case "y":
			accepted = append(accepted, e)
		case "n":
			rejectedIDs = append(rejectedIDs, e.ID)
			decision = "rejected"
		case "a":
			remaining := entries[i:]
			for _, rem := range remaining {
				accepted = append(accepted, rem)
			}
			rejectedIDs = append(rejectedIDs, collectIDsDaemon(entries[i+1:])...)
			decision = "accept_all"
			goto doneDaemon
		case "q":
			rejectedIDs = append(rejectedIDs, collectIDsDaemon(entries[i+1:])...)
			decision = "quit"
			goto doneDaemon
		default:
			accepted = append(accepted, e)
		}

		otel.RecordEvent(span, "brain.knowledge.entry",
			otel.BrainEntryID.String(e.ID),
			otel.BrainEntryTopic.String(e.Topic),
			otel.BrainEntryDecision.String(decision),
		)
	}

doneDaemon:
	writeAcceptedDaemon(ctx, span, accepted, pendingDir)
	removeEntriesDaemon(accepted, rejectedIDs, pendingDir)

	acceptedCount := len(accepted)
	rejectedCount := len(rejectedIDs)

	if rejectedCount > 0 {
		fmt.Printf("\nAccepted %d entries, rejected %d.\n", acceptedCount, rejectedCount)
	} else {
		fmt.Printf("\nAccepted %d entries.\n", acceptedCount)
	}

	return accepted, rejectedIDs
}

func collectIDsDaemon(entries []knowledge.PendingEntry) []string {
	var ids []string
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	return ids
}

func truncateForPromptDaemon(s string, maxLen int) string {
	lines := strings.Split(s, "\n")
	firstLine := lines[0]
	if len(firstLine) > maxLen {
		return firstLine[:maxLen-3] + "..."
	}
	return firstLine
}

func autoAcceptEntriesDaemon(entries []knowledge.PendingEntry, prof profile.Profile) ([]knowledge.PendingEntry, []string) {
	accepted := entries
	var rejectedIDs []string
	if prof.AutoDedup {
		seen := make(map[string]bool)
		var unique []knowledge.PendingEntry
		for _, e := range entries {
			fp := e.Fingerprint()
			if seen[fp] {
				rejectedIDs = append(rejectedIDs, e.ID)
				continue
			}
			seen[fp] = true
			unique = append(unique, e)
		}
		accepted = unique
	}
	return accepted, rejectedIDs
}

func writeAcceptedDaemon(ctx context.Context, span trace.Span, accepted []knowledge.PendingEntry, pendingDir string) {
	ctx, writeSpan := otel.StartSpan(ctx, "brain.knowledge.write")
	defer otel.EndSpan(writeSpan, nil)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	brainDir, _ := knowledge.FindBrainDir()
	idx, _ := knowledge.LoadIndex(brainDir)
	now := time.Now()

	for _, e := range accepted {
		path, err := knowledge.TopicFilePath(e.Topic)
		if err != nil {
			continue
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			continue
		}
		fmt.Fprintf(f, "\n### [%s] %s\n\n", timestamp, e.Content)
		f.Close()

		idx.Set(e.Topic, timestamp, knowledge.IndexEntry{
			Strength:       1.0,
			RetrievalCount: 0,
			LastRetrieved:  now,
			HalfLifeDays:   7,
			Confidence:     e.Confidence,
			Topics:         e.Topics,
		})

		otel.RecordEvent(writeSpan, "brain.knowledge.entry_written",
			otel.BrainEntryID.String(e.ID),
			otel.BrainEntryTopic.String(e.Topic),
		)
	}
	idx.Save(brainDir)

	otel.SetAttributes(writeSpan,
		otel.BrainReviewAccepted.Int(len(accepted)),
	)
}

func removeEntriesDaemon(accepted []knowledge.PendingEntry, rejectedIDs []string, pendingDir string) {
	for _, id := range rejectedIDs {
		knowledge.RemovePendingEntry(pendingDir, id)
	}
	for _, e := range accepted {
		knowledge.RemovePendingEntry(pendingDir, e.ID)
	}
}
