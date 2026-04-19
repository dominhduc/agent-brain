package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func ConsolidateCluster(entries []TopicEntry) string {
	if len(entries) == 0 {
		return ""
	}

	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].Message) > len(entries[j].Message)
	})

	base := entries[0].Message
	baseSentences := extractSentences(base)

	var additions []string
	addedNormalized := make(map[string]bool)
	for _, s := range baseSentences {
		addedNormalized[normalizeTextForCluster(s)] = true
	}
	for i := 1; i < len(entries); i++ {
		sentences := extractSentences(entries[i].Message)
		for _, s := range sentences {
			norm := normalizeTextForCluster(s)
			if addedNormalized[norm] {
				continue
			}
			if !isSentenceDuplicate(s, baseSentences) {
				additions = append(additions, s)
				addedNormalized[norm] = true
			}
		}
	}

	if len(additions) == 0 {
		return base
	}

	return base + " " + strings.Join(additions, " ")
}

func extractSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
			continue
		}
	}

	if current.Len() > 0 {
		s := strings.TrimSpace(current.String())
		if s != "" {
			sentences = append(sentences, s)
		}
	}

	return sentences
}

func isSentenceDuplicate(sentence string, existing []string) bool {
	normalized := normalizeTextForCluster(sentence)
	for _, e := range existing {
		if normalizeTextForCluster(e) == normalized {
			return true
		}
		sim := trigramJaccard(normalizeTextForCluster(e), normalized)
		if sim >= 0.6 {
			return true
		}
	}
	return false
}

type ConsolidationProposal struct {
	Topic     string
	Sources   []ConsolidationSource
	Consolidated string
	ID        string
}

type ConsolidationSource struct {
	Timestamp string
	Message   string
	Strength  float64
}

func (h *Hub) FindConsolidations() ([]ConsolidationProposal, error) {
	var proposals []ConsolidationProposal

	for _, topic := range AvailableTopics() {
		entries, err := GetTopicEntriesForDir(topic, h.dir)
		if err != nil {
			continue
		}

		clusters := ClusterEntries(entries, topic)
		for _, cluster := range clusters {
			var sources []ConsolidationSource
			var clusterEntries []TopicEntry

			index, _ := h.LoadIndex()
			for _, entryIdx := range cluster.MemberIndices {
				if entryIdx < 0 || entryIdx >= len(entries) {
					continue
				}
				e := entries[entryIdx]
				var strength float64 = 1.0
				if index != nil {
					if ie, found := index.Get(topic, e.Timestamp); found {
						strength = ie.Strength
					}
				}
				sources = append(sources, ConsolidationSource{
					Timestamp: e.Timestamp,
					Message:   e.Message,
					Strength:  strength,
				})
				clusterEntries = append(clusterEntries, TopicEntry{
					Timestamp: e.Timestamp,
					Message:   e.Message,
				})
			}

			consolidated := ConsolidateCluster(clusterEntries)
			if consolidated == "" {
				continue
			}

			proposals = append(proposals, ConsolidationProposal{
				Topic:        topic,
				Sources:      sources,
				Consolidated: consolidated,
				ID:           fmt.Sprintf("%s-%s", topic, cluster.Representative),
			})
		}
	}

	return proposals, nil
}

func (h *Hub) ApplyConsolidation(proposal ConsolidationProposal) error {
	topicPath, err := TopicFilePathForDir(proposal.Topic, h.dir)
	if err != nil {
		return fmt.Errorf("failed to get topic path: %w", err)
	}

	data, err := os.ReadFile(topicPath)
	if err != nil {
		return fmt.Errorf("failed to read topic file: %w", err)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	newEntry := fmt.Sprintf("\n### [%s] %s\n", now, proposal.Consolidated)

	var timeline strings.Builder
	timeline.WriteString("<!-- Source timeline:\n")
	for _, s := range proposal.Sources {
		timeline.WriteString(fmt.Sprintf("     %s: %s\n", s.Timestamp, s.Message))
	}
	timeline.WriteString(" -->\n")

	lines := strings.Split(string(data), "\n")
	var newLines []string
	var modified bool
	archived := make(map[string]bool)
	for _, ts := range proposal.Sources {
		archived[ts.Timestamp] = true
	}

	for _, line := range lines {
		matches := entryHeaderRe.FindStringSubmatch(line)
		if matches != nil {
			ts := matches[1]
			if archived[ts] {
				if !modified {
					newLines = append(newLines, newEntry)
					newLines = append(newLines, timeline.String())
					modified = true
				}
				continue
			}
		}
		newLines = append(newLines, line)
	}

	if !modified {
		newLines = append(newLines, newEntry)
		newLines = append(newLines, timeline.String())
	}

	if err := os.WriteFile(topicPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		return fmt.Errorf("failed to write topic file: %w", err)
	}

	idx, err := h.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	newKey := MakeKey(proposal.Topic, now)
	var maxStrength float64
	for _, s := range proposal.Sources {
		if s.Strength > maxStrength {
			maxStrength = s.Strength
		}
		if ie, found := idx.Get(proposal.Topic, s.Timestamp); found {
			ie.Confidence = "superseded"
			ie.Strength = 0
			ie.SupersededBy = newKey
			idx.Set(proposal.Topic, s.Timestamp, ie)
		}
	}

	idx.Entries[newKey] = IndexEntry{
		Strength:       maxStrength + 0.1,
		RetrievalCount: 0,
		LastRetrieved:  time.Now(),
		HalfLifeDays:   14,
		Confidence:     "verified",
		Topics:         DetectTopics(proposal.Consolidated),
		Version:        1,
	}

	return idx.Save(h.dir)
}

func (h *Hub) SaveConsolidationPending(proposals []ConsolidationProposal) error {
	pendingDir := filepath.Join(h.dir, "pending")
	if err := os.MkdirAll(pendingDir, 0700); err != nil {
		return err
	}

	for _, p := range proposals {
		pe := PendingEntry{
			ID:         "consolidation-" + p.ID,
			Topic:      p.Topic,
			Content:    p.Consolidated,
			Timestamp:  time.Now(),
			Confidence: "MEDIUM",
			Source:     "consolidation",
		}
		if err := SavePendingEntry(pendingDir, pe); err != nil {
			return err
		}
	}
	return nil
}
