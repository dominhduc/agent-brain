package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dominhduc/agent-brain/internal/provider"
	"gopkg.in/yaml.v3"
)

var (
	configCache     *Config
	configCacheTime time.Time
	configCacheMu   sync.RWMutex
	configCacheTTL  = 5 * time.Second
)

func strictUnmarshal(data []byte, v interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(v)
}

type Config struct {
	LLM           LLMConfig                 `yaml:"llm"`
	Analysis      AnalysisConfig            `yaml:"analysis"`
	Daemon        DaemonConfig              `yaml:"daemon"`
	Review        ReviewConfig              `yaml:"review"`
	Retrieval     RetrievalConfig           `yaml:"retrieval,omitempty"`
	CustomProviders map[string]CustomProviderConfig `yaml:"custom_providers,omitempty"`
}

type RetrievalConfig struct {
	MaxTokens       int     `yaml:"max_tokens"`
	MinStrength     float64 `yaml:"min_strength"`
	MaxEntries      int     `yaml:"max_entries"`
	IncludeRecentDays int   `yaml:"include_recent_days"`
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

func ProjectConfigPath(brainDir string) string {
	return filepath.Join(brainDir, "config.yaml")
}

func ProjectConfigExists(brainDir string) bool {
	path := ProjectConfigPath(brainDir)
	_, err := os.Stat(path)
	return err == nil
}

func GlobalConfigExists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

type ConfigSource string

const (
	SourceProject ConfigSource = "project"
	SourceGlobal  ConfigSource = "global"
	SourceDefault ConfigSource = "default"
)

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
		Retrieval: RetrievalConfig{
			MaxTokens:       3000,
			MinStrength:     0.15,
			MaxEntries:      50,
			IncludeRecentDays: 7,
		},
	}
}

func Load() (Config, error) {
	configCacheMu.RLock()
	if configCache != nil && time.Since(configCacheTime) < configCacheTTL {
		cfg := *configCache
		configCacheMu.RUnlock()
		if key := os.Getenv("BRAIN_API_KEY"); key != "" {
			cfg.LLM.APIKey = key
		}
		return cfg, nil
	}
	configCacheMu.RUnlock()

	cfg := DefaultConfig()

	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			configCacheMu.Lock()
			configCache = &cfg
			configCacheTime = time.Now()
			configCacheMu.Unlock()
			return cfg, nil
		}
		fmt.Fprintf(os.Stderr, "Warning: could not read config file: %v\n", err)
		configCacheMu.Lock()
		configCache = &cfg
		configCacheTime = time.Now()
		configCacheMu.Unlock()
		return cfg, nil
	}

	if err := strictUnmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unknown fields in config file: %v\n", err)
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not parse config file: %v\n", err)
		}
	}

	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	configCacheMu.Lock()
	configCache = &cfg
	configCacheTime = time.Now()
	configCacheMu.Unlock()

	return cfg, nil
}

func LoadForProject(brainDir string) (Config, error) {
	cfg := DefaultConfig()

	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	path := ProjectConfigPath(brainDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		fmt.Fprintf(os.Stderr, "Warning: could not read project config file: %v\n", err)
		return cfg, nil
	}

	if err := strictUnmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unknown fields in project config file: %v\n", err)
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not parse project config file: %v\n", err)
		}
	}

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

	configCacheMu.Lock()
	configCache = &cfg
	configCacheTime = time.Now()
	configCacheMu.Unlock()

	return nil
}

func SaveToProject(cfg Config, brainDir string) error {
	dir := brainDir
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create .brain config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := ProjectConfigPath(brainDir)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write project config file: %w", err)
	}

	return nil
}

func SaveToPath(cfg Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	configCacheMu.Lock()
	configCache = &cfg
	configCacheTime = time.Now()
	configCacheMu.Unlock()

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
	case "retrieval":
		switch parts[1] {
		case "max_tokens":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for max_tokens: %q is not a number", value)
			}
			if n < 100 {
				return fmt.Errorf("max_tokens must be at least 100, got %d", n)
			}
			cfg.Retrieval.MaxTokens = n
		case "min_strength":
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid value for min_strength: %q is not a number", value)
			}
			if f < 0 || f > 1 {
				return fmt.Errorf("min_strength must be between 0 and 1, got %.2f", f)
			}
			cfg.Retrieval.MinStrength = f
		case "max_entries":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for max_entries: %q is not a number", value)
			}
			if n < 1 {
				return fmt.Errorf("max_entries must be at least 1, got %d", n)
			}
			cfg.Retrieval.MaxEntries = n
		case "include_recent_days":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid value for include_recent_days: %q is not a number", value)
			}
			if n < 1 {
				return fmt.Errorf("include_recent_days must be at least 1, got %d", n)
			}
			cfg.Retrieval.IncludeRecentDays = n
		default:
			return fmt.Errorf("unknown retrieval config key: %s", parts[1])
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
			masked := cfg.LLM.APIKey
			if len(masked) > 8 {
				masked = masked[:4] + "****" + masked[len(masked)-2:]
			} else if masked != "" {
				masked = "****"
			}
			return masked, nil
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
	case "retrieval":
		switch parts[1] {
		case "max_tokens":
			return strconv.Itoa(cfg.Retrieval.MaxTokens), nil
		case "min_strength":
			return strconv.FormatFloat(cfg.Retrieval.MinStrength, 'f', 2, 64), nil
		case "max_entries":
			return strconv.Itoa(cfg.Retrieval.MaxEntries), nil
		case "include_recent_days":
			return strconv.Itoa(cfg.Retrieval.IncludeRecentDays), nil
		default:
			return "", fmt.Errorf("unknown retrieval config key: %s", parts[1])
		}
	default:
		return "", fmt.Errorf("unknown config section: %s", parts[0])
	}
}
