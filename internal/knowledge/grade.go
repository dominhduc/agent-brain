package knowledge

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Grade struct {
	Key         string  `json:"key"`
	Accuracy    float64 `json:"accuracy"`
	Specificity float64 `json:"specificity"`
	Generality  float64 `json:"generality"`
	Verdict     string  `json:"verdict"`
	Reason      string  `json:"reason"`
}

type GradeReport struct {
	Grades       []Grade `json:"grades"`
	KeepCount    int     `json:"keep_count"`
	RewriteCount int     `json:"rewrite_count"`
	ArchiveCount int     `json:"archive_count"`
}

type gradeEntry struct {
	key     string
	message string
	topic   string
	strength float64
}

func (h *Hub) GradeCandidates() ([]gradeEntry, error) {
	idx, err := h.LoadIndex()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	topicEntriesCache := make(map[string][]TopicEntry)

	var candidates []gradeEntry
	for key, entry := range idx.Entries {
		if entry.Confidence == "superseded" {
			continue
		}
		topicEnd := strings.Index(key, ":")
		if topicEnd < 0 {
			continue
		}
		topic := key[:topicEnd]
		timestamp := key[topicEnd+1:]

		if _, ok := topicEntriesCache[topic]; !ok {
			entries, err := GetTopicEntriesForDir(topic, h.dir)
			if err != nil {
				continue
			}
			topicEntriesCache[topic] = entries
		}

		var message string
		for _, e := range topicEntriesCache[topic] {
			if e.Timestamp == timestamp {
				message = e.Message
				break
			}
		}
		if message == "" {
			continue
		}

		candidates = append(candidates, gradeEntry{
			key:      key,
			message:  message,
			topic:    topic,
			strength: CalculateStrength(entry, now),
		})
	}
	return candidates, nil
}

func BuildGradingPrompt(entries []gradeEntry) string {
	var sb strings.Builder
	sb.WriteString("You are grading knowledge base entries for accuracy and usefulness.\n")
	sb.WriteString("For each entry, respond with ONLY a JSON array:\n\n")
	sb.WriteString("[{\n")
	sb.WriteString("  \"key\": \"<exact key>\",\n")
	sb.WriteString("  \"accuracy\": 0.0-1.0,\n")
	sb.WriteString("  \"specificity\": 0.0-1.0,\n")
	sb.WriteString("  \"generality\": 0.0-1.0,\n")
	sb.WriteString("  \"verdict\": \"keep|rewrite|archive\",\n")
	sb.WriteString("  \"reason\": \"one sentence\"\n")
	sb.WriteString("}]\n\n")
	sb.WriteString("Scoring guide:\n")
	sb.WriteString("- accuracy: Is this still true given current best practices? (0.0 = definitely wrong, 1.0 = definitely true)\n")
	sb.WriteString("- specificity: Is this specific enough that someone could act on it? (0.0 = too vague, 1.0 = crystal clear)\n")
	sb.WriteString("- generality: Is this reusable across projects? (0.0 = only applies to one codebase, 1.0 = universal principle)\n")
	sb.WriteString("- verdict: \"archive\" if accuracy < 0.3, \"rewrite\" if specificity < 0.5, otherwise \"keep\"\n\n")

	type jsonEntry struct {
		Key     string  `json:"key"`
		Topic   string  `json:"topic"`
		Message string  `json:"message"`
		Strength float64 `json:"strength"`
	}
	var jsonEntries []jsonEntry
	for _, e := range entries {
		jsonEntries = append(jsonEntries, jsonEntry{
			Key:     e.key,
			Topic:   e.topic,
			Message: e.message,
			Strength: e.strength,
		})
	}
	data, err := json.MarshalIndent(jsonEntries, "", "  ")
	if err != nil {
		return ""
	}
	sb.WriteString("Entries:\n")
	sb.WriteString(string(data))
	return sb.String()
}

func ParseGradingResponse(content string) ([]Grade, error) {
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")
	if jsonStart < 0 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON array found in grading response")
	}
	var grades []Grade
	if err := json.Unmarshal([]byte(content[jsonStart:jsonEnd+1]), &grades); err != nil {
		return nil, fmt.Errorf("failed to parse grades: %w", err)
	}
	validVerdicts := map[string]bool{"keep": true, "rewrite": true, "archive": true}
	var validated []Grade
	for _, g := range grades {
		if !validVerdicts[g.Verdict] {
			g.Verdict = "keep"
		}
		if g.Accuracy < 0 {
			g.Accuracy = 0
		}
		if g.Accuracy > 1 {
			g.Accuracy = 1
		}
		if g.Specificity < 0 {
			g.Specificity = 0
		}
		if g.Specificity > 1 {
			g.Specificity = 1
		}
		if g.Generality < 0 {
			g.Generality = 0
		}
		if g.Generality > 1 {
			g.Generality = 1
		}
		if g.Key == "" {
			continue
		}
		validated = append(validated, g)
	}
	return validated, nil
}

func (h *Hub) ApplyGrades(grades []Grade, dryRun bool) (*GradeReport, error) {
	report := &GradeReport{}

	idx, err := h.LoadIndex()
	if err != nil {
		return nil, err
	}

	for _, g := range grades {
		report.Grades = append(report.Grades, g)
		switch g.Verdict {
		case "keep":
			report.KeepCount++
		case "rewrite":
			report.RewriteCount++
		case "archive":
			report.ArchiveCount++
			if !dryRun {
				if entry, ok := idx.Entries[g.Key]; ok {
					entry.Confidence = "stale"
					entry.Strength = 0
					idx.Entries[g.Key] = entry
				}
			}
		}
	}

	if !dryRun {
		if err := idx.Save(h.dir); err != nil {
			return nil, fmt.Errorf("failed to save index after grading: %w", err)
		}
	}

	return report, nil
}
