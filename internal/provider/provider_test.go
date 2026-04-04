package provider

import (
	"testing"
)

func TestProviderNew(t *testing.T) {
	tests := []struct {
		name      string
		wantName  string
		wantError bool
	}{
		{"openrouter", "openrouter", false},
		{"openai", "openai", false},
		{"anthropic", "anthropic", false},
		{"gemini", "gemini", false},
		{"ollama", "ollama", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.name)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name() != tt.wantName {
				t.Errorf("got %s, want %s", p.Name(), tt.wantName)
			}
		})
	}
}

func TestNewCustom(t *testing.T) {
	p := NewCustom()
	if p.Name() != "custom" {
		t.Errorf("got %s, want custom", p.Name())
	}
}

func TestIsBuiltin(t *testing.T) {
	if !IsBuiltin("openrouter") {
		t.Error("expected openrouter to be builtin")
	}
	if !IsBuiltin("ollama") {
		t.Error("expected ollama to be builtin")
	}
	if IsBuiltin("custom") {
		t.Error("expected custom to NOT be builtin")
	}
	if IsBuiltin("groq") {
		t.Error("expected groq to NOT be builtin")
	}
}

func TestOpenAI_BuildURL(t *testing.T) {
	p := &OpenAI{}
	url := p.BuildURL("gpt-4o", "")
	if url != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("got %s, want OpenAI URL", url)
	}
}

func TestOpenRouter_BuildURL(t *testing.T) {
	p := &OpenRouter{}
	url := p.BuildURL("anthropic/claude-3.5-haiku", "")
	if url != "https://openrouter.ai/api/v1/chat/completions" {
		t.Errorf("got %s, want OpenRouter URL", url)
	}
}

func TestAnthropic_BuildURL(t *testing.T) {
	p := &Anthropic{}
	url := p.BuildURL("claude-3-5-haiku-20241022", "")
	if url != "https://api.anthropic.com/v1/messages" {
		t.Errorf("got %s, want Anthropic URL", url)
	}
}

func TestGemini_BuildURL(t *testing.T) {
	p := &Gemini{}
	url := p.BuildURL("gemini-2.0-flash", "api-key-123")
	expected := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=api-key-123"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestOllama_BuildURL(t *testing.T) {
	p := &Ollama{}
	
	url := p.BuildURL("llama3.2", "")
	if url != "http://localhost:11434/api/chat" {
		t.Errorf("got %s, want localhost URL", url)
	}

	url = p.BuildURL("llama3.2", "http://192.168.1.100:11434")
	if url != "http://192.168.1.100:11434/api/chat" {
		t.Errorf("got %s, want custom URL", url)
	}
}

func TestCustom_BuildURL(t *testing.T) {
	p := &Custom{}

	url := p.BuildURL("gpt-4o", "")
	if url != "" {
		t.Errorf("got %s, want empty", url)
	}

	url = p.BuildURL("gpt-4o", "http://localhost:8080")
	if url != "http://localhost:8080/v1/chat/completions" {
		t.Errorf("got %s, want custom URL with /v1/chat/completions", url)
	}

	url = p.BuildURL("gpt-4o", "http://localhost:8080/v1/chat/completions")
	if url != "http://localhost:8080/v1/chat/completions" {
		t.Errorf("got %s, want URL as-is", url)
	}
}

func TestProvider_BuildBody(t *testing.T) {
	system := "You are a helpful assistant."
	user := "Hello"

	tests := []struct {
		name      string
		provider  Provider
		wantError bool
	}{
		{"openai", &OpenAI{}, false},
		{"openrouter", &OpenRouter{}, false},
		{"anthropic", &Anthropic{}, false},
		{"gemini", &Gemini{}, false},
		{"ollama", &Ollama{}, false},
		{"custom", &Custom{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := tt.provider.BuildBody("test-model", system, user)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(body) == 0 {
				t.Error("expected non-empty body")
			}
		})
	}
}

func TestProvider_BuildHeaders(t *testing.T) {
	apiKey := "test-key-123"

	tests := []struct {
		name     string
		provider Provider
		wantKey  string
		wantVal  string
	}{
		{"openai", &OpenAI{}, "Authorization", "Bearer test-key-123"},
		{"openrouter", &OpenRouter{}, "Authorization", "Bearer test-key-123"},
		{"anthropic", &Anthropic{}, "x-api-key", "test-key-123"},
		{"ollama", &Ollama{}, "Content-Type", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := tt.provider.BuildHeaders(apiKey)
			if got := headers[tt.wantKey]; got != tt.wantVal {
				t.Errorf("got %s=%s, want %s=%s", tt.wantKey, got, tt.wantKey, tt.wantVal)
			}
		})
	}
}

func TestProvider_ParseResponse(t *testing.T) {
	tests := []struct {
		name      string
		provider  Provider
		response  string
		wantError bool
		wantContent string
	}{
		{
			name: "openai",
			provider: &OpenAI{},
			response: `{"choices":[{"message":{"content":"Hello!"}}]}`,
			wantContent: "Hello!",
		},
		{
			name: "anthropic",
			provider: &Anthropic{},
			response: `{"content":[{"type":"text","text":"Hello from Claude!"}]}`,
			wantContent: "Hello from Claude!",
		},
		{
			name: "gemini",
			provider: &Gemini{},
			response: `{"candidates":[{"content":{"parts":[{"text":"Hello from Gemini!"}]}}]}`,
			wantContent: "Hello from Gemini!",
		},
		{
			name: "ollama",
			provider: &Ollama{},
			response: `{"message":{"content":"Hello from Ollama!"}}`,
			wantContent: "Hello from Ollama!",
		},
		{
			name: "invalid",
			provider: &OpenAI{},
			response: `{"error":"something wrong"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := tt.provider.ParseResponse([]byte(tt.response))
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if content != tt.wantContent {
				t.Errorf("got %q, want %q", content, tt.wantContent)
			}
		})
	}
}
