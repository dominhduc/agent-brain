package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var safeSessionIDRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateSessionID(id string) error {
	if id == "" {
		return fmt.Errorf("session ID must not be empty")
	}
	if len(id) > 128 {
		return fmt.Errorf("session ID too long (max 128 chars)")
	}
	if !safeSessionIDRe.MatchString(id) {
		return fmt.Errorf("session ID contains invalid characters (only alphanumeric, dash, underscore allowed)")
	}
	return nil
}

type TraceStep struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Result    string    `json:"result"`
	Reasoning string    `json:"reasoning,omitempty"`
	Outcome   string    `json:"outcome"`
}

type SessionTrace struct {
	SessionID string      `json:"session_id"`
	Task      string      `json:"task"`
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Steps     []TraceStep `json:"steps"`
	Outcome   string      `json:"outcome"`
	Goal      string      `json:"goal"`
}

func (h *Hub) tracesDir() string {
	return filepath.Join(h.dir, "traces")
}

func (h *Hub) SaveTrace(trace SessionTrace) error {
	if err := validateSessionID(trace.SessionID); err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	dir := h.tracesDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, trace.SessionID+".json")
	return os.WriteFile(path, data, 0600)
}

const maxTracesLoad = 500

func (h *Hub) LoadTraces() ([]SessionTrace, error) {
	dir := h.tracesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var traces []SessionTrace
	for _, e := range entries {
		if len(traces) >= maxTracesLoad {
			break
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var trace SessionTrace
		if err := json.Unmarshal(data, &trace); err != nil {
			continue
		}
		traces = append(traces, trace)
	}
	return traces, nil
}

func (h *Hub) AppendTraceStep(step TraceStep) error {
	dir := h.tracesDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	currentPath := filepath.Join(dir, "current.json")

	data, err := os.ReadFile(currentPath)
	var trace SessionTrace
	if err == nil {
		if unmarshalErr := json.Unmarshal(data, &trace); unmarshalErr != nil {
			trace = SessionTrace{}
		}
	}

	if trace.SessionID == "" {
		trace.SessionID = fmt.Sprintf("trace-%s", time.Now().Format("20060102-150405"))
		trace.StartTime = time.Now()
	}

	step.Timestamp = time.Now()
	trace.Steps = append(trace.Steps, step)

	data, err = json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(currentPath, data, 0600)
}

func (h *Hub) FinalizeTrace(outcome, goal string) error {
	dir := h.tracesDir()
	currentPath := filepath.Join(dir, "current.json")

	var trace SessionTrace
	for attempt := 0; attempt < 3; attempt++ {
		data, err := os.ReadFile(currentPath)
		if err != nil {
			return fmt.Errorf("no active trace found")
		}
		if err := json.Unmarshal(data, &trace); err == nil {
			break
		}
		if attempt < 2 {
			time.Sleep(50 * time.Millisecond)
		} else {
			return fmt.Errorf("failed to parse trace: %w", err)
		}
	}

	trace.Outcome = outcome
	trace.Goal = goal
	trace.EndTime = time.Now()

	if err := h.SaveTrace(trace); err != nil {
		return err
	}

	os.Remove(currentPath)
	return nil
}

func (h *Hub) LoadUnextractedTraces() ([]SessionTrace, error) {
	traces, err := h.LoadTraces()
	if err != nil {
		return nil, err
	}

	extractedDir := filepath.Join(h.tracesDir(), "extracted")
	os.MkdirAll(extractedDir, 0700)

	extracted, _ := os.ReadDir(extractedDir)
	extractedSet := make(map[string]bool)
	for _, e := range extracted {
		extractedSet[e.Name()] = true
	}

	var unextracted []SessionTrace
	for _, t := range traces {
		if !extractedSet[t.SessionID+".json"] {
			unextracted = append(unextracted, t)
		}
	}
	return unextracted, nil
}

func (h *Hub) MarkTraceExtracted(sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	extractedDir := filepath.Join(h.tracesDir(), "extracted")
	os.MkdirAll(extractedDir, 0700)

	src := filepath.Join(h.tracesDir(), sessionID+".json")
	dst := filepath.Join(extractedDir, sessionID+".json")
	return os.Rename(src, dst)
}

func BuildTraceExtractionPrompt(trace SessionTrace) string {
	data, _ := json.MarshalIndent(trace, "", "  ")

	var prompt string
	if trace.Outcome == "success" {
		prompt = `You are given a session trace showing how a developer successfully completed a task.
Extract knowledge entries that capture:
1. The approach taken and why it worked
2. Any obstacles encountered and how they were overcome
3. Non-obvious insights discovered during the process
Focus on REASONING, not just outcomes.

`
	} else {
		prompt = `You are given a session trace showing a failed or partially successful task attempt.
Extract knowledge entries that capture:
1. What went wrong and WHY (not just what happened)
2. What was tried and why it didn't work
3. What would be done differently next time
These failure lessons are MORE valuable than success lessons — they prevent repeated mistakes.

`
	}

	prompt += "Respond with ONLY a JSON array of objects:\n"
	prompt += `[{"title":"...","topic":"gotchas|patterns|decisions|architecture","content":"2-3 sentence explanation","confidence":"HIGH|MEDIUM|LOW","tags":["..."]}]

Maximum 5 entries. Fewer is better. Only extract genuinely useful knowledge.

Trace:
` + string(data)

	return prompt
}

func SaveTrace(brainDir string, trace SessionTrace) error {
	h, err := Open(brainDir)
	if err != nil {
		return err
	}
	return h.SaveTrace(trace)
}

func LoadTraces(brainDir string) ([]SessionTrace, error) {
	h, err := Open(brainDir)
	if err != nil {
		return nil, err
	}
	return h.LoadTraces()
}
