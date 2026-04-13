package daemon

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
	Topics       []string `json:"topics"`
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

	systemPrompt := "You are a code analyst. Always respond with ONLY a valid JSON object, no markdown, no explanation."
	userPrompt := `Analyze this git diff and extract knowledge. Respond with ONLY this JSON format, no other text:
{"gotchas":["..."],"patterns":["..."],"decisions":["..."],"architecture":["..."],"confidence":"HIGH|MEDIUM|LOW","topics":["ui","backend","infrastructure","database","security","testing","architecture","general"]}

Rules: use empty arrays [] if nothing found. Keep entries short (one sentence each). Only include relevant categories. For topics, classify each entry into one or more: ui, backend, infrastructure, database, security, testing, architecture. Use "general" if none match.

Diff:
` + req.Diff

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
		content = content[jsonStart : jsonEnd+1]
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
