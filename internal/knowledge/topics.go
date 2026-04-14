package knowledge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func (h *Hub) Get(topic string) (string, error) {
	path, err := h.topicFilePath(topic)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w.\nWhat to do: run 'brain init' to recreate missing files.", filepath.Base(path), err)
	}
	return string(data), nil
}

func GetTopic(name string) (string, error) {
	path, err := TopicFilePath(name)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w.\nWhat to do: run 'brain init' to recreate missing files.", filepath.Base(path), err)
	}
	return string(data), nil
}

func GetAllTopics() (string, error) {
	var result strings.Builder
	for _, topic := range AvailableTopics() {
		content, err := GetTopic(topic)
		if err != nil {
			return "", err
		}
		result.WriteString(fmt.Sprintf("## %s\n\n%s\n---\n\n", strings.ToUpper(topic), content))
	}
	return result.String(), nil
}

func AddEntry(topic string, message string) error {
	path, err := TopicFilePath(topic)
	if err != nil {
		return err
	}

	normalizedMsg := normalizeEntry(message)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	if data != nil {
		existing := string(data)
		lines := strings.Split(existing, "\n")
		for _, line := range lines {
			msg := extractMessageFromEntry(line)
			lineNormalized := normalizeEntry(msg)
			if lineNormalized == normalizedMsg && lineNormalized != "" {
				return nil
			}
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n### [%s] %s\n\n", timestamp, message)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w.\nWhat to do: check file permissions.", filepath.Base(path), err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to %s: %w", filepath.Base(path), err)
	}

	if strings.ToLower(topic) == "memory" {
		lineCount, _ := MemoryLineCount()
		if lineCount > 200 {
			fmt.Fprintf(os.Stderr, "Warning: MEMORY.md is %d lines (recommended: under 200).\nWhat to do: move detailed entries to topic files (gotchas.md, patterns.md, etc.).\n", lineCount)
		}
	}

	return nil
}

func (h *Hub) GetAll() (string, error) {
	var result strings.Builder
	for _, topic := range AvailableTopics() {
		content, err := h.Get(topic)
		if err != nil {
			return "", err
		}
		result.WriteString(fmt.Sprintf("## %s\n\n%s\n---\n\n", strings.ToUpper(topic), content))
	}
	return result.String(), nil
}

func (h *Hub) Add(topic string, message string) error {
	path, err := h.topicFilePath(topic)
	if err != nil {
		return err
	}

	normalizedMsg := normalizeEntry(message)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	if data != nil {
		for _, line := range strings.Split(string(data), "\n") {
			msg := extractMessageFromEntry(line)
			if normalizeEntry(msg) == normalizedMsg && normalizedMsg != "" {
				return nil
			}
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n### [%s] %s\n\n", timestamp, message)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w.\nWhat to do: check file permissions.", filepath.Base(path), err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to %s: %w", filepath.Base(path), err)
	}

	if strings.ToLower(topic) == "memory" {
		lineCount, _ := h.MemoryLineCount()
		if lineCount > 200 {
			fmt.Fprintf(os.Stderr, "Warning: MEMORY.md is %d lines (recommended: under 200).\nWhat to do: move detailed entries to topic files.\n", lineCount)
		}
	}

	return nil
}

type TopicSummary struct {
	Name          string `json:"name"`
	EntryCount    int    `json:"entry_count"`
	LineCount     int    `json:"line_count"`
	HasDuplicates bool   `json:"has_duplicates"`
}

type TopicEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

func GetTopicEntries(name string) ([]TopicEntry, error) {
	path, err := TopicFilePath(name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}

	content := string(data)
	var entries []TopicEntry
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### [") {
			endIdx := strings.Index(line, "] ")
			if endIdx > 0 {
				timestamp := line[5:endIdx]
				message := strings.TrimSpace(line[endIdx+2:])
				if message != "" {
					entries = append(entries, TopicEntry{
						Timestamp: timestamp,
						Message:   message,
					})
				}
			}
		}
	}
	return entries, nil
}

func (h *Hub) GetSummary(name string) (TopicSummary, error) {
	path, err := h.topicFilePath(name)
	if err != nil {
		return TopicSummary{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return TopicSummary{}, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}

	content := string(data)
	entryCount := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### [") {
			entryCount++
		}
	}

	return TopicSummary{
		Name:          name,
		EntryCount:    entryCount,
		LineCount:     len(strings.Split(content, "\n")),
		HasDuplicates: detectDuplicates(content),
	}, nil
}

func GetTopicSummary(name string) (TopicSummary, error) {
	path, err := TopicFilePath(name)
	if err != nil {
		return TopicSummary{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return TopicSummary{}, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}

	content := string(data)
	entryCount := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### [") {
			entryCount++
		}
	}

	return TopicSummary{
		Name:          name,
		EntryCount:    entryCount,
		LineCount:     len(strings.Split(content, "\n")),
		HasDuplicates: detectDuplicates(content),
	}, nil
}

func GetAllSummaries() ([]TopicSummary, error) {
	var summaries []TopicSummary
	for _, topic := range AvailableTopics() {
		summary, err := GetTopicSummary(topic)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (h *Hub) GetAllSummaries() ([]TopicSummary, error) {
	var summaries []TopicSummary
	for _, topic := range AvailableTopics() {
		summary, err := h.GetSummary(topic)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func GetAllTopicsWithSummary() (string, error) {
	summaries, err := GetAllSummaries()
	if err != nil {
		return "", err
	}

	var result strings.Builder

	result.WriteString("# PROJECT MEMORY SUMMARY\n\n")
	result.WriteString("## Overview\n\n")
	for _, s := range summaries {
		result.WriteString(fmt.Sprintf("- **%s**: %d entries, %d lines", s.Name, s.EntryCount, s.LineCount))
		if s.HasDuplicates {
			result.WriteString(" (duplicates detected)")
		}
		result.WriteString("\n")
	}
	result.WriteString("\n---\n\n")

	for _, topic := range AvailableTopics() {
		content, err := GetTopic(topic)
		if err != nil {
			return "", err
		}
		deduped := deduplicateContent(content)
		result.WriteString(fmt.Sprintf("## %s\n\n%s\n---\n\n", strings.ToUpper(topic), deduped))
	}
	return result.String(), nil
}

func (h *Hub) GetAllWithSummary() (string, error) {
	summaries, err := h.GetAllSummaries()
	if err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString("# PROJECT MEMORY SUMMARY\n\n")
	result.WriteString("## Overview\n\n")
	for _, s := range summaries {
		result.WriteString(fmt.Sprintf("- **%s**: %d entries, %d lines", s.Name, s.EntryCount, s.LineCount))
		if s.HasDuplicates {
			result.WriteString(" (duplicates detected)")
		}
		result.WriteString("\n")
	}
	result.WriteString("\n---\n\n")

	for _, topic := range AvailableTopics() {
		content, err := h.Get(topic)
		if err != nil {
			return "", err
		}
		deduped := deduplicateContent(content)
		result.WriteString(fmt.Sprintf("## %s\n\n%s\n---\n\n", strings.ToUpper(topic), deduped))
	}
	return result.String(), nil
}

func (h *Hub) MemoryLineCount() (int, error) {
	path := filepath.Join(h.dir, "MEMORY.md")
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func MemoryLineCount() (int, error) {
	brainDir, err := FindBrainDir()
	if err != nil {
		return 0, err
	}
	path := filepath.Join(brainDir, "MEMORY.md")
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func normalizeEntry(message string) string {
	re := strings.NewReplacer("\n", " ", "\r", "", "  ", " ")
	return strings.ToLower(strings.TrimSpace(re.Replace(message)))
}

func extractMessageFromEntry(line string) string {
	if strings.HasPrefix(line, "### [") {
		if idx := strings.Index(line, "] "); idx > 0 {
			return line[idx+2:]
		}
	}
	return line
}

func detectDuplicates(content string) bool {
	seen := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### [") {
			normalized := strings.ToLower(strings.TrimSpace(line))
			if seen[normalized] {
				return true
			}
			seen[normalized] = true
		}
	}
	return false
}

func deduplicateContent(content string) string {
	seen := make(map[string]bool)
	var result []string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "### [") {
			normalized := strings.ToLower(strings.TrimSpace(line))
			if seen[normalized] {
				continue
			}
			seen[normalized] = true
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

const defaultDuplicateThreshold = 0.55

func IsDuplicateOfExisting(topicFilePath string, content string) (bool, error) {
	data, err := os.ReadFile(topicFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	normalized := normalizeEntry(content)
	if normalized == "" {
		return false, nil
	}

	for _, line := range strings.Split(string(data), "\n") {
		msg := extractMessageFromEntry(line)
		existing := normalizeEntry(msg)
		if existing == "" {
			continue
		}
		if existing == normalized {
			return true, nil
		}
		if trigramJaccard(existing, normalized) >= defaultDuplicateThreshold {
			return true, nil
		}
	}

	return false, nil
}

var entryPattern = regexp.MustCompile(`^### \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] (.+)$`)

func ExtractTopicEntries(topicFile string) ([]PendingEntry, error) {
	data, err := os.ReadFile(topicFile)
	if err != nil {
		return nil, fmt.Errorf("reading topic file: %w", err)
	}

	var entries []PendingEntry
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var currentContent strings.Builder
	var currentTimestamp string

	for scanner.Scan() {
		line := scanner.Text()

		matches := entryPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			if currentContent.Len() > 0 && currentTimestamp != "" {
				content := strings.TrimSpace(currentContent.String())
				if content != "" {
					entries = append(entries, PendingEntry{Content: content})
				}
			}
			currentTimestamp = matches[1]
			currentContent.Reset()
			currentContent.WriteString(matches[2])
		} else if currentTimestamp != "" && line != "" {
			currentContent.WriteString(" ")
			currentContent.WriteString(line)
		}
	}

	if currentContent.Len() > 0 && currentTimestamp != "" {
		content := strings.TrimSpace(currentContent.String())
		if content != "" {
			entries = append(entries, PendingEntry{Content: content})
		}
	}

	return entries, scanner.Err()
}

func TopicEntriesToPending(topicName, topicFile, pendingDir string) (int, error) {
	existing, err := LoadPendingEntries(pendingDir)
	if err != nil {
		return 0, err
	}
	existingFPs := make(map[string]bool)
	for _, e := range existing {
		existingFPs[e.Fingerprint()] = true
	}

	entries, err := ExtractTopicEntries(topicFile)
	if err != nil {
		return 0, err
	}

	added := 0
	for _, e := range entries {
		pe := PendingEntry{
			ID:         fmt.Sprintf("import-%s-%d", topicName, added),
			Topic:      topicName,
			Content:    e.Content,
			CommitSHA:  "",
			Timestamp:  time.Now(),
			Confidence: "MEDIUM",
			Source:     "import",
		}
		fp := pe.Fingerprint()
		if existingFPs[fp] {
			continue
		}
		if err := SavePendingEntry(pendingDir, pe); err != nil {
			return added, err
		}
		existingFPs[fp] = true
		added++
	}

	return added, nil
}
