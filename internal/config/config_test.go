package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
	configDir := filepath.Join(tmpHome, ".config", "brain")
	os.MkdirAll(configDir, 0700)
	t.Setenv("HOME", tmpHome)
	t.Setenv("BRAIN_API_KEY", "")
	return configDir
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LLM.Provider != "openrouter" {
		t.Errorf("expected provider 'openrouter', got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "anthropic/claude-3.5-haiku" {
		t.Errorf("expected default model, got %q", cfg.LLM.Model)
	}
	if cfg.LLM.APIKey != "" {
		t.Error("expected empty default API key")
	}
	if cfg.Analysis.MaxDiffLines != 2000 {
		t.Errorf("expected max_diff_lines 2000, got %d", cfg.Analysis.MaxDiffLines)
	}
	if cfg.Daemon.MaxRetries != 3 {
		t.Errorf("expected max_retries 3, got %d", cfg.Daemon.MaxRetries)
	}
	if cfg.Daemon.PollInterval != "5s" {
		t.Errorf("expected poll_interval '5s', got %q", cfg.Daemon.PollInterval)
	}
}

func TestSaveAndLoad(t *testing.T) {
	setupTestConfig(t)

	cfg := DefaultConfig()
	cfg.LLM.APIKey = "sk-or-v1-testkey1234567890"
	cfg.LLM.Model = "test-model"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.LLM.APIKey != cfg.LLM.APIKey {
		t.Errorf("API key mismatch: got %q, want %q", loaded.LLM.APIKey, cfg.LLM.APIKey)
	}
	if loaded.LLM.Model != cfg.LLM.Model {
		t.Errorf("Model mismatch: got %q, want %q", loaded.LLM.Model, cfg.LLM.Model)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	configDir := setupTestConfig(t)

	cfg := DefaultConfig()
	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file permissions = %o, want 0600", info.Mode().Perm())
	}

	dirInfo, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Stat dir failed: %v", err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("config dir permissions = %o, want 0700", dirInfo.Mode().Perm())
	}
}

func TestLoad_NoFile(t *testing.T) {
	setupTestConfig(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with no file should not error: %v", err)
	}
	if cfg.LLM.Provider != "openrouter" {
		t.Errorf("expected defaults, got provider %q", cfg.LLM.Provider)
	}
}

func TestGetAPIKey_FromEnv(t *testing.T) {
	setupTestConfig(t)
	t.Setenv("BRAIN_API_KEY", "env-key-12345")

	key := GetAPIKey()
	if key != "env-key-12345" {
		t.Errorf("expected env key, got %q", key)
	}
}

func TestGetAPIKey_FromFile(t *testing.T) {
	setupTestConfig(t)

	cfg := DefaultConfig()
	cfg.LLM.APIKey = "file-key-12345"
	Save(cfg)

	key := GetAPIKey()
	if key != "file-key-12345" {
		t.Errorf("expected file key, got %q", key)
	}
}

func TestGetAPIKey_EnvOverridesFile(t *testing.T) {
	setupTestConfig(t)

	cfg := DefaultConfig()
	cfg.LLM.APIKey = "file-key"
	Save(cfg)
	t.Setenv("BRAIN_API_KEY", "env-key")

	key := GetAPIKey()
	if key != "env-key" {
		t.Errorf("env should override file, got %q", key)
	}
}

func TestSetValue_InvalidNumeric(t *testing.T) {
	setupTestConfig(t)

	err := SetValue("analysis.max_diff_lines", "not-a-number")
	if err == nil {
		t.Error("expected error for non-numeric max_diff_lines value")
	}

	err = SetValue("daemon.max_retries", "abc")
	if err == nil {
		t.Error("expected error for non-numeric max_retries value")
	}
}

func TestSetValue_NegativeNumeric(t *testing.T) {
	setupTestConfig(t)

	err := SetValue("analysis.max_diff_lines", "-1")
	if err == nil {
		t.Error("expected error for negative max_diff_lines value")
	}

	err = SetValue("daemon.max_retries", "0")
	if err == nil {
		t.Error("expected error for zero max_retries value")
	}
}

func TestSetValue_LLMFields(t *testing.T) {
	setupTestConfig(t)

	tests := []struct {
		key   string
		value string
		check func(cfg Config) bool
	}{
		{"llm.provider", "custom-provider", func(c Config) bool { return c.LLM.Provider == "custom-provider" }},
		{"llm.api_key", "sk-test-key", func(c Config) bool { return c.LLM.APIKey == "sk-test-key" }},
		{"llm.model", "test/model", func(c Config) bool { return c.LLM.Model == "test/model" }},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := SetValue(tt.key, tt.value); err != nil {
				t.Fatalf("SetValue(%q, %q) failed: %v", tt.key, tt.value, err)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if !tt.check(cfg) {
				t.Errorf("SetValue(%q, %q) did not take effect", tt.key, tt.value)
			}
		})
	}
}

func TestSetValue_DaemonFields(t *testing.T) {
	setupTestConfig(t)

	if err := SetValue("daemon.poll_interval", "10s"); err != nil {
		t.Fatal(err)
	}
	if err := SetValue("daemon.max_retries", "5"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Daemon.PollInterval != "10s" {
		t.Errorf("expected poll_interval '10s', got %q", cfg.Daemon.PollInterval)
	}
	if cfg.Daemon.MaxRetries != 5 {
		t.Errorf("expected max_retries 5, got %d", cfg.Daemon.MaxRetries)
	}
}

func TestSetValue_AnalysisFields(t *testing.T) {
	setupTestConfig(t)

	if err := SetValue("analysis.max_diff_lines", "5000"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Analysis.MaxDiffLines != 5000 {
		t.Errorf("expected max_diff_lines 5000, got %d", cfg.Analysis.MaxDiffLines)
	}
}

func TestSetValue_InvalidKey(t *testing.T) {
	setupTestConfig(t)

	err := SetValue("invalid", "value")
	if err == nil {
		t.Error("expected error for single-part key")
	}

	err = SetValue("llm.nonexistent", "value")
	if err == nil {
		t.Error("expected error for unknown subkey")
	}

	err = SetValue("nonexistent.field", "value")
	if err == nil {
		t.Error("expected error for unknown section")
	}
}

func TestPollInterval(t *testing.T) {
	setupTestConfig(t)

	if err := SetValue("daemon.poll_interval", "30s"); err != nil {
		t.Fatal(err)
	}

	d := PollInterval()
	if d.String() != "30s" {
		t.Errorf("expected 30s, got %s", d)
	}
}

func TestConfigDir_HomeUnset(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	result := ConfigDir()
	if result == "" {
		t.Error("ConfigDir should return a reasonable fallback even when HOME is empty")
	}
}

func TestConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	expected := filepath.Join(tmpHome, ".config", "brain", "config.yaml")
	if ConfigPath() != expected {
		t.Errorf("ConfigPath() = %q, want %q", ConfigPath(), expected)
	}
}

func TestSetKey(t *testing.T) {
	setupTestConfig(t)

	if err := SetKey("sk-or-v1-newkey"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.APIKey != "sk-or-v1-newkey" {
		t.Errorf("expected 'sk-or-v1-newkey', got %q", cfg.LLM.APIKey)
	}
}
