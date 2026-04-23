package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MemoryFeedback struct {
	Key       string    `json:"key"`
	Retrieved time.Time `json:"retrieved"`
	Helped    *bool     `json:"helped,omitempty"`
}

type BehaviorSignals struct {
	CommandCounts   map[string]int              `json:"command_counts"`
	TopicAccess     map[string]*TopicAccessInfo `json:"topic_access"`
	SearchQueries   []SearchQuery               `json:"search_queries"`
	EvalOutcomes    EvalOutcomeInfo             `json:"eval_outcomes"`
	MemoryFeedbacks []MemoryFeedback            `json:"memory_feedbacks"`
	LastUpdated     time.Time                   `json:"last_updated"`
}

type TopicAccessInfo struct {
	Retrievals int       `json:"retrievals"`
	LastAccess time.Time `json:"last_access"`
}

type SearchQuery struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

type EvalOutcomeInfo struct {
	Good          int       `json:"good"`
	Bad           int       `json:"bad"`
	TotalSessions int       `json:"total_sessions"`
	LastEval      time.Time `json:"last_eval"`
}

var behaviorMu sync.Mutex

func (h *Hub) behaviorPath() string {
	return filepath.Join(h.dir, "behavior", "signals.json")
}

func (h *Hub) LoadBehavior() (*BehaviorSignals, error) {
	behaviorMu.Lock()
	defer behaviorMu.Unlock()

	data, err := os.ReadFile(h.behaviorPath())
	if err != nil {
		if os.IsNotExist(err) {
			return h.emptyBehavior(), nil
		}
		return nil, fmt.Errorf("reading behavior: %w", err)
	}
	var signals BehaviorSignals
	if err := json.Unmarshal(data, &signals); err != nil {
		return h.emptyBehavior(), nil
	}
	return &signals, nil
}

func (h *Hub) SaveBehavior(signals *BehaviorSignals) error {
	behaviorMu.Lock()
	defer behaviorMu.Unlock()

	signals.LastUpdated = time.Now()
	data, err := json.MarshalIndent(signals, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling behavior: %w", err)
	}

	dir := filepath.Dir(h.behaviorPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating behavior dir: %w", err)
	}

	return os.WriteFile(h.behaviorPath(), data, 0600)
}

func (h *Hub) TrackCommand(command string) error {
	signals, err := h.LoadBehavior()
	if err != nil {
		return err
	}
	if signals.CommandCounts == nil {
		signals.CommandCounts = make(map[string]int)
	}
	signals.CommandCounts[command]++
	return h.SaveBehavior(signals)
}

func (h *Hub) TrackTopicAccess(topic string) error {
	signals, err := h.LoadBehavior()
	if err != nil {
		return err
	}
	if signals.TopicAccess == nil {
		signals.TopicAccess = make(map[string]*TopicAccessInfo)
	}
	info, ok := signals.TopicAccess[topic]
	if !ok {
		info = &TopicAccessInfo{}
		signals.TopicAccess[topic] = info
	}
	info.Retrievals++
	info.LastAccess = time.Now()
	return h.SaveBehavior(signals)
}

func (h *Hub) TrackSearch(query string) error {
	signals, err := h.LoadBehavior()
	if err != nil {
		return err
	}
	for i := range signals.SearchQueries {
		if signals.SearchQueries[i].Query == query {
			signals.SearchQueries[i].Count++
			return h.SaveBehavior(signals)
		}
	}
	signals.SearchQueries = append(signals.SearchQueries, SearchQuery{Query: query, Count: 1})
	if len(signals.SearchQueries) > 200 {
		signals.SearchQueries = signals.SearchQueries[len(signals.SearchQueries)-200:]
	}
	return h.SaveBehavior(signals)
}

func (h *Hub) TrackEvalOutcome(good bool) error {
	signals, err := h.LoadBehavior()
	if err != nil {
		return err
	}
	if good {
		signals.EvalOutcomes.Good++
	} else {
		signals.EvalOutcomes.Bad++
	}
	signals.EvalOutcomes.TotalSessions++
	signals.EvalOutcomes.LastEval = time.Now()
	return h.SaveBehavior(signals)
}

func (h *Hub) emptyBehavior() *BehaviorSignals {
	return &BehaviorSignals{
		CommandCounts:   make(map[string]int),
		TopicAccess:     make(map[string]*TopicAccessInfo),
		SearchQueries:   []SearchQuery{},
		MemoryFeedbacks: []MemoryFeedback{},
	}
}

func (h *Hub) RecordMemoryFeedback(keys []string, helped bool) error {
	signals, err := h.LoadBehavior()
	if err != nil {
		return err
	}
	now := time.Now()
	for _, key := range keys {
		signals.MemoryFeedbacks = append(signals.MemoryFeedbacks, MemoryFeedback{
			Key:       key,
			Retrieved: now,
			Helped:    &helped,
		})
	}
	if len(signals.MemoryFeedbacks) > 100 {
		signals.MemoryFeedbacks = signals.MemoryFeedbacks[len(signals.MemoryFeedbacks)-100:]
	}
	return h.SaveBehavior(signals)
}

func (h *Hub) ApplyFeedbackToStrength() (int, error) {
	signals, err := h.LoadBehavior()
	if err != nil {
		return 0, err
	}

	keyHelpCount := make(map[string]int)
	keyTotalCount := make(map[string]int)
	for _, fb := range signals.MemoryFeedbacks {
		keyTotalCount[fb.Key]++
		if fb.Helped != nil && *fb.Helped {
			keyHelpCount[fb.Key]++
		}
	}

	idx, err := h.LoadIndex()
	if err != nil {
		return 0, err
	}

	updated := 0
	for key, entry := range idx.Entries {
		total := keyTotalCount[key]
		if total == 0 {
			continue
		}
		helpCount := keyHelpCount[key]
		successRate := float64(helpCount) / float64(total)

		bonus := successRate * 7.0
		effectiveHalfLife := float64(entry.HalfLifeDays) + float64(entry.RetrievalCount)*2 + bonus
		if effectiveHalfLife > float64(entry.HalfLifeDays)+14 {
			effectiveHalfLife = float64(entry.HalfLifeDays) + 14
		}
		entry.HalfLifeDays = int(effectiveHalfLife)
		idx.Entries[key] = entry
		updated++
	}

	if updated > 0 {
		if err := idx.Save(h.dir); err != nil {
			return 0, err
		}
	}
	return updated, nil
}
