package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetTopic(name string) (string, error) {
	path, err := TopicFilePath(name)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}

	return string(data), nil
}

func GetAllTopics() (string, error) {
	topics := []string{"memory", "gotchas", "patterns", "decisions", "architecture"}
	var result strings.Builder

	for _, topic := range topics {
		content, err := GetTopic(topic)
		if err != nil {
			return "", err
		}

		result.WriteString(fmt.Sprintf("## %s\n\n", strings.ToUpper(topic)))
		result.WriteString(content)
		result.WriteString("\n---\n\n")
	}

	return result.String(), nil
}

func AddEntry(topic string, message string) error {
	path, err := TopicFilePath(topic)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n### [%s] %s\n\n", timestamp, message)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filepath.Base(path), err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to %s: %w", filepath.Base(path), err)
	}

	// Warn if MEMORY.md exceeds 200 lines
	if strings.ToLower(topic) == "memory" {
		lineCount, err := MemoryLineCount()
		if err == nil && lineCount > 200 {
			fmt.Fprintf(os.Stderr, "Warning: MEMORY.md is %d lines (recommended: under 200). Consider moving details to topic files.\n", lineCount)
		}
	}

	return nil
}

func MemoryLineCount() (int, error) {
	brainDir, err := FindBrainDir()
	if err != nil {
		return 0, err
	}

	path := filepath.Join(brainDir, "MEMORY.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	return strings.Count(string(data), "\n") + 1, nil
}

func AvailableTopics() []string {
	return []string{"memory", "gotchas", "patterns", "decisions", "architecture"}
}
