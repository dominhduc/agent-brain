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
	Items      []FindingItem `json:"items"`
	Confidence string        `json:"confidence"`
}

type FindingItem struct {
	Title      string   `json:"title"`
	Topic      string   `json:"topic"`
	Content    string   `json:"content"`
	Confidence string   `json:"confidence"`
	Tags       []string `json:"tags"`
}

type AnalyzeRequest struct {
	Diff     string
	APIKey   string
	Model    string
	Provider string
	BaseURL  string
}

func CallLLM(req AnalyzeRequest, userPrompt string) (string, error) {
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
			return "", fmt.Errorf("invalid provider: %w", err)
		}
	}

	url := p.BuildURL(req.Model, req.BaseURL)
	if url == "" {
		return "", fmt.Errorf("no URL for provider: %s (hint: set base-url for custom provider)", req.Provider)
	}

	headers := p.BuildHeaders(req.APIKey)
	if req.Provider == "gemini" {
		headers["Content-Type"] = "application/json"
	}

	body, err := p.BuildBody(req.Model, "You are a knowledge base grading assistant.", userPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to build request body: %w", err)
	}

	respBody, err := httpclient.PostJSON(url, headers, body)
	if err != nil {
		return "", err
	}

	return p.ParseResponse(respBody)
}

func Analyze(req AnalyzeRequest) (Finding, error) {
	return AnalyzeWithPrompt(req, "")
}

func AnalyzeWithPrompt(req AnalyzeRequest, extraGuidance string) (Finding, error) {
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

	systemPrompt := "You are a senior engineer writing a knowledge base for future developers (human and AI). Extract knowledge that would help someone encountering similar code changes in the future."

	userPrompt := `Analyze this git diff and extract knowledge. Respond with ONLY a JSON array of objects, no other text.

Each object represents one knowledge entry with this structure:
{"title":"Short imperative title (e.g. 'Set explicit TTL on all Redis keys')","topic":"gotchas|patterns|decisions|architecture","content":"2-3 sentence explanation: WHAT to do, WHY it matters, and HOW to do it. Include specific values, file patterns, or function names when they are essential to the lesson. Omit them when they are incidental to this specific change.","confidence":"HIGH|MEDIUM|LOW","tags":["ui","backend","infrastructure","database","security","testing","architecture","general"]}

Rules:
1. Maximum 5 entries. Fewer is better. Only extract genuinely useful knowledge.
2. Do NOT extract trivial changes (formatting, version bumps, typo fixes).
3. Do NOT include specific variable names, file paths, or literal values UNLESS they represent a non-obvious gotcha (e.g., "GORM's First() returns ErrRecordNotFound, not nil").
4. For gotchas: explain the mistake, the symptom it causes, and the correct approach.
5. For patterns: describe the general technique, when to apply it, and what it replaces.
6. For decisions: state the choice made, the alternatives rejected, and the reasoning.
7. For architecture: describe the structural change, why it was needed, and what it enables.
8. Use "general" tag only when no other tag applies.`

	if extraGuidance != "" {
		userPrompt += "\n\nAdditional guidance based on review history:\n" + extraGuidance
	}

	userPrompt += "\n\nDiff:\n" + req.Diff

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

	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		content = content[jsonStart : jsonEnd+1]
	} else {
		objStart := strings.Index(content, "{")
		objEnd := strings.LastIndex(content, "}")
		if objStart >= 0 && objEnd > objStart {
			content = "[" + content[objStart:objEnd+1] + "]"
		} else {
			return finding, fmt.Errorf("no JSON found in response")
		}
	}

	var items []FindingItem
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		return finding, fmt.Errorf("JSON parsing failed: %w", err)
	}

	if len(items) == 0 {
		return finding, nil
	}

	confidence := "MEDIUM"
	for _, item := range items {
		if item.Confidence == "HIGH" {
			confidence = "HIGH"
			break
		}
	}

	return Finding{
		Items:      items,
		Confidence: confidence,
	}, nil
}

func ContrastiveAnalyze(req AnalyzeRequest, trials int, extraGuidance string) (Finding, error) {
	if trials < 2 {
		trials = 2
	}
	if trials > 4 {
		trials = 4
	}

	type scoredItem struct {
		item         FindingItem
		appearances  int
	}

	var allFindings []Finding
	for i := 0; i < trials; i++ {
		f, err := AnalyzeWithPrompt(req, extraGuidance)
		if err != nil {
			continue
		}
		allFindings = append(allFindings, f)
	}

	if len(allFindings) == 0 {
		return Finding{}, fmt.Errorf("all extraction attempts failed")
	}

	itemCounts := make(map[string]*scoredItem)
	for _, f := range allFindings {
		for _, item := range f.Items {
			key := normalizeEntryKey(item.Title)
			if existing, ok := itemCounts[key]; ok {
				existing.appearances++
			} else {
				itemCounts[key] = &scoredItem{item: item, appearances: 1}
			}
		}
	}

	threshold := len(allFindings) / 2
	if threshold < 1 {
		threshold = 1
	}

	var consensus []FindingItem
	for _, si := range itemCounts {
		if si.appearances > threshold {
			consensus = append(consensus, si.item)
		}
	}

	confidence := "MEDIUM"
	if len(allFindings) == trials {
		confidence = "HIGH"
	}

	return Finding{
		Items:      consensus,
		Confidence: confidence,
	}, nil
}

func normalizeEntryKey(title string) string {
	s := strings.ToLower(title)
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}
