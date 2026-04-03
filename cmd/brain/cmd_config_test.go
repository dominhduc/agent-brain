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
