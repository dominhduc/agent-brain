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

	systemPrompt := `You are analyzing a git commit to extract knowledge for a coding agent's memory system.

The agent works on this codebase over time. Your job is to identify patterns, gotchas,
decisions, and architectural insights from the code changes.

## Rules
- Only extract knowledge that is NOT obvious from reading the code
- Focus on things that would save time or prevent mistakes in future sessions
- Be specific: mention file paths, function names, exact patterns
- If nothing noteworthy was found, return empty arrays
- Do NOT hallucinate — only report what the diff actually shows
- Output ONLY valid JSON, no markdown formatting, no explanation

## Categories
- **gotchas**: Things that could trip up the agent (error patterns, edge cases, quirks)
- **patterns**: Conventions the code follows (naming, structure, tool choices)
- **decisions**: Why certain choices were made (trade-offs, rejected alternatives visible in diff)
- **architecture**: Module relationships, key abstractions, data flow`

	userPrompt := fmt.Sprintf("## Input\nFull diff:\n%s\n\n## Output Format (JSON only)\n{\n  \"gotchas\": [\"Finding 1\", \"Finding 2\"],\n  \"patterns\": [\"Finding 1\"],\n  \"decisions\": [\"Finding 1\"],\n  \"architecture\": [],\n  \"confidence\": \"HIGH|MEDIUM|LOW\"\n}", req.Diff)

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
	}

	if err := json.Unmarshal([]byte(content), &finding); err != nil {
		return finding, fmt.Errorf("failed to parse findings JSON: %w", err)
	}

	return finding, nil
}
