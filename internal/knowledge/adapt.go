package knowledge

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type AdaptationSection struct {
	TopGotchas   []string
	TopicInsights []string
	Adjustments  []string
}

func (h *Hub) GenerateAdaptation() (*AdaptationSection, error) {
	signals, err := h.LoadBehavior()
	if err != nil {
		return nil, err
	}

	idx, err := h.LoadIndex()
	if err != nil {
		return nil, err
	}
	now := time.Now()

	adaptation := &AdaptationSection{}

	adaptation.TopGotchas = h.topGotchasFromIndex(idx, now, 5)

	adaptation.TopicInsights = h.topicInsightsFromBehavior(signals)

	adaptation.Adjustments = h.adjustmentsFromData(signals, idx, now)

	return adaptation, nil
}

func (h *Hub) FormatAdaptation(adaptation *AdaptationSection) string {
	var b strings.Builder

	if len(adaptation.TopGotchas) > 0 {
		b.WriteString("### Top Gotchas\n")
		for _, g := range adaptation.TopGotchas {
			b.WriteString(fmt.Sprintf("- %s\n", g))
		}
		b.WriteString("\n")
	}

	if len(adaptation.TopicInsights) > 0 {
		b.WriteString("### Workflow Insights\n")
		for _, insight := range adaptation.TopicInsights {
			b.WriteString(fmt.Sprintf("- %s\n", insight))
		}
		b.WriteString("\n")
	}

	if len(adaptation.Adjustments) > 0 {
		b.WriteString("### Suggested Adjustments\n")
		for _, adj := range adaptation.Adjustments {
			b.WriteString(fmt.Sprintf("- %s\n", adj))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (h *Hub) topGotchasFromIndex(idx *Index, now time.Time, n int) []string {
	type scored struct {
		key     string
		entry   IndexEntry
		content string
	}

	var entries []scored
	for key, entry := range idx.Entries {
		if !strings.HasPrefix(key, "gotchas:") {
			continue
		}
		strength := CalculateStrength(entry, now)
		if strength < 0.5 || entry.RetrievalCount < 2 {
			continue
		}
		msg := h.extractEntryMessage(key)
		if msg == "" {
			continue
		}
		entries = append(entries, scored{key: key, entry: entry, content: msg})
	}

	sort.Slice(entries, func(i, j int) bool {
		si := CalculateStrength(entries[i].entry, now)
		sj := CalculateStrength(entries[j].entry, now)
		return si > sj
	})

	if len(entries) > n {
		entries = entries[:n]
	}

	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = fmt.Sprintf("%s (retrieved %dx, strength %.2f)", e.content, e.entry.RetrievalCount, CalculateStrength(e.entry, now))
	}
	return result
}

func (h *Hub) topicInsightsFromBehavior(signals *BehaviorSignals) []string {
	if len(signals.TopicAccess) == 0 {
		return nil
	}

	var insights []string

	var mostAccessed string
	var mostCount int
	var totalAccess int
	for topic, info := range signals.TopicAccess {
		totalAccess += info.Retrievals
		if info.Retrievals > mostCount {
			mostCount = info.Retrievals
			mostAccessed = topic
		}
	}

	if mostAccessed != "" && mostCount > 3 {
		insights = append(insights, fmt.Sprintf("Most used topic: %s (%d retrievals)", mostAccessed, mostCount))
	}

	for topic, info := range signals.TopicAccess {
		if info.Retrievals == 0 && !info.LastAccess.IsZero() {
			insights = append(insights, fmt.Sprintf("Topic '%s' has zero retrievals — consider if entries are still relevant", topic))
		}
	}

	if totalAccess > 20 && len(signals.TopicAccess) > 2 {
		insights = append(insights, fmt.Sprintf("Total knowledge accessed %d times across %d topics", totalAccess, len(signals.TopicAccess)))
	}

	return insights
}

func (h *Hub) adjustmentsFromData(signals *BehaviorSignals, idx *Index, now time.Time) []string {
	var adjustments []string

	totalSessions := signals.EvalOutcomes.TotalSessions
	if totalSessions > 0 {
		goodRate := float64(signals.EvalOutcomes.Good) / float64(totalSessions)
		if goodRate >= 0.8 {
			adjustments = append(adjustments, fmt.Sprintf("High success rate (%.0f%%) — current workflow is effective", goodRate*100))
		} else if goodRate < 0.5 {
			adjustments = append(adjustments, "Low success rate — review recent gotchas and adjust approach")
		}
	}

	if len(signals.SearchQueries) > 0 {
		topQuery := signals.SearchQueries[0]
		for _, q := range signals.SearchQueries[1:] {
			if q.Count > topQuery.Count {
				topQuery = q
			}
		}
		if topQuery.Count > 3 {
			adjustments = append(adjustments, fmt.Sprintf("Most searched: '%s' (%dx) — consider adding a dedicated pattern entry", topQuery.Query, topQuery.Count))
		}
	}

	strongCount := 0
	weakCount := 0
	for _, entry := range idx.Entries {
		s := CalculateStrength(entry, now)
		if s > 0.7 {
			strongCount++
		} else if s < 0.3 {
			weakCount++
		}
	}
	if weakCount > strongCount && weakCount > 5 {
		adjustments = append(adjustments, fmt.Sprintf("%d weak entries vs %d strong — run 'brain clean --decay' to clean up", weakCount, strongCount))
	}

	return adjustments
}

func (h *Hub) extractEntryMessage(key string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	topic := parts[0]
	timestamp := parts[1]

	path, err := h.topicFilePath(topic)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "### [") {
			idx := strings.Index(line, "] ")
			if idx > 0 {
				ts := line[5:idx]
				if ts == timestamp {
					return line[idx+2:]
				}
			}
		}
	}
	return ""
}
