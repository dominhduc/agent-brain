package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dominhduc/agent-brain/internal/config"
)

func TestCmdConfig_ShowConfig(t *testing.T) {
	tmpDir := setupTestProject(t)

	configDir := filepath.Join(tmpDir, "test-config")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "test-api-key"
	config.Save(cfg)

	oldArgs := os.Args
	os.Args = []string{"brain", "config"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdConfig() })

	if !strings.Contains(output, "Current Configuration") {
		t.Errorf("expected config header, got: %s", output)
	}
	if !strings.Contains(output, "test****ey") {
		t.Errorf("expected masked API key, got: %s", output)
	}
	if strings.Contains(output, "test-api-key") {
		t.Errorf("API key should be masked, but got full key: %s", output)
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "****"},
		{"abcdef", "****"},
		{"sk-or-v1-abcdefghijklmnop", "sk-o****op"},
	}

	for _, tt := range tests {
		result := maskKey(tt.input)
		if result != tt.expected {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCmdConfig_Get(t *testing.T) {
	tmpDir := setupTestProject(t)

	configDir := filepath.Join(tmpDir, "test-config")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "test-api-key"
	config.Save(cfg)

	oldArgs := os.Args
	os.Args = []string{"brain", "config", "get", "api-key"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdConfig() })

	if !strings.Contains(output, "test") {
		t.Errorf("expected masked API key in output, got: %s", output)
	}
}

func TestCmdConfig_List(t *testing.T) {
	tmpDir := setupTestProject(t)

	configDir := filepath.Join(tmpDir, "test-config")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cfg := config.DefaultConfig()
	config.Save(cfg)

	oldArgs := os.Args
	os.Args = []string{"brain", "config", "list"}
	defer func() { os.Args = oldArgs }()

	output := captureStdout(func() { cmdConfig() })

	if !strings.Contains(output, "api-key") {
		t.Errorf("expected 'api-key' in list output, got: %s", output)
	}
	if !strings.Contains(output, "model") {
		t.Errorf("expected 'model' in list output, got: %s", output)
	}
}

func TestCmdConfig_Set_FriendlyKey(t *testing.T) {
	tmpDir := setupTestProject(t)

	configDir := filepath.Join(tmpDir, "test-config")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	oldArgs := os.Args
	os.Args = []string{"brain", "config", "set", "model", "test/model"}
	defer func() { os.Args = oldArgs }()

	cmdConfig()

	cfg, _ := config.Load()
	if cfg.LLM.Model != "test/model" {
		t.Errorf("expected model 'test/model', got %q", cfg.LLM.Model)
	}
}

func TestCmdConfig_Set_DotNotation(t *testing.T) {
	tmpDir := setupTestProject(t)

	configDir := filepath.Join(tmpDir, "test-config")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	oldArgs := os.Args
	os.Args = []string{"brain", "config", "set", "llm.model", "dot/model"}
	defer func() { os.Args = oldArgs }()

	cmdConfig()

	cfg, _ := config.Load()
	if cfg.LLM.Model != "dot/model" {
		t.Errorf("expected model 'dot/model', got %q", cfg.LLM.Model)
	}
}

func TestCmdConfig_Reset(t *testing.T) {
	tmpDir := setupTestProject(t)

	configDir := filepath.Join(tmpDir, "test-config")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	cfg := config.DefaultConfig()
	cfg.LLM.Model = "custom-model"
	config.Save(cfg)

	oldArgs := os.Args
	os.Args = []string{"brain", "config", "reset", "model"}
	defer func() { os.Args = oldArgs }()

	cmdConfig()

	cfg, _ = config.Load()
	if cfg.LLM.Model != "anthropic/claude-3.5-haiku" {
		t.Errorf("expected default model after reset, got %q", cfg.LLM.Model)
	}
}
