package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GlobalBrainDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	dir := filepath.Join(home, ".brain", "global")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create global brain dir: %w", err)
	}
	return dir, nil
}

func GlobalTopicFilePath(topic string) (string, error) {
	globalDir, err := GlobalBrainDir()
	if err != nil {
		return "", err
	}
	return TopicFilePathForDir(topic, globalDir)
}

func AddGlobalEntry(topic, message string) (bool, error) {
	globalDir, err := GlobalBrainDir()
	if err != nil {
		return false, err
	}

	path, err := TopicFilePathForDir(topic, globalDir)
	if err != nil {
		return false, err
	}

	normalizedMsg := normalizeEntry(message)
	if normalizedMsg == "" {
		return false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	if data != nil {
		existing := string(data)
		lines := strings.Split(existing, "\n")
		for _, line := range lines {
			msg := extractMessageFromEntry(line)
			lineNormalized := normalizeEntry(msg)
			if lineNormalized == normalizedMsg && lineNormalized != "" {
				return true, nil
			}
		}
		for _, line := range lines {
			msg := extractMessageFromEntry(line)
			lineNormalized := normalizeEntry(msg)
			if lineNormalized != "" && trigramJaccard(lineNormalized, normalizedMsg) >= defaultDuplicateThreshold {
				return true, nil
			}
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("\n### [%s] %s\n\n", timestamp, message)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return false, fmt.Errorf("failed to open %s: %w", filepath.Base(path), err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return false, fmt.Errorf("failed to write to %s: %w", filepath.Base(path), err)
	}

	return false, nil
}

func GetGlobalTopicEntries(topic string) ([]TopicEntry, error) {
	globalDir, err := GlobalBrainDir()
	if err != nil {
		return nil, err
	}
	return GetTopicEntriesForDir(topic, globalDir)
}

func GlobalIndex() (*Index, error) {
	globalDir, err := GlobalBrainDir()
	if err != nil {
		return nil, err
	}
	return LoadIndex(globalDir)
}

func SaveGlobalIndex(idx *Index) error {
	globalDir, err := GlobalBrainDir()
	if err != nil {
		return err
	}
	return idx.Save(globalDir)
}
