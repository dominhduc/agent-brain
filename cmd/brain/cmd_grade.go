package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/daemon"
	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdGrade() {
	dryRun := hasFlag("--dry-run")

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no .brain/ directory found.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain init' first.")
		os.Exit(1)
	}

	hub, err := knowledge.Open(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	candidates, err := hub.GradeCandidates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading candidates: %v\n", err)
		os.Exit(1)
	}

	if len(candidates) == 0 {
		fmt.Println("No entries to grade.")
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
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key not configured.")
		fmt.Fprintln(os.Stderr, "What to do: run 'brain config set api-key <key>'")
		os.Exit(1)
	}

	batchSize := 10
	totalBatches := (len(candidates) + batchSize - 1) / batchSize
	var allGrades []knowledge.Grade
	failedBatches := 0
	for i := 0; i < len(candidates); i += batchSize {
		end := i + batchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[i:end]
		batchNum := i/batchSize + 1

		fmt.Printf("Grading batch %d/%d (%d entries)...\n", batchNum, totalBatches, len(batch))

		prompt := knowledge.BuildGradingPrompt(batch)
		if prompt == "" {
			fmt.Fprintf(os.Stderr, "Warning: failed to build grading prompt for batch %d\n", batchNum)
			failedBatches++
			continue
		}
		content, err := daemon.CallLLM(daemon.AnalyzeRequest{
			APIKey:   apiKey,
			Model:    cfg.LLM.Model,
			Provider: cfg.LLM.Provider,
			BaseURL:  cfg.LLM.BaseURL,
		}, prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: grading batch %d failed: %v\n", batchNum, err)
			failedBatches++
			continue
		}

		grades, err := knowledge.ParseGradingResponse(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse grades: %v\n", err)
			failedBatches++
			continue
		}
		allGrades = append(allGrades, grades...)
	}

	if len(allGrades) == 0 {
		if failedBatches > 0 {
			fmt.Fprintf(os.Stderr, "All %d grading batches failed. Check your API key and model.\n", failedBatches)
		} else {
			fmt.Println("Grading completed but no parseable grades returned. Try again.")
		}
		return
	}

	report, err := hub.ApplyGrades(allGrades, dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error applying grades: %v\n", err)
		os.Exit(1)
	}

	suffix := ""
	if dryRun {
		suffix = " (dry-run)"
	}
	fmt.Printf("GRADE REPORT%s\n", suffix)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  Keep:    %d\n", report.KeepCount)
	fmt.Printf("  Rewrite: %d\n", report.RewriteCount)
	fmt.Printf("  Archive: %d\n", report.ArchiveCount)
	if failedBatches > 0 {
		fmt.Printf("  Failed:  %d batches\n", failedBatches)
	}
	fmt.Println()

	for _, g := range report.Grades {
		verdict := strings.ToUpper(g.Verdict)
		shortened := g.Key
		if len(shortened) > 60 {
			shortened = shortened[:57] + "..."
		}
		fmt.Printf("  [%s] %s — %s\n", verdict, shortened, g.Reason)
	}

	if dryRun {
		fmt.Println("\nRun 'brain grade' to apply verdicts.")
	} else {
		fmt.Printf("\nApplied %d verdicts.\n", len(report.Grades))
	}
}
