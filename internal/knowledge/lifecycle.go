package knowledge

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (h *Hub) UpdateEntry(topic, timestampPrefix, newMessage string) error {
	idx, err := h.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	var matchingKey string
	for key := range idx.Entries {
		if strings.HasPrefix(key, topic+":"+timestampPrefix) {
			matchingKey = key
			break
		}
	}
	if matchingKey == "" {
		return fmt.Errorf("no entry found matching %q with timestamp prefix %q", topic, timestampPrefix)
	}

	topicPath, err := TopicFilePathForDir(topic, h.dir)
	if err != nil {
		return fmt.Errorf("failed to get topic file path: %w", err)
	}

	data, err := os.ReadFile(topicPath)
	if err != nil {
		return fmt.Errorf("failed to read topic file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var entryStart, entryEnd int = -1, -1
	var oldMessage strings.Builder
	found := false

	for i, line := range lines {
		matches := entryHeaderRe.FindStringSubmatch(line)
		if matches != nil {
			ts := matches[1]
			key := topic + ":" + ts
			if key == matchingKey {
				entryStart = i
				found = true
				continue
			}
			if found && entryEnd == -1 {
				entryEnd = i
				break
			}
		}
		if found && entryStart >= 0 && entryEnd == -1 {
			if oldMessage.Len() > 0 {
				oldMessage.WriteString("\n")
			}
			oldMessage.WriteString(line)
		}
	}
	if found && entryEnd == -1 {
		entryEnd = len(lines)
	}

	if entryStart < 0 {
		return fmt.Errorf("entry not found in topic file")
	}

	h.archiveEntryVersion(matchingKey, topic, oldMessage.String())

	timestamp := matchingKey[len(topic)+1:]
	newEntry := fmt.Sprintf("### [%s] %s", timestamp, newMessage)

	newLines := make([]string, 0, entryStart+1+len(lines)-entryEnd)
	newLines = append(newLines, lines[:entryStart]...)
	newLines = append(newLines, newEntry)
	newLines = append(newLines, lines[entryEnd:]...)

	if err := os.WriteFile(topicPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		return fmt.Errorf("failed to write topic file: %w", err)
	}

	idxEntry := idx.Entries[matchingKey]
	idxEntry.Version++
	idxEntry.UpdatedAt = time.Now()
	idxEntry.Confidence = "verified"
	idx.SetByRawKey(matchingKey, idxEntry)

	if err := idx.Save(h.dir); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

func (h *Hub) archiveEntryVersion(key, topic, content string) {
	archiveDir := filepath.Join(h.dir, "archived", "versions")
	if err := os.MkdirAll(archiveDir, 0700); err != nil {
		return
	}
	fingerprint := contentFingerprint(content)
	path := filepath.Join(archiveDir, fmt.Sprintf("%s-%s.md", topic, fingerprint))
	os.WriteFile(path, []byte(content), 0600)
}

func (h *Hub) SupersedeEntry(topic, oldTS, newTS string) error {
	idx, err := h.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	oldKey := MakeKey(topic, oldTS)
	newKey := MakeKey(topic, newTS)

	oldEntry, oldFound := idx.GetByRawKey(oldKey)
	newEntry, newFound := idx.GetByRawKey(newKey)

	if !oldFound {
		return fmt.Errorf("old entry not found: %s", oldKey)
	}
	if !newFound {
		return fmt.Errorf("new entry not found: %s", newKey)
	}

	topicPath, err := TopicFilePathForDir(topic, h.dir)
	if err != nil {
		return fmt.Errorf("failed to get topic path: %w", err)
	}

	data, err := os.ReadFile(topicPath)
	if err != nil {
		return fmt.Errorf("failed to read topic file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		matches := entryHeaderRe.FindStringSubmatch(line)
		if matches != nil {
			ts := matches[1]
			if ts == oldTS {
				msg := strings.TrimPrefix(line, matches[0])
				lines[i] = fmt.Sprintf("### [%s] ~~%s~~ (superseded)", ts, strings.TrimSpace(msg))
				break
			}
		}
	}

	if err := os.WriteFile(topicPath, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		return fmt.Errorf("failed to write topic file: %w", err)
	}

	oldEntry.Confidence = "superseded"
	oldEntry.Strength = 0
	oldEntry.SupersededBy = newKey
	idx.SetByRawKey(oldKey, oldEntry)

	newEntry.Supersedes = oldKey
	newEntry.Version++
	idx.SetByRawKey(newKey, newEntry)

	if err := idx.Save(h.dir); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

func (h *Hub) FindConflicts() ([]ConflictPair, error) {
	idx, err := h.LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	var conflicts []ConflictPair
	entries := make(map[string]IndexEntry)
	messages := make(map[string]string)

	for _, topic := range AvailableTopics() {
		topicEntries, err := GetTopicEntriesForDir(topic, h.dir)
		if err != nil {
			continue
		}
		for _, e := range topicEntries {
			key := MakeKey(topic, e.Timestamp)
			entries[key] = idx.Entries[key]
			messages[key] = e.Message
		}
	}

	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			k1, k2 := keys[i], keys[j]
			if normalizeEntry(messages[k1]) == "" || normalizeEntry(messages[k2]) == "" {
				continue
			}
			sim := trigramJaccard(normalizeEntry(messages[k1]), normalizeEntry(messages[k2]))
			if sim > 0.4 && hasOpposingSentiment(messages[k1], messages[k2]) {
				conflicts = append(conflicts, ConflictPair{
					Key1: k1,
					Key2: k2,
				})
			}
		}
	}

	return conflicts, nil
}

type ConflictPair struct {
	Key1 string
	Key2 string
}

func hasOpposingSentiment(a, b string) bool {
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)

	negationWords := []string{"don't", "never", "avoid", "not ", "no ", "without", "disable", "remove"}
	positivePatterns := []string{"always", "use ", "do ", "enable", "add "}

	aNegated := false
	bNegated := false
	for _, w := range negationWords {
		if strings.Contains(aLower, w) {
			aNegated = true
			break
		}
	}
	for _, w := range negationWords {
		if strings.Contains(bLower, w) {
			bNegated = true
			break
		}
	}

	aPositive := false
	bPositive := false
	for _, p := range positivePatterns {
		if strings.Contains(aLower, p) {
			aPositive = true
			break
		}
	}
	for _, p := range positivePatterns {
		if strings.Contains(bLower, p) {
			bPositive = true
			break
		}
	}

	return (aNegated && bPositive) || (bNegated && aPositive)
}

func GetTopicEntriesForDir(topic, brainDir string) ([]TopicEntry, error) {
	path, err := TopicFilePathForDir(topic, brainDir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	return parseEntriesFromContent(string(data))
}

func parseEntriesFromContent(content string) ([]TopicEntry, error) {
	var entries []TopicEntry
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "### [") {
			endIdx := strings.Index(line, "] ")
			if endIdx > 0 {
				timestamp := line[5:endIdx]
				message := strings.TrimSpace(line[endIdx+2:])
				if !strings.HasPrefix(message, "~~") {
					if message != "" {
						entries = append(entries, TopicEntry{
							Timestamp: timestamp,
							Message:   message,
						})
					}
				}
			}
		}
	}
	return entries, nil
}
