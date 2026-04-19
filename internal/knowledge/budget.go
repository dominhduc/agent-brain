package knowledge

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type RetrievalBudget struct {
	MaxTokens         int
	MinStrength       float64
	MaxEntries        int
	IncludeRecentDays int
	GotchasPct        float64
	PatternsPct       float64
	DecisionsPct      float64
	ArchitecturePct   float64
	RemainingPct      float64
}

func DefaultBudget() RetrievalBudget {
	return RetrievalBudget{
		MaxTokens:         3000,
		MinStrength:       0.15,
		MaxEntries:        50,
		IncludeRecentDays: 7,
		GotchasPct:        0.30,
		PatternsPct:       0.25,
		DecisionsPct:      0.20,
		ArchitecturePct:   0.15,
		RemainingPct:      0.10,
	}
}

type ScoredEntry struct {
	Key       string
	Topic     string
	Message   string
	Score     float64
	Tokens    int
	Timestamp string
}

type BudgetResult struct {
	Entries     []ScoredEntry
	TotalTokens int
	BudgetUsed  float64
	Skipped     int
	SkippedLow  int
}

func (h *Hub) RetrieveWithBudget(budget RetrievalBudget, contextTopics []string) (*BudgetResult, error) {
	entries, err := h.loadAllEntries()
	if err != nil {
		return nil, err
	}

	idx, err := h.LoadIndex()
	if err != nil {
		idx = &Index{Entries: make(map[string]IndexEntry)}
	}

	now := time.Now()
	contextSet := make(map[string]bool)
	for _, t := range contextTopics {
		contextSet[t] = true
	}

	var scored []ScoredEntry
	for _, e := range entries {
		idxEntry, found := idx.GetByRawKey(e.Key)
		if found && idxEntry.Confidence == "superseded" {
			continue
		}

		var strength float64
		if found {
			strength = CalculateStrength(idxEntry, now)
		} else {
			strength = 1.0
		}

		if strength < budget.MinStrength {
			continue
		}

		recencyBoost := 1.0
		t, _ := time.Parse("2006-01-02 15:04:05", e.Timestamp)
		ageHours := now.Sub(t).Hours()
		if ageHours < 24 {
			recencyBoost = 2.0
		} else if ageHours < 24*float64(budget.IncludeRecentDays) {
			recencyBoost = 1.5
		}

		contextAffinity := 1.0
		if len(contextSet) > 0 {
			entryTopics := idxEntry.Topics
			if len(entryTopics) == 0 {
				entryTopics = DetectTopics(e.Message)
			}
			for _, et := range entryTopics {
				if contextSet[et] {
					contextAffinity = 1.5
					break
				}
			}
		}

		versionBoost := 1.0
		if found && idxEntry.Version > 1 {
			versionBoost = 1.2
		}

		combinedScore := strength * recencyBoost * contextAffinity * versionBoost
		tokens := EstimateTokens(e.Message)

		scored = append(scored, ScoredEntry{
			Key:       e.Key,
			Topic:     e.Topic,
			Message:   e.Message,
			Score:     combinedScore,
			Tokens:    tokens,
			Timestamp: e.Timestamp,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	result := &BudgetResult{}
	used := 0
	skippedLow := 0
	totalCandidates := len(scored)

	topicMaxTokens := map[string]int{
		"gotchas":      int(float64(budget.MaxTokens) * budget.GotchasPct),
		"patterns":     int(float64(budget.MaxTokens) * budget.PatternsPct),
		"decisions":    int(float64(budget.MaxTokens) * budget.DecisionsPct),
		"architecture": int(float64(budget.MaxTokens) * budget.ArchitecturePct),
	}
	topicUsed := make(map[string]int)
	topicCounts := make(map[string]int)
	topicMaxEntries := budget.MaxEntries / 4
	if topicMaxEntries < 3 {
		topicMaxEntries = 3
	}

	var leftovers []ScoredEntry
	for _, se := range scored {
		if len(result.Entries) >= budget.MaxEntries {
			leftovers = append(leftovers, se)
			continue
		}

		maxTokens, hasMax := topicMaxTokens[se.Topic]
		if hasMax {
			if topicUsed[se.Topic]+se.Tokens > maxTokens {
				leftovers = append(leftovers, se)
				continue
			}
			if topicCounts[se.Topic] >= topicMaxEntries {
				leftovers = append(leftovers, se)
				continue
			}
		}

		if used+se.Tokens > budget.MaxTokens {
			leftovers = append(leftovers, se)
			continue
		}

		topicUsed[se.Topic] += se.Tokens
		topicCounts[se.Topic]++
		used += se.Tokens
		result.Entries = append(result.Entries, se)
	}

	for _, se := range leftovers {
		if len(result.Entries) >= budget.MaxEntries {
			break
		}
		if used+se.Tokens > budget.MaxTokens {
			skippedLow++
			continue
		}
		used += se.Tokens
		topicUsed[se.Topic] += se.Tokens
		result.Entries = append(result.Entries, se)
	}

	result.TotalTokens = used
	result.BudgetUsed = float64(used) / float64(budget.MaxTokens)
	result.Skipped = skippedLow + (totalCandidates - len(result.Entries) - skippedLow)
	result.SkippedLow = skippedLow

	return result, nil
}

type rawEntry struct {
	Key       string
	Topic     string
	Message   string
	Timestamp string
}

func (h *Hub) loadAllEntries() ([]rawEntry, error) {
	var entries []rawEntry
	for _, topic := range AvailableTopics() {
		path, err := TopicFilePathForDir(topic, h.dir)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		topicEntries, err := parseTopicEntries(string(data), topic)
		if err != nil {
			continue
		}
		for _, e := range topicEntries {
			entries = append(entries, rawEntry{
				Key:       MakeKey(topic, e.Timestamp),
				Topic:     topic,
				Message:   e.Message,
				Timestamp: e.Timestamp,
			})
		}
	}
	return entries, nil
}

func parseTopicEntries(content, topic string) ([]TopicEntry, error) {
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

func (r *BudgetResult) Summary() string {
	var sb strings.Builder
	sb.WriteString("PROJECT MEMORY\n")
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	topicCounts := make(map[string]int)
	topicTokens := make(map[string]int)
	for _, e := range r.Entries {
		topicCounts[e.Topic]++
		topicTokens[e.Topic] += e.Tokens
	}

	topics := []string{"gotchas", "patterns", "decisions", "architecture", "memory"}
	for _, topic := range topics {
		if count := topicCounts[topic]; count > 0 {
			sb.WriteString(fmt.Sprintf("  %-15s %d entries shown  (%d tokens)\n", topic, count, topicTokens[topic]))
		}
	}

	otherCount := len(r.Entries)
	for _, topic := range topics {
		otherCount -= topicCounts[topic]
	}
	if otherCount > 0 {
		sb.WriteString(fmt.Sprintf("  %-15s %d entries shown  (%d tokens)\n", "other", otherCount, r.TotalTokens-sumValues(topicTokens)))
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Skipped: %d entries\n", r.Skipped))
	sb.WriteString("  Use --full for complete content or --budget 5000 for more.\n")

	return sb.String()
}

func sumValues(m map[string]int) int {
	total := 0
	for _, v := range m {
		total += v
	}
	return total
}
