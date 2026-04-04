package config

import (
	"fmt"
	"strconv"
	"time"
)

type ConfigKey struct {
	Friendly    string
	DotPath     string
	Type        string // "string", "duration", "int", "enum"
	Default     string
	Description string
	EnvVar      string
	Options     []string // for enum types
}

var allKeys = []ConfigKey{
	{
		Friendly:    "provider",
		DotPath:     "llm.provider",
		Type:        "enum",
		Default:     "openrouter",
		Description: "LLM provider",
		Options:     []string{"openrouter", "openai", "anthropic", "gemini", "ollama", "custom"},
	},
	{
		Friendly:    "api-key",
		DotPath:     "llm.api_key",
		Type:        "string",
		Default:     "",
		Description: "API key",
		EnvVar:      "BRAIN_API_KEY",
	},
	{
		Friendly:    "base-url",
		DotPath:     "llm.base_url",
		Type:        "string",
		Default:     "",
		Description: "Custom provider base URL (only for provider=custom)",
	},
	{
		Friendly:    "model",
		DotPath:     "llm.model",
		Type:        "string",
		Default:     "anthropic/claude-3.5-haiku",
		Description: "LLM model name",
	},
	{
		Friendly:    "profile",
		DotPath:     "review.profile",
		Type:        "enum",
		Default:     "guard",
		Description: "Review profile (guard/assist/agent)",
		Options:     []string{"guard", "assist", "agent"},
	},
	{
		Friendly:    "poll-interval",
		DotPath:     "daemon.poll_interval",
		Type:        "duration",
		Default:     "5s",
		Description: "Daemon poll interval",
	},
	{
		Friendly:    "max-retries",
		DotPath:     "daemon.max_retries",
		Type:        "int",
		Default:     "3",
		Description: "Daemon max retries",
	},
	{
		Friendly:    "retry-backoff",
		DotPath:     "daemon.retry_backoff",
		Type:        "string",
		Default:     "exponential",
		Description: "Daemon retry backoff",
	},
	{
		Friendly:    "max-diff-lines",
		DotPath:     "analysis.max_diff_lines",
		Type:        "int",
		Default:     "2000",
		Description: "Max diff lines for analysis",
	},
}

func AllKeys() []ConfigKey {
	return allKeys
}

func GetKeyByFriendly(friendly string) *ConfigKey {
	for i := range allKeys {
		if allKeys[i].Friendly == friendly {
			return &allKeys[i]
		}
	}
	return nil
}

func GetKeyByDotPath(dotPath string) *ConfigKey {
	for i := range allKeys {
		if allKeys[i].DotPath == dotPath {
			return &allKeys[i]
		}
	}
	return nil
}

func ResolveKey(input string) (*ConfigKey, error) {
	if key := GetKeyByFriendly(input); key != nil {
		return key, nil
	}
	if key := GetKeyByDotPath(input); key != nil {
		return key, nil
	}
	return nil, fmt.Errorf("unknown config key: %q. Run 'brain config list' to see available keys", input)
}

func (k *ConfigKey) Validate(value string) error {
	switch k.Type {
	case "duration":
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("invalid duration %q for %s", value, k.Friendly)
		}
	case "int":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid number %q for %s", value, k.Friendly)
		}
		if n < 1 {
			return fmt.Errorf("%s must be at least 1, got %d", k.Friendly, n)
		}
		if k.Friendly == "max-diff-lines" && n < 100 {
			return fmt.Errorf("max-diff-lines must be at least 100, got %d", n)
		}
	case "enum":
		valid := false
		for _, opt := range k.Options {
			if value == opt {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value %q for %s. Valid options: %v", value, k.Friendly, k.Options)
		}
	}
	return nil
}

func (k *ConfigKey) ApplyValue(cfg *Config, value string) error {
	if err := k.Validate(value); err != nil {
		return err
	}

	switch k.DotPath {
	case "llm.api_key":
		cfg.LLM.APIKey = value
	case "llm.model":
		cfg.LLM.Model = value
	case "llm.provider":
		cfg.LLM.Provider = value
	case "llm.base_url":
		cfg.LLM.BaseURL = value
	case "review.profile":
		cfg.Review.Profile = value
	case "daemon.poll_interval":
		cfg.Daemon.PollInterval = value
	case "daemon.max_retries":
		n, _ := strconv.Atoi(value)
		cfg.Daemon.MaxRetries = n
	case "daemon.retry_backoff":
		cfg.Daemon.RetryBackoff = value
	case "analysis.max_diff_lines":
		n, _ := strconv.Atoi(value)
		cfg.Analysis.MaxDiffLines = n
	default:
		return fmt.Errorf("cannot apply value to %s", k.DotPath)
	}
	return nil
}

func (k *ConfigKey) GetValue(cfg *Config) string {
	switch k.DotPath {
	case "llm.api_key":
		return cfg.LLM.APIKey
	case "llm.model":
		return cfg.LLM.Model
	case "llm.provider":
		return cfg.LLM.Provider
	case "llm.base_url":
		return cfg.LLM.BaseURL
	case "review.profile":
		return cfg.Review.Profile
	case "daemon.poll_interval":
		return cfg.Daemon.PollInterval
	case "daemon.max_retries":
		return strconv.Itoa(cfg.Daemon.MaxRetries)
	case "daemon.retry_backoff":
		return cfg.Daemon.RetryBackoff
	case "analysis.max_diff_lines":
		return strconv.Itoa(cfg.Analysis.MaxDiffLines)
	default:
		return ""
	}
}
