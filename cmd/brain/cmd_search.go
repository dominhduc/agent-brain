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
)

func cmdSearch(jsonFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain search <query>")
		fmt.Println("What to do: provide a search term to look for across knowledge files.")
		os.Exit(1)
	}

	query := os.Args[2]

	brainDir, err := brain.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' in your project directory first.\n", err)
		os.Exit(1)
	}

	files := []string{"MEMORY.md", "gotchas.md", "patterns.md", "decisions.md", "architecture.md"}
	pattern := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))

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
