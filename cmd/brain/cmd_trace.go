package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/daemon"
	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdTrace() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain trace <action>")
		fmt.Println()
		fmt.Println("Actions:")
		fmt.Println("  save --outcome <success|partial|failure> [--goal \"description\"]")
		fmt.Println("  extract [--dry-run]")
		fmt.Println("  list")
		fmt.Println("  step --action <action> --target <target> --outcome <success|failure|partial>")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "save":
		cmdTraceSave()
	case "extract":
		cmdTraceExtract()
	case "list":
		cmdTraceList()
	case "step":
		cmdTraceStep()
	default:
		fmt.Printf("Unknown trace action: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func cmdTraceSave() {
	outcome := flagValue("--outcome", "partial")
	goal := flagValue("--goal", "")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := hub.FinalizeTrace(outcome, goal); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Trace saved with outcome: %s\n", outcome)
}

func cmdTraceExtract() {
	dryRun := hasFlag("--dry-run")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	traces, err := hub.LoadUnextractedTraces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading traces: %v\n", err)
		os.Exit(1)
	}

	if len(traces) == 0 {
		fmt.Println("No unextracted traces found.")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	apiKey := cfg.LLM.APIKey
	if envKey := os.Getenv("BRAIN_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	extracted := 0
	for _, trace := range traces {
		prompt := knowledge.BuildTraceExtractionPrompt(trace)
		finding, err := daemon.AnalyzeWithPrompt(daemon.AnalyzeRequest{
			Diff:     prompt,
			APIKey:   apiKey,
			Model:    cfg.LLM.Model,
			Provider: cfg.LLM.Provider,
			BaseURL:  cfg.LLM.BaseURL,
		}, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to extract from trace %s: %v\n", trace.SessionID, err)
			continue
		}

		if !dryRun {
			allSucceeded := true
			for _, item := range finding.Items {
				content := item.Title + ": " + item.Content
				entry := knowledge.PendingEntry{
					ID:         fmt.Sprintf("trace-%s-%d", trace.SessionID, extracted),
					Topic:      item.Topic,
					Content:    content,
					Timestamp:  time.Now(),
					Confidence: item.Confidence,
					Source:     "trace",
					Topics:     item.Tags,
				}
				if err := hub.AddPending(entry); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save entry: %v\n", err)
					allSucceeded = false
				}
			}
			if allSucceeded {
				hub.MarkTraceExtracted(trace.SessionID)
			}
		}

		if dryRun {
			fmt.Printf("Would extract %d entries from trace %s (%s)\n", len(finding.Items), trace.SessionID, trace.Outcome)
		} else {
			fmt.Printf("Extracted %d entries from trace %s (%s)\n", len(finding.Items), trace.SessionID, trace.Outcome)
		}
		extracted++
	}

	if dryRun {
		fmt.Printf("\n%d traces would be processed. Run without --dry-run to extract.\n", extracted)
	} else {
		fmt.Printf("\nProcessed %d traces. Run 'brain daemon review' to approve entries.\n", extracted)
	}
}

func cmdTraceList() {
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	traces, err := hub.LoadTraces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading traces: %v\n", err)
		os.Exit(1)
	}

	if len(traces) == 0 {
		fmt.Println("No traces found.")
		return
	}

	fmt.Printf("TRACES (%d)\n", len(traces))
	for _, t := range traces {
		fmt.Printf("  %s  [%s]  %d steps  %s\n", t.SessionID, t.Outcome, len(t.Steps), t.Goal)
	}
}

func cmdTraceStep() {
	action := flagValue("--action", "")
	target := flagValue("--target", "")
	outcome := flagValue("--outcome", "success")
	reasoning := flagValue("--reasoning", "")

	if action == "" {
		fmt.Fprintln(os.Stderr, "Error: --action is required")
		os.Exit(1)
	}

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	step := knowledge.TraceStep{
		Action:    action,
		Target:    target,
		Result:    outcome,
		Reasoning: reasoning,
		Outcome:   outcome,
	}

	if err := hub.AppendTraceStep(step); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Step recorded: %s → %s\n", action, outcome)
}

func flagValue(flag string, defaultVal string) string {
	for i, arg := range os.Args {
		if arg == flag && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return defaultVal
}
