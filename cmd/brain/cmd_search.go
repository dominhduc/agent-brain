package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dominhduc/agent-brain/internal/brain"
	"github.com/dominhduc/agent-brain/internal/index"
	"github.com/dominhduc/agent-brain/internal/knowledge"
)

func cmdSearch(jsonFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain search <query>")
		fmt.Println("Flags:")
		fmt.Println("  --json       Output as JSON")
		fmt.Println("  --topic      Filter by topic (e.g., --topic \"infrastructure\")")
		fmt.Println("What to do: provide a search term to look for across knowledge files.")
		os.Exit(1)
	}

	query := os.Args[2]
	topicFilter := ""
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--topic" && i+1 < len(os.Args) {
			topicFilter = os.Args[i+1]
			break
		}
	}

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' in your project directory first.\n", err)
		os.Exit(1)
	}

	if hub, err := knowledge.Open(brainDir); err == nil {
		_ = hub.TrackCommand("search")
		_ = hub.TrackSearch(query)
	}

	files := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	fileToTopic := map[string]string{
		"MEMORY.md":      "memory",
		"gotchas.md":     "gotchas",
		"patterns.md":    "patterns",
		"decisions.md":   "decisions",
		"architecture.md": "architecture",
	}
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))

	idx, _ := index.Load(brainDir)

	type Match struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}

	var matches []Match

	for _, f := range files {
		path := filepath.Join(brainDir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			if pattern.MatchString(line) {
				if topicFilter != "" && idx != nil {
					regexMatches := entryLineRe.FindStringSubmatch(line)
					if regexMatches != nil {
						timestamp := regexMatches[1]
						fileTopic := fileToTopic[f]
						entry, found := idx.Get(fileTopic, timestamp)
						if !found {
							lineNum++
							continue
						}
						hasTopic := false
						for _, t := range entry.Topics {
							if t == topicFilter {
								hasTopic = true
								break
							}
						}
						if !hasTopic {
							lineNum++
							continue
						}
					} else {
						lineNum++
						continue
					}
				}
				matches = append(matches, Match{
					File:    f,
					Line:    lineNum + 1,
					Content: strings.TrimSpace(line),
				})
			}
			lineNum++
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No matches found for '%s'\n", query)
		return
	}

	if jsonFlag {
		data, _ := json.MarshalIndent(matches, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Found %d match(es) for '%s':\n\n", len(matches), query)
		for _, m := range matches {
			fmt.Printf("  %s:%d  %s\n", m.File, m.Line, m.Content)
		}
	}
}
