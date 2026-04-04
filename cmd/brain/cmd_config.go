package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/profile"
)

func cmdConfig() {
	if len(os.Args) < 3 {
		cmdConfigShow()
		return
	}

	switch os.Args[2] {
	case "get":
		cmdConfigGet()
	case "set":
		cmdConfigSet()
	case "list":
		cmdConfigList()
	case "reset":
		cmdConfigReset()
	case "setup":
		cmdConfigSetup()
	default:
		if len(os.Args) >= 5 {
			os.Args = append([]string{os.Args[0], "config", "set"}, os.Args[2:]...)
			cmdConfigSet()
			return
		}
		fmt.Println("Usage: brain config <subcommand> [args...]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  brain config           Show current configuration")
		fmt.Println("  brain config list      List all settings")
		fmt.Println("  brain config get <key> Get a value")
		fmt.Println("  brain config set <key> <value> Set a value")
		fmt.Println("  brain config reset <key> Reset to default")
		fmt.Println("  brain config setup     Interactive setup wizard")
	}
}

func cmdConfigShow() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\nWhat to do: check ~/.config/brain/config.yaml\n", err)
		os.Exit(1)
	}

	fmt.Println("Current Configuration")
	fmt.Println("=====================")
	fmt.Printf("Provider:    %s\n", cfg.LLM.Provider)
	fmt.Printf("Model:       %s\n", cfg.LLM.Model)
	if cfg.LLM.APIKey != "" {
		fmt.Printf("API Key:     %s\n", maskKey(cfg.LLM.APIKey))
	} else {
		fmt.Println("API Key:     not set")
	}
	fmt.Printf("Profile:     %s\n", cfg.Review.Profile)
	if prof, err := profile.FromName(cfg.Review.Profile); err == nil {
		fmt.Printf("  → %s\n", prof.Description())
	}
	fmt.Printf("Poll Interval:  %s\n", cfg.Daemon.PollInterval)
	fmt.Printf("Max Retries:    %d\n", cfg.Daemon.MaxRetries)
	fmt.Printf("Retry Backoff:  %s\n", cfg.Daemon.RetryBackoff)
	fmt.Printf("Max Diff Lines: %d\n", cfg.Analysis.MaxDiffLines)
	fmt.Printf("Categories:     %s\n", strings.Join(cfg.Analysis.Categories, ", "))
	fmt.Printf("\nConfig file: %s\n", config.ConfigPath())
}

func cmdConfigGet() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: brain config get <key>")
		fmt.Println("Keys: api-key, model, provider, profile, poll-interval, max-retries, retry-backoff, max-diff-lines")
		os.Exit(1)
	}

	key, err := config.ResolveKey(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	value := key.GetValue(&cfg)
	if key.Friendly == "api-key" {
		if value == "" {
			fmt.Println("not set")
		} else {
			fmt.Println(maskKey(value))
		}
	} else {
		fmt.Println(value)
	}
}

func cmdConfigSet() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: brain config set <key> <value>")
		fmt.Println("Keys: api-key, model, provider, profile, poll-interval, max-retries, retry-backoff, max-diff-lines")
		os.Exit(1)
	}

	keyInput := os.Args[3]
	value := os.Args[4]

	key, err := config.ResolveKey(keyInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := key.ApplyValue(&cfg, value); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	displayValue := value
	if key.Friendly == "api-key" {
		displayValue = maskKey(value)
	}
	fmt.Printf("Set %s = %s\n", key.Friendly, displayValue)
}

func cmdConfigList() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration Keys")
	fmt.Println("==================")
	for _, k := range config.AllKeys() {
		value := k.GetValue(&cfg)
		current := value
		if current == "" {
			current = "not set"
		}
		if k.Friendly == "api-key" && value != "" {
			current = maskKey(value)
		}
		fmt.Printf("  %-16s %-32s current: %s\n", k.Friendly, k.Description, current)
	}
}

func cmdConfigReset() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	defaultCfg := config.DefaultConfig()

	if len(os.Args) < 4 {
		if err := config.Save(defaultCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All configuration values reset to defaults.")
		return
	}

	keyInput := os.Args[3]
	key, err := config.ResolveKey(keyInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	defaultKey := config.GetKeyByFriendly(key.Friendly)
	if defaultKey == nil {
		fmt.Fprintf(os.Stderr, "Error: could not find default for %s\n", key.Friendly)
		os.Exit(1)
	}

	if err := key.ApplyValue(&cfg, defaultKey.Default); err != nil {
		fmt.Fprintf(os.Stderr, "Error resetting %s: %v\n", key.Friendly, err)
		os.Exit(1)
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Reset %s to default: %s\n", key.Friendly, defaultKey.Default)
}

func cmdConfigSetup() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Setting up brain configuration")
	fmt.Println("===============================")
	fmt.Println()

	fmt.Println("Step 1/3: API Key")
	fmt.Print("Enter your OpenRouter API key (or press Enter to skip): ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	fmt.Println()

	fmt.Println("Step 2/3: Model")
	fmt.Println("  1. anthropic/claude-3.5-haiku (fast, cheap ~$0.01/commit) [default]")
	fmt.Println("  2. openai/gpt-4o-mini")
	fmt.Println("  3. google/gemini-2.5-flash")
	fmt.Println("  4. Custom model name")
	fmt.Print("Choose a model (1-4, or press Enter for default): ")
	modelChoice, _ := reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)
	fmt.Println()

	model := "anthropic/claude-3.5-haiku"
	switch modelChoice {
	case "2":
		model = "openai/gpt-4o-mini"
	case "3":
		model = "google/gemini-2.5-flash"
	case "4":
		fmt.Print("Enter custom model name: ")
		customModel, _ := reader.ReadString('\n')
		model = strings.TrimSpace(customModel)
		if model == "" {
			model = "anthropic/claude-3.5-haiku"
		}
	}

	fmt.Println("Step 3/3: Review Profile")
	fmt.Println("  1. guard   - Review every entry (recommended for new projects) [default]")
	fmt.Println("  2. assist  - Auto-deduplicate, but review each unique entry")
	fmt.Println("  3. agent   - Fully automatic, no review needed")
	fmt.Print("Choose a profile (1-3, or press Enter for default): ")
	profileChoice, _ := reader.ReadString('\n')
	profileChoice = strings.TrimSpace(profileChoice)
	fmt.Println()

	prof := "guard"
	switch profileChoice {
	case "2":
		prof = "assist"
	case "3":
		prof = "agent"
	}

	cfg := config.DefaultConfig()
	if apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	cfg.LLM.Model = model
	cfg.Review.Profile = prof

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration saved!")
	fmt.Printf("  API Key:  %s\n", maskKeyOrNotSet(cfg.LLM.APIKey))
	fmt.Printf("  Model:    %s\n", cfg.LLM.Model)
	fmt.Printf("  Profile:  %s\n", cfg.Review.Profile)
	fmt.Println()
	fmt.Println("Run 'brain config list' to see all settings.")
	fmt.Println("Run 'brain init' to initialize your project.")
}

func maskKeyOrNotSet(key string) string {
	if key == "" {
		return "not set"
	}
	return maskKey(key)
}

func maskKey(key string) string {
	if len(key) <= 6 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-2:]
}
