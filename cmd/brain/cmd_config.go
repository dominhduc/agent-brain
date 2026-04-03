package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/profile"
)

func cmdConfig() {
	if len(os.Args) < 3 {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\nWhat to do: check ~/.config/brain/config.yaml\n", err)
			os.Exit(1)
		}

		fmt.Println("Current Configuration")
		fmt.Println("=====================")
		fmt.Printf("LLM Provider:    %s\n", cfg.LLM.Provider)
		if cfg.LLM.APIKey != "" {
			fmt.Printf("API Key:         %s\n", maskKey(cfg.LLM.APIKey))
		} else {
			fmt.Println("API Key:         not set")
		}
		fmt.Printf("Model:           %s\n", cfg.LLM.Model)
		fmt.Printf("Max Diff Lines:  %d\n", cfg.Analysis.MaxDiffLines)
		fmt.Printf("Categories:      %s\n", strings.Join(cfg.Analysis.Categories, ", "))
		fmt.Printf("Poll Interval:   %s\n", cfg.Daemon.PollInterval)
		fmt.Printf("Max Retries:     %d\n", cfg.Daemon.MaxRetries)
		fmt.Printf("Retry Backoff:   %s\n", cfg.Daemon.RetryBackoff)
		fmt.Printf("Review Profile:  %s\n", cfg.Review.Profile)
		if prof, err := profile.FromName(cfg.Review.Profile); err == nil {
			fmt.Printf("  → %s\n", prof.Description())
		}
		fmt.Printf("\nConfig file:     %s\n", config.ConfigPath())
		return
	}

	if os.Args[2] == "set" {
		if len(os.Args) < 5 {
			fmt.Println("Usage: brain config set <key> <value>")
			fmt.Println("Example: brain config set llm.api_key <your-openrouter-key>")
			os.Exit(1)
		}

		key := os.Args[3]
		value := os.Args[4]

		if err := config.SetValue(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		displayValue := value
		if strings.Contains(key, "api_key") || strings.Contains(key, "apikey") {
			displayValue = maskKey(value)
		}
		fmt.Printf("Set %s = %s\n", key, displayValue)
		return
	}

	fmt.Println("Usage: brain config [set <key> <value>]")
}

func maskKey(key string) string {
	if len(key) <= 6 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-2:]
}
