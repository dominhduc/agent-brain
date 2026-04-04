package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dominhduc/agent-brain/internal/provider"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM           LLMConfig                 `yaml:"llm"`
	Analysis      AnalysisConfig            `yaml:"analysis"`
	Daemon        DaemonConfig              `yaml:"daemon"`
	Review        ReviewConfig              `yaml:"review"`
	CustomProviders map[string]CustomProviderConfig `yaml:"custom_providers,omitempty"`
}

type CustomProviderConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

type ReviewConfig struct {
	Profile string `yaml:"profile"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url"`
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
			BaseURL:  "",
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
		Review: ReviewConfig{
			Profile: "guard",
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
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for max_diff_lines: %q is not a number", value)
			}
			if n < 100 {
				return fmt.Errorf("max_diff_lines must be at least 100, got %d", n)
			}
			cfg.Analysis.MaxDiffLines = n
		default:
			return fmt.Errorf("unknown analysis config key: %s", parts[1])
		}
	case "daemon":
		switch parts[1] {
		case "poll_interval":
			d, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration for poll_interval: %q", value)
			}
			if d < time.Second {
				return fmt.Errorf("poll_interval must be at least 1s, got %s", d)
			}
			cfg.Daemon.PollInterval = value
		case "max_retries":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for max_retries: %q is not a number", value)
			}
			if n < 1 {
				return fmt.Errorf("max_retries must be at least 1, got %d", n)
			}
			cfg.Daemon.MaxRetries = n
		case "retry_backoff":
			cfg.Daemon.RetryBackoff = value
		default:
			return fmt.Errorf("unknown daemon config key: %s", parts[1])
		}
	case "review":
		switch parts[1] {
		case "profile":
			valid := map[string]bool{"guard": true, "assist": true, "agent": true}
			if !valid[value] {
				return fmt.Errorf("invalid profile %q. Valid profiles: guard, assist, agent", value)
			}
			cfg.Review.Profile = value
		default:
			return fmt.Errorf("unknown review config key: %s", parts[1])
		}
	default:
		return fmt.Errorf("unknown config section: %s", parts[0])
	}

	return Save(cfg)
}

func GetCustomProvider(name string) (*CustomProviderConfig, bool) {
	cfg, err := Load()
	if err != nil {
		return nil, false
	}
	cp, ok := cfg.CustomProviders[name]
	if !ok {
		return nil, false
	}
	return &cp, true
}

func IsCustomProvider(name string) bool {
	if provider.IsValid(name) {
		return false
	}
	_, ok := GetCustomProvider(name)
	return ok
}

func SaveCustomProvider(name string, cp CustomProviderConfig) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if cfg.CustomProviders == nil {
		cfg.CustomProviders = make(map[string]CustomProviderConfig)
	}
	cfg.CustomProviders[name] = cp
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

func GetValue(dotPath string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	parts := strings.Split(dotPath, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid config key format: %s", dotPath)
	}

	switch parts[0] {
	case "llm":
		switch parts[1] {
		case "provider":
			return cfg.LLM.Provider, nil
		case "api_key":
			return cfg.LLM.APIKey, nil
		case "model":
			return cfg.LLM.Model, nil
		default:
			return "", fmt.Errorf("unknown llm config key: %s", parts[1])
		}
	case "analysis":
		switch parts[1] {
		case "max_diff_lines":
			return strconv.Itoa(cfg.Analysis.MaxDiffLines), nil
		default:
			return "", fmt.Errorf("unknown analysis config key: %s", parts[1])
		}
	case "daemon":
		switch parts[1] {
		case "poll_interval":
			return cfg.Daemon.PollInterval, nil
		case "max_retries":
			return strconv.Itoa(cfg.Daemon.MaxRetries), nil
		case "retry_backoff":
			return cfg.Daemon.RetryBackoff, nil
		default:
			return "", fmt.Errorf("unknown daemon config key: %s", parts[1])
		}
	case "review":
		switch parts[1] {
		case "profile":
			return cfg.Review.Profile, nil
		default:
			return "", fmt.Errorf("unknown review config key: %s", parts[1])
		}
	default:
		return "", fmt.Errorf("unknown config section: %s", parts[0])
	}
}
