package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Dimensions() int { return 1536 }

func (p *OpenAIProvider) Embed(text string) ([]float32, error) {
	results, err := p.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return results[0], nil
}

func (p *OpenAIProvider) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"model": p.model,
		"input": texts,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := p.baseURL + "/embeddings"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI returned status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}
	return embeddings, nil
}
