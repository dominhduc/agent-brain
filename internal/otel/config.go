package otel

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Enabled  bool              `yaml:"enabled"`
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:  false,
		Endpoint: "",
		Headers:  make(map[string]string),
	}
}

func LoadConfig() Config {
	cfg := DefaultConfig()

	configDir, err := os.UserConfigDir()
	if err != nil {
		return cfg
	}

	configPath := filepath.Join(configDir, "brain", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg
	}

	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return cfg
	}

	if otelRaw, ok := rawConfig["otel"]; ok {
		if otelMap, ok := otelRaw.(map[string]interface{}); ok {
			if v, ok := otelMap["enabled"].(bool); ok {
				cfg.Enabled = v
			}
			if v, ok := otelMap["endpoint"].(string); ok {
				cfg.Endpoint = v
			}
			if v, ok := otelMap["headers"].(map[string]interface{}); ok {
				cfg.Headers = make(map[string]string)
				for k, val := range v {
					if s, ok := val.(string); ok {
						cfg.Headers[k] = s
					}
				}
			}
		}
	}

	return cfg
}
