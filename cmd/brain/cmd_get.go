package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

var entryLineRe = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

func cmdGet(jsonFlag, summaryFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain get <topic>")
		fmt.Println("Topics: memory, gotchas, patterns, decisions, architecture, all")
		fmt.Println("Flags:")
		fmt.Println("  --summary    Show summary with entry counts and duplicate warnings")
		fmt.Println("  --json       Output as JSON")
		fmt.Println("  --focus      Filter by topic (e.g., --focus \"infrastructure\")")
		fmt.Println("What to do: specify a topic name to retrieve.")
		os.Exit(1)
	}

	topic := os.Args[2]
	focusFlag := hasFlag("--focus")
	var focusTopic string
	if focusFlag {
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--focus" && i+1 < len(os.Args) {
				focusTopic = os.Args[i+1]
				break
			}
		}
	}

	if brainDir, err := knowledge.FindBrainDir(); err == nil {
		if hub, err := knowledge.Open(brainDir); err == nil {
			_ = hub.TrackCommand("get")
			if topic != "all" {
				_ = hub.TrackTopicAccess(topic)
			}
		}
	}

	if summaryFlag {
		summaries, err := knowledge.GetAllSummaries()
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
		brainDir, _ := knowledge.FindBrainDir()
		idx, _ := knowledge.LoadIndex(brainDir)
		now := time.Now()

		if focusFlag && focusTopic != "" {
			content, err := getFocusedTopics(focusTopic, idx, now, jsonFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(content)
			return
		}

		if jsonFlag {
			topics := map[string]string{}
			for _, t := range knowledge.AvailableTopics() {
				c, err := knowledge.GetTopic(t)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", t, err)
					os.Exit(1)
				}
				topics[t] = c
			}
			data, _ := json.MarshalIndent(topics, "", "  ")
			fmt.Println(string(data))
		} else {
			content, err := knowledge.GetAllTopicsWithSummary()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
				os.Exit(1)
			}
			fmt.Println(content)
		}
		return
	}

	path, err := knowledge.TopicFilePath(topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", topic, err)
		os.Exit(1)
	}

	brainDir, _ := knowledge.FindBrainDir()
	idx, _ := knowledge.LoadIndex(brainDir)
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
				strength := knowledge.CalculateStrength(entry, now)
				fmt.Printf("●%.2f  %s\n", strength, line)
				entry.RetrievalCount++
				entry.LastRetrieved = now
				idx.Set(topic, timestamp, entry)
				retrievedKeys = append(retrievedKeys, knowledge.MakeKey(topic, timestamp))
			} else {
				fmt.Println(line)
			}
		} else {
			fmt.Println(line)
		}
	}

	if len(retrievedKeys) > 0 {
		idx.Save(brainDir)
		knowledge.RecordRetrieval(brainDir, retrievedKeys)
	}
}

func getFocusedTopics(focusTopic string, idx *knowledge.Index, now time.Time, jsonFlag bool) (string, error) {
	type scoredEntry struct {
		topicFile   string
		timestamp   string
		content     string
		strength    float64
		relevance   int
		entry       knowledge.IndexEntry
	}

	var highRelevance, medRelevance, otherEntries []scoredEntry

	for _, topicFile := range knowledge.AvailableTopics() {
		content, err := knowledge.GetTopic(topicFile)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(content))
		var currentContent strings.Builder
		var currentTimestamp string

		flushEntry := func() {
			if currentTimestamp == "" {
				return
			}
			entry, found := idx.Get(topicFile, currentTimestamp)
			if !found {
				return
			}

			strength := knowledge.CalculateStrength(entry, now)
			content := strings.TrimSpace(currentContent.String())

			var relevance int
			for _, t := range entry.Topics {
				if t == focusTopic {
					relevance = 2
					break
				}
				if t == "general" {
					relevance = 1
				}
			}

			se := scoredEntry{
				topicFile: topicFile,
				timestamp: currentTimestamp,
				content:   content,
				strength:  strength,
				relevance: relevance,
				entry:     entry,
			}

			switch relevance {
			case 2:
				highRelevance = append(highRelevance, se)
			case 1:
				medRelevance = append(medRelevance, se)
			default:
				otherEntries = append(otherEntries, se)
			}
		}

		for scanner.Scan() {
			line := scanner.Text()
			matches := entryLineRe.FindStringSubmatch(line)
			if matches != nil {
				flushEntry()
				currentTimestamp = matches[1]
				currentContent.Reset()
				currentContent.WriteString(strings.TrimPrefix(line, matches[0]))
			} else if currentTimestamp != "" {
				currentContent.WriteString(" ")
				currentContent.WriteString(strings.TrimSpace(line))
			}
		}
		flushEntry()
	}

	if jsonFlag {
		result := map[string]interface{}{
			"focus":          focusTopic,
			"high_relevance": len(highRelevance),
			"general":        len(medRelevance),
			"other_topics":   len(otherEntries),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return string(data), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Focused: %s\n\n", strings.ToUpper(focusTopic)))

	if len(highRelevance) > 0 {
		result.WriteString(fmt.Sprintf("## High Relevance (%d entries)\n\n", len(highRelevance)))
		for _, se := range highRelevance {
			result.WriteString(fmt.Sprintf("●%.2f  ### [%s] %s\n", se.strength, se.timestamp, se.content))
		}
		result.WriteString("\n")
	}

	if len(medRelevance) > 0 {
		result.WriteString(fmt.Sprintf("## General (%d entries)\n\n", len(medRelevance)))
		for _, se := range medRelevance {
			result.WriteString(fmt.Sprintf("●%.2f  ### [%s] %s\n", se.strength, se.timestamp, se.content))
		}
		result.WriteString("\n")
	}

	if len(otherEntries) > 0 {
		result.WriteString(fmt.Sprintf("## Other Topics (collapsed, %d entries)\n\n", len(otherEntries)))
		result.WriteString("<details>\n")
		for _, se := range otherEntries {
			result.WriteString(fmt.Sprintf("●%.2f  ### [%s] %s\n", se.strength, se.timestamp, se.content))
		}
		result.WriteString("</details>\n")
	}

	return result.String(), nil
}
