package analyzer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/httpclient"
	"github.com/dominhduc/agent-brain/internal/provider"
)

type Finding struct {
	Gotchas      []string `json:"gotchas"`
	Patterns     []string `json:"patterns"`
	Decisions    []string `json:"decisions"`
	Architecture []string `json:"architecture"`
	Confidence   string   `json:"confidence"`
}

type AnalyzeRequest struct {
	Diff     string
	APIKey   string
	Model    string
	Provider string
	BaseURL  string
}

func Analyze(req AnalyzeRequest) (Finding, error) {
	var finding Finding

	p, err := provider.New(req.Provider)
	if err != nil {
		if cp, ok := config.GetCustomProvider(req.Provider); ok {
			p = provider.NewCustom()
			if req.BaseURL == "" {
				req.BaseURL = cp.BaseURL
			}
			if req.APIKey == "" {
				req.APIKey = cp.APIKey
			}
			if req.Model == "" {
				req.Model = cp.Model
			}
		} else if req.BaseURL != "" {
			p = provider.NewCustom()
		} else {
			return finding, fmt.Errorf("invalid provider: %w", err)
		}
	}

	systemPrompt := `You are a coding assistant analyzing git commits.

Your task: extract knowledge from the diff that would help future coding sessions.

Output: ONLY valid JSON. No markdown, no code fences, no explanations.
Start your response with { and end with }.

JSON format:
{"gotchas": [], "patterns": [], "decisions": [], "architecture": [], "confidence": "HIGH"}

Rules:
- Only extract non-obvious knowledge
- If nothing found, return empty arrays
- Be specific: include file paths, function names
- confidence must be HIGH, MEDIUM, or LOW

Categories:
- gotchas: error patterns, edge cases, quirks
- patterns: naming conventions, structure patterns
- decisions: why choices were made
- architecture: module relationships, data flow`

	userPrompt := fmt.Sprintf("Analyze this git diff.\n\n%s\n\nRespond with ONLY JSON.", req.Diff)

	url := p.BuildURL(req.Model, req.BaseURL)
	if url == "" {
		return finding, fmt.Errorf("no URL for provider: %s (hint: set base-url for custom provider)", req.Provider)
	}

	headers := p.BuildHeaders(req.APIKey)
	if req.Provider == "gemini" {
		headers["Content-Type"] = "application/json"
	}

	body, err := p.BuildBody(req.Model, systemPrompt, userPrompt)
	if err != nil {
		return finding, fmt.Errorf("failed to build request body: %w", err)
	}

	respBody, err := httpclient.PostJSON(url, headers, body)
	if err != nil {
		return finding, err
	}

	content, err := p.ParseResponse(respBody)
	if err != nil {
		return finding, fmt.Errorf("failed to parse response: %w", err)
	}

	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		content = content[jsonStart:jsonEnd+1]
	} else {
		return finding, fmt.Errorf("no JSON object found in response")
	}

	if err := json.Unmarshal([]byte(content), &finding); err != nil {
		return finding, fmt.Errorf("JSON parsing failed: %w", err)
	}

	if finding.Confidence == "" {
		finding.Confidence = "MEDIUM"
	}

	return finding, nil
}
