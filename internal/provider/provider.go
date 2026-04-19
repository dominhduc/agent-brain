package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Provider interface {
	Name() string
	BuildURL(model, baseURL string) string
	BuildHeaders(apiKey string) map[string]string
	BuildBody(model, systemPrompt, userPrompt string) ([]byte, error)
	ParseResponse(body []byte) (string, error)
}

var providers = map[string]func() Provider{
	"openrouter": func() Provider { return &OpenRouter{} },
	"openai":     func() Provider { return &OpenAI{} },
	"anthropic":  func() Provider { return &Anthropic{} },
	"gemini":     func() Provider { return &Gemini{} },
	"ollama":     func() Provider { return &Ollama{} },
}

func New(name string) (Provider, error) {
	f, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return f(), nil
}

func NewCustom() Provider {
	return &Custom{}
}

func IsValid(name string) bool {
	_, ok := providers[name]
	return ok
}

func IsBuiltin(name string) bool {
	_, ok := providers[name]
	return ok
}

type OpenAI struct{}

func (p *OpenAI) Name() string     { return "openai" }
func (p *OpenAI) BuildURL(_, _ string) string {
	return "https://api.openai.com/v1/chat/completions"
}

func (p *OpenAI) BuildHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
}

func (p *OpenAI) BuildBody(model, systemPrompt, userPrompt string) ([]byte, error) {
	messages := []map[string]string{
		{"role": "user", "content": userPrompt},
	}
	if systemPrompt != "" {
		messages = append([]map[string]string{{"role": "system", "content": systemPrompt}}, messages...)
	}
	return json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})
}

func (p *OpenAI) ParseResponse(body []byte) (string, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}

type OpenRouter struct{}

func (p *OpenRouter) Name() string     { return "openrouter" }
func (p *OpenRouter) BuildURL(_, _ string) string {
	return "https://openrouter.ai/api/v1/chat/completions"
}

func (p *OpenRouter) BuildHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
		"HTTP-Referer":  "https://github.com/dominhduc/agent-brain",
		"X-Title":       "agent-brain",
	}
}

func (p *OpenRouter) BuildBody(model, systemPrompt, userPrompt string) ([]byte, error) {
	messages := []map[string]string{
		{"role": "user", "content": userPrompt},
	}
	if systemPrompt != "" {
		messages = append([]map[string]string{{"role": "system", "content": systemPrompt}}, messages...)
	}
	return json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})
}

func (p *OpenRouter) ParseResponse(body []byte) (string, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				Refusal   string `json:"refusal"`
				Reasoning string `json:"reasoning"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	msg := resp.Choices[0].Message
	if msg.Content != "" && !strings.HasPrefix(msg.Content, "Thinking") {
		return msg.Content, nil
	}
	if msg.Refusal != "" {
		return "", fmt.Errorf("refused: %s", msg.Refusal)
	}
	if msg.Reasoning != "" {
		start := strings.Index(msg.Reasoning, "{")
		end := strings.LastIndex(msg.Reasoning, "}")
		if start >= 0 && end > start {
			return msg.Reasoning[start:end+1], nil
		}
		return msg.Reasoning, nil
	}
	return "", fmt.Errorf("empty response")
}

type Anthropic struct{}

func (p *Anthropic) Name() string     { return "anthropic" }
func (p *Anthropic) BuildURL(_, _ string) string {
	return "https://api.anthropic.com/v1/messages"
}

func (p *Anthropic) BuildHeaders(apiKey string) map[string]string {
	return map[string]string{
		"x-api-key":        apiKey,
		"anthropic-version": "2023-06-01",
		"Content-Type":     "application/json",
	}
}

func (p *Anthropic) BuildBody(model, systemPrompt, userPrompt string) ([]byte, error) {
	messages := []map[string]string{
		{"role": "user", "content": userPrompt},
	}
	body := map[string]interface{}{
		"model":      model,
		"max_tokens": 1024,
		"messages":   messages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	return json.Marshal(body)
}

func (p *Anthropic) ParseResponse(body []byte) (string, error) {
	var resp struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}
	return resp.Content[0].Text, nil
}

type Gemini struct{}

func (p *Gemini) Name() string { return "gemini" }

func (p *Gemini) BuildURL(model, _ string) string {
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
}

func (p *Gemini) BuildHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Content-Type":   "application/json",
		"x-goog-api-key": apiKey,
	}
}

func (p *Gemini) BuildBody(model, systemPrompt, userPrompt string) ([]byte, error) {
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": userPrompt},
				},
			},
		},
	}
	if systemPrompt != "" {
		body["systemInstruction"] = map[string]interface{}{
			"role": "system",
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		}
	}
	_ = model
	return json.Marshal(body)
}

func (p *Gemini) ParseResponse(body []byte) (string, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}

type Ollama struct{}

func (p *Ollama) Name() string { return "ollama" }

func (p *Ollama) BuildURL(model, baseURL string) string {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return strings.TrimSuffix(baseURL, "/") + "/api/chat"
}

func (p *Ollama) BuildHeaders(_ string) map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

func (p *Ollama) BuildBody(model, systemPrompt, userPrompt string) ([]byte, error) {
	messages := []map[string]string{}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userPrompt})
	return json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	})
}

func (p *Ollama) ParseResponse(body []byte) (string, error) {
	var resp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	return resp.Message.Content, nil
}

type Custom struct{}

func (p *Custom) Name() string { return "custom" }

func (p *Custom) BuildURL(model, baseURL string) string {
	if baseURL == "" {
		return ""
	}
	url := strings.TrimSuffix(baseURL, "/")
	if !strings.Contains(url, "/v1/chat/completions") {
		url += "/v1/chat/completions"
	}
	return url
}

func (p *Custom) BuildHeaders(apiKey string) map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	return headers
}

func (p *Custom) BuildBody(model, systemPrompt, userPrompt string) ([]byte, error) {
	messages := []map[string]string{
		{"role": "user", "content": userPrompt},
	}
	if systemPrompt != "" {
		messages = append([]map[string]string{{"role": "system", "content": systemPrompt}}, messages...)
	}
	return json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})
}

func (p *Custom) ParseResponse(body []byte) (string, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}
