package main

import (
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/brain"
)

func cmdDedup(dryRun bool) {
	_, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
		os.Exit(1)
	}

	report, err := brain.RunDedup(dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(report.Groups) == 0 {
		fmt.Println("No duplicates found. All entries are unique!")
		return
	}

	fmt.Printf("Found %d duplicate groups (%d total duplicates):\n\n",
		len(report.Groups), report.TotalRemoved)

	for i, group := range report.Groups {
		fmt.Printf("  [%d] Content: %s\n", i+1, truncate(group.Message, 60))
		fmt.Printf("      Kept: %s.md line %d\n", group.Kept.Topic, group.Kept.LineNum)
		for _, dup := range group.Duplicates {
			crossTopic := ""
			if dup.Topic != group.Kept.Topic {
				crossTopic = " (cross-topic)"
			}
			fmt.Printf("      - %s.md line %d%s\n", dup.Topic, dup.LineNum, crossTopic)
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("Dry run — no files modified.")
	} else {
		fmt.Printf("Removed %d duplicate entries.\n", report.TotalRemoved)
		if report.CrossTopicCount > 0 {
			fmt.Printf("  (%d were cross-topic duplicates)\n", report.CrossTopicCount)
		}
		fmt.Printf("Archived to %s\n", report.ArchivedPath)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
