package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/httpclient"
)

type Finding struct {
	Gotchas      []string `json:"gotchas"`
	Patterns     []string `json:"patterns"`
	Decisions    []string `json:"decisions"`
	Architecture []string `json:"architecture"`
	Confidence   string   `json:"confidence"`
}

type AnalyzeRequest struct {
	Diff       string
	APIKey     string
	Model      string
	APIBaseURL string
}

func Analyze(req AnalyzeRequest) (Finding, error) {
	var finding Finding

	prompt := fmt.Sprintf(`You are analyzing a git commit to extract knowledge for a coding agent's memory system.

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
- **architecture**: Module relationships, key abstractions, data flow

## Input
Full diff:
%s

## Output Format (JSON only)
{
  "gotchas": ["Finding 1", "Finding 2"],
  "patterns": ["Finding 1"],
  "decisions": ["Finding 1"],
  "architecture": [],
  "confidence": "HIGH|MEDIUM|LOW"
}`, req.Diff)

	reqBody := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	headers := map[string]string{
		"Authorization": "Bearer " + req.APIKey,
		"HTTP-Referer":  "https://github.com/dominhduc/agent-brain",
		"X-Title":       "agent-brain",
	}

	url := req.APIBaseURL
	if url == "" {
		url = "https://openrouter.ai/api/v1/chat/completions"
	}

	respBody, err := httpclient.PostJSON(url, headers, reqBody)
	if err != nil {
		return finding, err
	}

	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return finding, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Error.Message != "" {
		return finding, fmt.Errorf("API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return finding, fmt.Errorf("no choices in response")
	}

	content := resp.Choices[0].Message.Content

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

func WriteFindings(finding Finding, brainDir string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	writeEntries := func(filename string, entries []string) error {
		if len(entries) == 0 {
			return nil
		}
		path := filepath.Join(brainDir, filename)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		for _, entry := range entries {
			fmt.Fprintf(f, "\n### [%s] %s\n\n", timestamp, entry)
		}
		return nil
	}

	if err := writeEntries("gotchas.md", finding.Gotchas); err != nil {
		return err
	}
	if err := writeEntries("patterns.md", finding.Patterns); err != nil {
		return err
	}
	if err := writeEntries("decisions.md", finding.Decisions); err != nil {
		return err
	}
	if err := writeEntries("architecture.md", finding.Architecture); err != nil {
		return err
	}

	return nil
}
