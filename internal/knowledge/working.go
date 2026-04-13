package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type WMEntry struct {
	Content    string    `json:"content"`
	Importance float64   `json:"importance"`
	Timestamp  time.Time `json:"timestamp"`
}

const MaxWMEntries = 20

func (h *Hub) PushWM(content string, importance float64) error {
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}

	dir := filepath.Join(h.dir, "buffer")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating buffer dir: %w", err)
	}

	entries, _ := h.loadWM()
	entries = append(entries, WMEntry{
		Content:    content,
		Importance: importance,
		Timestamp:  time.Now(),
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Importance > entries[j].Importance
	})
	if len(entries) > MaxWMEntries {
		entries = entries[:MaxWMEntries]
	}

	return h.saveWM(entries)
}

func (h *Hub) ReadWM() ([]WMEntry, error) {
	return h.loadWM()
}

func (h *Hub) ClearWM() error {
	path := filepath.Join(h.dir, "buffer", "wm.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (h *Hub) loadWM() ([]WMEntry, error) {
	path := filepath.Join(h.dir, "buffer", "wm.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading buffer: %w", err)
	}
	var entries []WMEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, nil
	}
	return entries, nil
}

func (h *Hub) saveWM(entries []WMEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling buffer: %w", err)
	}
	dir := filepath.Join(h.dir, "buffer")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating buffer dir: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "wm.json"), data, 0600)
}
