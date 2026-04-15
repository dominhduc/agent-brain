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
	"time"

	"github.com/dominhduc/agent-brain/internal/knowledge"
)

var entryLineRe = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)

func stripMarkdownPrefix(line string) (timestamp, message string) {
	matches := entryLineRe.FindStringSubmatch(line)
	if matches != nil {
		return matches[1], strings.TrimSpace(line[len(matches[0]):])
	}
	return "", line
}

func relativeTime(timestamp string, now time.Time) string {
	t, err := time.Parse("2006-01-02 15:04:05", timestamp)
	if err != nil {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		return t.Format("Jan 06")
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 06")
	}
}

func cmdGet(jsonFlag, summaryFlag, compactFlag, messageOnlyFlag, fullFlag bool) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: brain get <topic>")
		fmt.Println("Topics: memory, gotchas, patterns, decisions, architecture, all")
		fmt.Println("Flags:")
		fmt.Println("  --summary      Show summary with entry counts and duplicate warnings")
		fmt.Println("  --json         Output as JSON (structured format)")
		fmt.Println("  --compact      One-line-per-entry, no blank lines")
		fmt.Println("  --message-only Output only the message text (no timestamps, no scores)")
		fmt.Println("  --full         Show complete content (default: tiered view)")
		fmt.Println("  --focus        Filter by topic (e.g., --focus \"infrastructure\")")
		fmt.Println("What to do: specify a topic name to retrieve.")
		os.Exit(1)
	}

	topic := os.Args[2]

	if strings.HasPrefix(topic, "--") && hasFlag("--focus") {
		topic = "all"
		fmt.Fprintln(os.Stderr, "Tip: use \"brain get all --focus <topic>\" for explicit syntax.")
	}

	if hasFlag("--search") {
		cmdGetSearch(topic, jsonFlag)
		return
	}

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
					status = " ⚠ duplicates"
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
			result := make(map[string]interface{})
			for _, t := range knowledge.AvailableTopics() {
				entries, err := knowledge.GetTopicEntries(t)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", t, err)
					os.Exit(1)
				}
				result[t] = map[string]interface{}{
					"entry_count": len(entries),
					"entries":     entries,
				}
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else if fullFlag {
			content, err := knowledge.GetAllTopicsWithSummary()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
				os.Exit(1)
			}
			fmt.Println(content)
		} else {
			content, err := getTieredAll(idx, now)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' first.\n", err)
				os.Exit(1)
			}
			fmt.Println(content)
		}
		return
	}

	var knownTopics = []string{"memory", "gotchas", "patterns", "decisions", "architecture", "all"}

	isKnown := false
	for _, t := range knownTopics {
		if strings.EqualFold(t, topic) {
			isKnown = true
			break
		}
	}

	if !isKnown {
		cmdGetSearch(topic, jsonFlag)
		return
	}

	path, err := knowledge.TopicFilePath(topic)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: topic '%s' not found. Did you mean to search? Use 'brain get --search %s'.\n", topic, topic)
		} else {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", topic, err)
		}
		os.Exit(1)
	}

	brainDir, _ := knowledge.FindBrainDir()
	idx, _ := knowledge.LoadIndex(brainDir)
	now := time.Now()

	if jsonFlag {
		entries, err := knowledge.GetTopicEntries(topic)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		type topicJSON struct {
			Topic   string                  `json:"topic"`
			Count   int                     `json:"entry_count"`
			Entries []knowledge.TopicEntry `json:"entries"`
		}
		data, _ := json.MarshalIndent(topicJSON{
			Topic:   topic,
			Count:   len(entries),
			Entries: entries,
		}, "", "  ")
		fmt.Println(string(data))
		return
	}

	if messageOnlyFlag {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			_, msg := stripMarkdownPrefix(line)
			if msg != "" && msg != line {
				fmt.Println(msg)
			}
		}
		return
	}

	if compactFlag {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			matches := entryLineRe.FindStringSubmatch(line)
			if matches != nil {
				timestamp := matches[1]
				entry, found := idx.Get(topic, timestamp)
				strength := ""
				if found {
					strength = fmt.Sprintf("%.2f  ", knowledge.CalculateStrength(entry, now))
					entry.RetrievalCount++
					entry.LastRetrieved = now
					idx.Set(topic, timestamp, entry)
				}
				_, msg := stripMarkdownPrefix(line)
				rel := relativeTime(timestamp, now)
				if rel != "" {
					rel = "  " + rel
				}
				fmt.Printf("%s%s%s\n", strength, msg, rel)
			}
		}
		idx.Save(brainDir)
		knowledge.RecordRetrieval(brainDir, getRetrievedKeys(topic, data, idx, now))
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var retrievedKeys []string
	fmt.Println("Strength: 1.00=high confidence  0.97=medium  <0.95=low")
	fmt.Println()
	for scanner.Scan() {
		line := scanner.Text()
		matches := entryLineRe.FindStringSubmatch(line)
		if matches != nil {
			timestamp := matches[1]
			entry, found := idx.Get(topic, timestamp)
			if found {
				strength := knowledge.CalculateStrength(entry, now)
				_, msg := stripMarkdownPrefix(line)
				fmt.Printf("%.2f  ### [%s] %s\n", strength, timestamp, msg)
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

func getTieredAll(idx *knowledge.Index, now time.Time) (string, error) {
	summaries, err := knowledge.GetAllSummaries()
	if err != nil {
		return "", err
	}

	type scoredEntry struct {
		timestamp string
		message   string
		strength  float64
		topic     string
		age       time.Duration
	}

	var allEntries []scoredEntry

	for _, topicFile := range knowledge.AvailableTopics() {
		entries, err := knowledge.GetTopicEntries(topicFile)
		if err != nil {
			continue
		}
		for _, e := range entries {
			t, _ := time.Parse("2006-01-02 15:04:05", e.Timestamp)
			var strength float64 = 1.0
			if idx != nil {
				idxEntry, found := idx.Get(topicFile, e.Timestamp)
				if found {
					strength = knowledge.CalculateStrength(idxEntry, now)
				}
			}
			allEntries = append(allEntries, scoredEntry{
				timestamp: e.Timestamp,
				message:   e.Message,
				strength:  strength,
				topic:     topicFile,
				age:       now.Sub(t),
			})
		}
	}

	var result strings.Builder

	result.WriteString("PROJECT MEMORY OVERVIEW\n")
	result.WriteString(strings.Repeat("─", 50) + "\n")
	for _, s := range summaries {
		status := ""
		if s.HasDuplicates {
			status = "  ⚠ duplicates"
		}
		result.WriteString(fmt.Sprintf("  %-15s %d entries  %d lines%s\n", s.Name, s.EntryCount, s.LineCount, status))
	}
	result.WriteString("\n")

	var recentEntries []scoredEntry
	for _, e := range allEntries {
		if e.age <= 7*24*time.Hour {
			recentEntries = append(recentEntries, e)
		}
	}

	if len(recentEntries) > 0 {
		result.WriteString(fmt.Sprintf("RECENT (last 7 days, %d entries)\n", len(recentEntries)))
		result.WriteString(strings.Repeat("─", 50) + "\n")
		for _, e := range recentEntries {
			rel := relativeTime(e.timestamp, now)
			result.WriteString(fmt.Sprintf("  %.2f  %s  %s\n", e.strength, e.message, rel))
		}
		result.WriteString("\n")
	}

	topCount := 15
	if len(allEntries) < topCount {
		topCount = len(allEntries)
	}
	if len(allEntries) > topCount {
		result.WriteString(fmt.Sprintf("TOP %d BY RETRIEVAL\n", topCount))
		result.WriteString(strings.Repeat("─", 50) + "\n")
		for i := 0; i < topCount; i++ {
			e := allEntries[i]
			result.WriteString(fmt.Sprintf("  %.2f  %s\n", e.strength, e.message))
		}
		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("Total: %d entries across %d topics.\n", len(allEntries), len(knowledge.AvailableTopics())))
	result.WriteString("Run 'brain get all --full' for complete content.\n")

	return result.String(), nil
}

func getRetrievedKeys(topic string, data []byte, idx *knowledge.Index, now time.Time) []string {
	var keys []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		matches := entryLineRe.FindStringSubmatch(line)
		if matches != nil {
			timestamp := matches[1]
			if _, found := idx.Get(topic, timestamp); found {
				keys = append(keys, knowledge.MakeKey(topic, timestamp))
			}
		}
	}
	return keys
}

func cmdGetSearch(query string, jsonFlag bool) {
	topicFilter := ""
	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--topic" && i+1 < len(os.Args) {
			topicFilter = os.Args[i+1]
			break
		}
		if os.Args[i] == "--search" && i+1 < len(os.Args) {
			query = os.Args[i+1]
		}
	}

	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nWhat to do: run 'brain init' in your project directory first.\n", err)
		os.Exit(1)
	}

	if hub, err := knowledge.Open(brainDir); err == nil {
		_ = hub.TrackCommand("get")
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

	results := make(map[string][]string)
	var totalMatches int

	for _, file := range files {
		filePath := filepath.Join(brainDir, file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		topic := fileToTopic[file]
		if topicFilter != "" && topic != topicFilter {
			continue
		}

		re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(query))
		if err != nil {
			continue
		}

		var matches []string
		var currentEntry strings.Builder
		var inEntry bool

		scanner := bufio.NewScanner(bytes.NewReader(content))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "### [") {
				if inEntry && currentEntry.Len() > 0 {
					entryText := currentEntry.String()
					if re.MatchString(entryText) {
						matches = append(matches, strings.TrimSpace(entryText))
					}
				}
				currentEntry.Reset()
				currentEntry.WriteString(line + "\n")
				inEntry = true
			} else if inEntry {
				currentEntry.WriteString(line + "\n")
			}
		}
		if inEntry && currentEntry.Len() > 0 {
			entryText := currentEntry.String()
			if re.MatchString(entryText) {
				matches = append(matches, strings.TrimSpace(entryText))
			}
		}

		if len(matches) > 0 {
			results[topic] = matches
			totalMatches += len(matches)
		}
	}

	if jsonFlag {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"query":       query,
			"total":       totalMatches,
			"by_topic":    results,
		}, "", "  ")
		fmt.Println(string(data))
		return
	}

	if totalMatches == 0 {
		fmt.Printf("No results for \"%s\"\n", query)
		return
	}

	fmt.Printf("Search results for \"%s\" (%d matches)\n\n", query, totalMatches)
	for topic, matches := range results {
		fmt.Printf("## %s (%d)\n\n", topic, len(matches))
		for _, m := range matches {
			lines := strings.Split(m, "\n")
			for _, l := range lines {
				fmt.Println(l)
			}
			fmt.Println()
		}
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
			rel := relativeTime(se.timestamp, now)
			if rel != "" {
				rel = "  " + rel
			}
			result.WriteString(fmt.Sprintf("%.2f  %s%s\n", se.strength, se.content, rel))
		}
		result.WriteString("\n")
	}

	if len(medRelevance) > 0 {
		result.WriteString(fmt.Sprintf("## General (%d entries)\n\n", len(medRelevance)))
		for _, se := range medRelevance {
			rel := relativeTime(se.timestamp, now)
			if rel != "" {
				rel = "  " + rel
			}
			result.WriteString(fmt.Sprintf("%.2f  %s%s\n", se.strength, se.content, rel))
		}
		result.WriteString("\n")
	}

	if len(otherEntries) > 0 {
		result.WriteString(fmt.Sprintf("## Other Topics (collapsed, %d entries)\n\n", len(otherEntries)))
		result.WriteString("<details>\n")
		for _, se := range otherEntries {
			rel := relativeTime(se.timestamp, now)
			if rel != "" {
				rel = "  " + rel
			}
			result.WriteString(fmt.Sprintf("%.2f  %s%s\n", se.strength, se.content, rel))
		}
		result.WriteString("</details>\n")
	}

	return result.String(), nil
}
