package knowledge

import (
	"sort"
	"time"
)

type MergedEntry struct {
	Key       string
	Topic     string
	Message   string
	Source    string
	Score     float64
	Timestamp string
}

func MergeRetrieval(projectResult *BudgetResult, globalEntries []MergedEntry, budget RetrievalBudget) *BudgetResult {
	if len(globalEntries) == 0 {
		return projectResult
	}

	now := time.Now()
	globalIdx, _ := GlobalIndex()

	var globalScored []ScoredEntry
	for _, ge := range globalEntries {
		var strength float64 = 1.0
		if globalIdx != nil {
			if ie, found := globalIdx.GetByRawKey(ge.Key); found {
				strength = CalculateStrength(ie, now)
			}
		}

		globalScored = append(globalScored, ScoredEntry{
			Key:     ge.Key,
			Topic:   ge.Topic,
			Message: ge.Message,
			Score:   strength,
			Tokens:  EstimateTokens(ge.Message),
		})
	}

	sort.Slice(globalScored, func(i, j int) bool {
		return globalScored[i].Score > globalScored[j].Score
	})

	projectKeys := make(map[string]bool)
	for _, e := range projectResult.Entries {
		projectKeys[e.Key] = true
	}

	used := projectResult.TotalTokens
	var additional []ScoredEntry

	for _, gs := range globalScored {
		if projectKeys[gs.Key] {
			continue
		}

		if used+gs.Tokens <= budget.MaxTokens {
			additional = append(additional, gs)
			used += gs.Tokens
		}
	}

	if len(additional) == 0 {
		return projectResult
	}

	result := &BudgetResult{
		Entries:     append(projectResult.Entries, additional...),
		TotalTokens: used,
		BudgetUsed:  float64(used) / float64(budget.MaxTokens),
		Skipped:     projectResult.Skipped,
	}

	return result
}

func LoadGlobalEntriesForMerge() ([]MergedEntry, error) {
	globalDir, err := GlobalBrainDir()
	if err != nil {
		return nil, err
	}

	var entries []MergedEntry
	for _, topic := range AvailableTopics() {
		topicEntries, err := GetTopicEntriesForDir(topic, globalDir)
		if err != nil {
			continue
		}
		for _, e := range topicEntries {
			entries = append(entries, MergedEntry{
				Key:       MakeKey(topic, e.Timestamp),
				Topic:     topic,
				Message:   e.Message,
				Source:    "global",
				Timestamp: e.Timestamp,
			})
		}
	}

	return entries, nil
}

func FilterGlobalByStack(entries []MergedEntry, stack []string) []MergedEntry {
	stackSet := make(map[string]bool)
	for _, s := range stack {
		stackSet[s] = true
	}

	if len(stackSet) == 0 {
		return entries
	}

	var filtered []MergedEntry
	for _, e := range entries {
		entryTags := DetectTopics(e.Message)
		relevant := false
		for _, tag := range entryTags {
			if stackSet[tag] {
				relevant = true
				break
			}
		}
		if !relevant {
			for _, tag := range entryTags {
				if tag == "general" {
					relevant = true
					break
				}
			}
		}
		if relevant {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
