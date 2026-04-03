package httpclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{201, false},
		{301, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := IsRetryable(tt.code)
			if got != tt.want {
				t.Errorf("IsRetryable(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestPostJSON_Success(t *testing.T) {
	expected := map[string]string{"result": "ok"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		if ua := r.Header.Get("User-Agent"); ua == "" {
			t.Error("expected User-Agent header to be set")
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["query"] != "test" {
			t.Errorf("expected body.query='test', got '%s'", body["query"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	respBody, err := PostJSON(server.URL, nil, map[string]string{"query": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("expected result='ok', got '%s'", result["result"])
	}
}

func TestPostJSON_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("X-Custom"); v != "test-value" {
			t.Errorf("expected X-Custom='test-value', got '%s'", v)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, err := PostJSON(server.URL, map[string]string{"X-Custom": "test-value"}, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJSON_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	_, err := PostJSON(server.URL, nil, map[string]string{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	apiErr, ok := err.(APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Body != "internal server error" {
		t.Errorf("expected body 'internal server error', got '%s'", apiErr.Body)
	}
}

func TestPostJSON_429Retryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	_, err := PostJSON(server.URL, nil, map[string]string{})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	apiErr, ok := err.(APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if !IsRetryable(apiErr.StatusCode) {
		t.Error("expected 429 to be retryable")
	}
}
