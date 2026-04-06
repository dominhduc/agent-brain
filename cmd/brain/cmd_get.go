package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dominhduc/agent-brain/internal/brain"
)

func cmdGet(jsonFlag, summaryFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain get <topic>")
		fmt.Println("Topics: memory, gotchas, patterns, decisions, architecture, all")
		fmt.Println("Flags:")
		fmt.Println("  --summary    Show summary with entry counts and duplicate warnings")
		fmt.Println("  --json       Output as JSON")
		fmt.Println("What to do: specify a topic name to retrieve.")
		os.Exit(1)
	}

	topic := os.Args[2]

	if summaryFlag {
		summaries, err := brain.GetAllSummaries()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
			os.Exit(1)
		}

		if jsonFlag {
			data, _ := json.MarshalIndent(summaries, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println("# Project Memory Summary")
			fmt.Println()
			for _, s := range summaries {
				status := ""
				if s.HasDuplicates {
					status = " ⚠️ duplicates"
				}
				fmt.Printf("- %s: %d entries, %d lines%s\n", s.Name, s.EntryCount, s.LineCount, status)
			}
			fmt.Println()
			fmt.Println("For full content, run: brain get all")
		}
		return
	}

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
			content, err := brain.GetAllTopicsWithSummary()
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
