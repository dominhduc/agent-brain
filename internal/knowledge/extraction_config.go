package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ExtractionConfig struct {
	AcceptanceRates  map[string]float64 `json:"acceptance_rates"`
	AvgItemsPerCommit float64           `json:"avg_items_per_commit"`
	TotalReviewed    int                `json:"total_reviewed"`
	LastUpdated      time.Time          `json:"last_updated"`
}

func (h *Hub) extractionConfigPath() string {
	return filepath.Join(h.dir, "extraction-config.json")
}

func (h *Hub) LoadExtractionConfig() ExtractionConfig {
	data, err := os.ReadFile(h.extractionConfigPath())
	if err != nil {
		return ExtractionConfig{
			AcceptanceRates: make(map[string]float64),
		}
	}
	var cfg ExtractionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ExtractionConfig{
			AcceptanceRates: make(map[string]float64),
		}
	}
	if cfg.AcceptanceRates == nil {
		cfg.AcceptanceRates = make(map[string]float64)
	}
	return cfg
}

func (h *Hub) SaveExtractionConfig(cfg ExtractionConfig) error {
	cfg.LastUpdated = time.Now()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.extractionConfigPath(), data, 0600)
}

func (h *Hub) UpdateExtractionStats(accepted, rejected map[string]int) error {
	config := h.LoadExtractionConfig()

	for topic, count := range accepted {
		totalForTopic := count + rejected[topic]
		if totalForTopic > 0 {
			rate := float64(count) / float64(totalForTopic)
			if existing, ok := config.AcceptanceRates[topic]; ok {
				config.AcceptanceRates[topic] = movingAverage(existing, rate, 0.3)
			} else {
				config.AcceptanceRates[topic] = rate
			}
		}
	}

	totalAccepted := 0
	totalRejected := 0
	for _, c := range accepted {
		totalAccepted += c
	}
	for _, c := range rejected {
		totalRejected += c
	}
	config.TotalReviewed += totalAccepted + totalRejected

	if config.TotalReviewed > 0 {
		config.AvgItemsPerCommit = float64(totalAccepted+totalRejected) / float64(config.TotalReviewed)
	}

	return h.SaveExtractionConfig(config)
}

func (h *Hub) BuildAdaptiveGuidance() string {
	config := h.LoadExtractionConfig()
	if config.TotalReviewed < 10 {
		return ""
	}

	var hints []string
	for topic, rate := range config.AcceptanceRates {
		if rate < 0.4 {
			hints = append(hints, fmt.Sprintf(
				"Be especially selective about %s entries — only extract if clearly valuable.",
				topic,
			))
		}
	}

	if len(hints) == 0 {
		return ""
	}

	return fmt.Sprintf("\n\nAdditional guidance based on review history (%d entries reviewed):\n%s",
		config.TotalReviewed,
		strings.Join(hints, "\n"))
}

func movingAverage(old, new, alpha float64) float64 {
	return old*(1-alpha) + new*alpha
}
