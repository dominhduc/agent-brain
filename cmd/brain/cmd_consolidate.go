package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdConsolidate() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain consolidate [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --dry-run    Show proposals without applying")
		fmt.Println("  --apply      Apply consolidations")
		fmt.Println("  --topic      Filter to specific topic (e.g., --topic gotchas)")
		fmt.Println()
		fmt.Println("What to do: run 'brain consolidate --dry-run' first to review proposals.")
		os.Exit(1)
	}

	dryRun := hasFlag("--dry-run")
	applyFlag := hasFlag("--apply")
	autoFlag := hasFlag("--auto")

	topicFilter := ""
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--topic" && i+1 < len(os.Args) {
			topicFilter = os.Args[i+1]
		}
	}

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

	proposals, err := hub.FindConsolidations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding consolidations: %v\n", err)
		os.Exit(1)
	}

	if topicFilter != "" {
		var filtered []knowledge.ConsolidationProposal
		for _, p := range proposals {
			if p.Topic == topicFilter {
				filtered = append(filtered, p)
			}
		}
		proposals = filtered
	}

	if len(proposals) == 0 {
		fmt.Println("No consolidation opportunities found.")
		return
	}

	if dryRun || (!applyFlag && !autoFlag) {
		fmt.Println(formatConsolidationProposal(proposals))
		return
	}

	if applyFlag || autoFlag {
		applied := 0
		for _, p := range proposals {
			if topicFilter != "" && p.Topic != topicFilter {
				continue
			}
			if err := hub.ApplyConsolidation(p); err != nil {
				fmt.Fprintf(os.Stderr, "Error applying consolidation %s: %v\n", p.ID, err)
				continue
			}
			applied++
		}
		fmt.Printf("Applied %d consolidations.\n", applied)
	}
}

func formatConsolidationProposal(proposals []knowledge.ConsolidationProposal) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CONSOLIDATION PROPOSAL (%d clusters found)\n", len(proposals)))
	sb.WriteString(strings.Repeat("─", 50) + "\n\n")

	for i, p := range proposals {
		sb.WriteString(fmt.Sprintf("Cluster %d: %s (%d entries → 1)\n", i+1, p.Topic, len(p.Sources)))
		sb.WriteString("  Sources:\n")
		for _, s := range p.Sources {
			sb.WriteString(fmt.Sprintf("    • %q (strength: %.2f)\n", s.Message, s.Strength))
		}
		sb.WriteString("  Proposed:\n")
		sb.WriteString("    " + p.Consolidated + "\n")
		sb.WriteString("\n")
	}

	sb.WriteString("Run 'brain consolidate --apply' to apply these consolidations.\n")
	return sb.String()
}
