package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/index"
	"github.com/dominhduc/agent-brain/internal/outcome"
)

var entryLineRe = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

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

	path, err := brain.TopicFilePath(topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", topic, err)
		os.Exit(1)
	}

	brainDir, _ := brain.FindBrainDir()
	idx, _ := index.Load(brainDir)
	now := time.Now()

	if jsonFlag {
		fmt.Println(string(data))
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var retrievedKeys []string
	for scanner.Scan() {
		line := scanner.Text()
		matches := entryLineRe.FindStringSubmatch(line)
		if matches != nil {
			timestamp := matches[1]
			entry, found := idx.Get(topic, timestamp)
			if found {
				strength := index.CalculateStrength(entry, now)
				fmt.Printf("●%.2f  %s\n", strength, line)
				entry.RetrievalCount++
				entry.LastRetrieved = now
				idx.Set(topic, timestamp, entry)
				retrievedKeys = append(retrievedKeys, index.MakeKey(topic, timestamp))
			} else {
				fmt.Println(line)
			}
		} else {
			fmt.Println(line)
		}
	}

	if len(retrievedKeys) > 0 {
		idx.Save(brainDir)
		outcome.Track(brainDir, retrievedKeys)
	}
}
