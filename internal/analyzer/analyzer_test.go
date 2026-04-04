package analyzer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyze_Success(t *testing.T) {
	expected := Finding{
		Gotchas:      []string{"Project uses argon2id"},
		Patterns:     []string{"Tests use Vitest"},
		Decisions:    []string{"SQLite over PostgreSQL"},
		Architecture: []string{"cmd/ separates CLI from logic"},
		Confidence:   "HIGH",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{
					"content": `{"gotchas":["Project uses argon2id"],"patterns":["Tests use Vitest"],"decisions":["SQLite over PostgreSQL"],"architecture":["cmd/ separates CLI from logic"],"confidence":"HIGH"}`,
				}},
			},
		}
		json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	finding, err := Analyze(AnalyzeRequest{
		Diff:     "some diff content",
		APIKey:   "test-key",
		Model:    "test-model",
		Provider: "custom",
		BaseURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(finding.Gotchas) != 1 || finding.Gotchas[0] != expected.Gotchas[0] {
		t.Errorf("expected gotcha %q, got %v", expected.Gotchas[0], finding.Gotchas)
	}
	if finding.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence, got %s", finding.Confidence)
	}
	if len(finding.Patterns) != 1 || finding.Patterns[0] != expected.Patterns[0] {
		t.Errorf("expected pattern %q, got %v", expected.Patterns[0], finding.Patterns)
	}
	if len(finding.Decisions) != 1 || finding.Decisions[0] != expected.Decisions[0] {
		t.Errorf("expected decision %q, got %v", expected.Decisions[0], finding.Decisions)
	}
	if len(finding.Architecture) != 1 || finding.Architecture[0] != expected.Architecture[0] {
		t.Errorf("expected architecture %q, got %v", expected.Architecture[0], finding.Architecture)
	}
}

func TestAnalyze_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid API key"}}`))
	}))
	defer server.Close()

	_, err := Analyze(AnalyzeRequest{
		Diff:     "some diff",
		APIKey:   "bad-key",
		Model:    "test-model",
		Provider: "custom",
		BaseURL:  server.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestAnalyze_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	_, err := Analyze(AnalyzeRequest{
		Diff:     "some diff",
		APIKey:   "test-key",
		Model:    "test-model",
		Provider: "custom",
		BaseURL:  server.URL,
	})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestAnalyze_InvalidProvider(t *testing.T) {
	_, err := Analyze(AnalyzeRequest{
		Diff:     "some diff",
		APIKey:   "test-key",
		Model:    "test-model",
		Provider: "invalid-provider",
	})
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}
