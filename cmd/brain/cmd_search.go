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

	brainDir, err := knowledge.FindBrainDir()
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

	idx, _ := knowledge.LoadIndex(brainDir)

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
		fmt.Printf("Search: \"%s\" — %d matches\n\n", query, len(matches))

		type groupedMatch struct {
			topic   string
			matches []Match
		}
		groups := make(map[string][]Match)
		for _, m := range matches {
			topic := fileToTopic[m.File]
			groups[topic] = append(groups[topic], m)
		}

		topicOrder := []string{"gotchas", "patterns", "decisions", "architecture", "memory"}
		for _, topic := range topicOrder {
			ms, ok := groups[topic]
			if !ok || len(ms) == 0 {
				continue
			}
			fmt.Printf("%s (%d matches)\n", strings.ToUpper(topic), len(ms))
			for _, m := range ms {
				_, msg := stripMarkdownPrefix(m.Content)
				if msg == "" {
					msg = m.Content
				}
				fmt.Printf("  %s:%d  %s\n", m.File, m.Line, strings.TrimSpace(msg))
			}
			fmt.Println()
		}
	}
}
