package config

import (
	"testing"
)

func TestResolveKey_FriendlyKey(t *testing.T) {
	key, err := ResolveKey("api-key")
	if err != nil {
		t.Fatalf("ResolveKey(\"api-key\") error: %v", err)
	}
	if key.DotPath != "llm.api_key" {
		t.Errorf("expected dot path \"llm.api_key\", got %q", key.DotPath)
	}
}

func TestResolveKey_DotPath(t *testing.T) {
	key, err := ResolveKey("llm.api_key")
	if err != nil {
		t.Fatalf("ResolveKey(\"llm.api_key\") error: %v", err)
	}
	if key.Friendly != "api-key" {
		t.Errorf("expected friendly \"api-key\", got %q", key.Friendly)
	}
}

func TestResolveKey_Unknown(t *testing.T) {
	_, err := ResolveKey("unknown-key")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestAllKeys_Count(t *testing.T) {
	keys := AllKeys()
	if len(keys) != 13 {
		t.Errorf("expected 9 keys, got %d", len(keys))
	}
}

func TestGetKeyByFriendly(t *testing.T) {
	key := GetKeyByFriendly("model")
	if key == nil {
		t.Fatal("expected key for \"model\"")
	}
	if key.DotPath != "llm.model" {
		t.Errorf("expected dot path \"llm.model\", got %q", key.DotPath)
	}
}

func TestGetKeyByFriendly_NotFound(t *testing.T) {
	key := GetKeyByFriendly("nonexistent")
	if key != nil {
		t.Error("expected nil for nonexistent key")
	}
}

func TestGetKeyByDotPath(t *testing.T) {
	key := GetKeyByDotPath("daemon.poll_interval")
	if key == nil {
		t.Fatal("expected key for \"daemon.poll_interval\"")
	}
	if key.Friendly != "poll-interval" {
		t.Errorf("expected friendly \"poll-interval\", got %q", key.Friendly)
	}
}

func TestConfigKey_Validate(t *testing.T) {
	tests := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"api-key", "sk-test-key", false},
		{"model", "anthropic/claude-3.5-haiku", false},
		{"profile", "guard", false},
		{"profile", "invalid", true},
		{"poll-interval", "10s", false},
		{"poll-interval", "not-a-duration", true},
		{"max-retries", "5", false},
		{"max-retries", "abc", true},
		{"max-retries", "0", true},
		{"max-diff-lines", "3000", false},
		{"max-diff-lines", "50", true},
	}

	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.value, func(t *testing.T) {
			key := GetKeyByFriendly(tt.key)
			if key == nil {
				t.Fatalf("key %q not found", tt.key)
			}
			err := key.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestConfigKey_ApplyValue(t *testing.T) {
	cfg := DefaultConfig()

	key := GetKeyByFriendly("api-key")
	if key == nil {
		t.Fatal("key not found")
	}

	if err := key.ApplyValue(&cfg, "test-key"); err != nil {
		t.Fatalf("ApplyValue error: %v", err)
	}
	if cfg.LLM.APIKey != "test-key" {
		t.Errorf("expected API key \"test-key\", got %q", cfg.LLM.APIKey)
	}
}
