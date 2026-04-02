package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM       LLMConfig       `yaml:"llm"`
	Analysis  AnalysisConfig  `yaml:"analysis"`
	Daemon    DaemonConfig    `yaml:"daemon"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
}

type AnalysisConfig struct {
	MaxDiffLines int      `yaml:"max_diff_lines"`
	Categories   []string `yaml:"categories"`
}

type DaemonConfig struct {
	PollInterval string `yaml:"poll_interval"`
	MaxRetries   int    `yaml:"max_retries"`
	RetryBackoff string `yaml:"retry_backoff"`
}

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "brain")
	}
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "brain")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "brain")
	}
	return filepath.Join(".", ".brain-config")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func DefaultConfig() Config {
	return Config{
		LLM: LLMConfig{
			Provider: "openrouter",
			APIKey:   "",
			Model:    "anthropic/claude-3.5-haiku",
		},
		Analysis: AnalysisConfig{
			MaxDiffLines: 2000,
			Categories:   []string{"gotchas", "patterns", "decisions", "architecture"},
		},
		Daemon: DaemonConfig{
			PollInterval: "5s",
			MaxRetries:   3,
			RetryBackoff: "exponential",
		},
	}
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	// Try env var first
	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	// Read config file
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Env var overrides file
	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	return cfg, nil
}

func Save(cfg Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func GetAPIKey() string {
	// Check env var first
	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		return key
	}

	// Check config file
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.LLM.APIKey
}

func SetKey(key string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.LLM.APIKey = key
	return Save(cfg)
}

func SetValue(dotPath, value string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}

	parts := strings.Split(dotPath, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid config key format: %s (expected section.key)", dotPath)
	}

	switch parts[0] {
	case "llm":
		switch parts[1] {
		case "provider":
			cfg.LLM.Provider = value
		case "api_key":
			cfg.LLM.APIKey = value
		case "model":
			cfg.LLM.Model = value
		default:
			return fmt.Errorf("unknown llm config key: %s", parts[1])
		}
	case "analysis":
		switch parts[1] {
		case "max_diff_lines":
			fmt.Sscanf(value, "%d", &cfg.Analysis.MaxDiffLines)
		default:
			return fmt.Errorf("unknown analysis config key: %s", parts[1])
		}
	case "daemon":
		switch parts[1] {
		case "poll_interval":
			cfg.Daemon.PollInterval = value
		case "max_retries":
			fmt.Sscanf(value, "%d", &cfg.Daemon.MaxRetries)
		case "retry_backoff":
			cfg.Daemon.RetryBackoff = value
		default:
			return fmt.Errorf("unknown daemon config key: %s", parts[1])
		}
	default:
		return fmt.Errorf("unknown config section: %s", parts[0])
	}

	return Save(cfg)
}

func PollInterval() time.Duration {
	cfg, err := Load()
	if err != nil {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(cfg.Daemon.PollInterval)
	if err != nil {
		return 5 * time.Second
	}
	return d
}
