package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) Dimensions() int { return 768 }

func (p *OllamaProvider) Embed(text string) ([]float32, error) {
	results, err := p.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return results[0], nil
}

func (p *OllamaProvider) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"model": p.model,
		"input": texts,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := p.baseURL + "/api/embeddings"
	resp, err := p.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embeddings, nil
}
