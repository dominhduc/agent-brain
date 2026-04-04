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
	if cfg.LLM.Provider == "custom" && cfg.LLM.BaseURL != "" {
		fmt.Printf("Base URL:    %s\n", cfg.LLM.BaseURL)
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
		fmt.Println("Keys: provider, api-key, base-url, model, profile, poll-interval, max-retries, retry-backoff, max-diff-lines")
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
		fmt.Println("Keys: provider, api-key, base-url, model, profile, poll-interval, max-retries, retry-backoff, max-diff-lines")
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
		if k.Friendly == "base-url" && cfg.LLM.Provider != "custom" {
			continue
		}
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

	fmt.Println("Step 1/4: Provider")
	fmt.Println("  1. openrouter  - Aggregates 100+ models, single API key [default]")
	fmt.Println("  2. openai     - Direct OpenAI API")
	fmt.Println("  3. anthropic  - Claude models")
	fmt.Println("  4. gemini     - Google Gemini models")
	fmt.Println("  5. ollama     - Local models (requires Ollama running)")
	fmt.Println("  6. custom     - Custom OpenAI-compatible endpoint")
	fmt.Print("Choose a provider (1-6, or press Enter for default): ")
	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(providerChoice)
	fmt.Println()

	provider := "openrouter"
	switch providerChoice {
	case "2":
		provider = "openai"
	case "3":
		provider = "anthropic"
	case "4":
		provider = "gemini"
	case "5":
		provider = "ollama"
	case "6":
		provider = "custom"
	}

	needsAPIKey := provider != "ollama"
	var apiKey string
	if needsAPIKey {
		fmt.Println("Step 2/4: API Key")
		fmt.Print("Enter your API key (or press Enter to skip): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
		fmt.Println()
	}

	modelMap := map[string][]string{
		"openrouter": {"anthropic/claude-3.5-haiku", "openai/gpt-4o-mini", "google/gemini-2.5-flash"},
		"openai":     {"gpt-4o-mini", "gpt-4o", "gpt-3.5-turbo"},
		"anthropic":  {"claude-3-5-haiku-20241022", "claude-3-opus-20240229", "claude-3-sonnet-20240229"},
		"gemini":     {"gemini-2.0-flash", "gemini-1.5-flash", "gemini-1.5-pro"},
		"ollama":     {"llama3.2", "qwen2.5", "phi3"},
		"custom":    {"gpt-4o-mini", "gpt-4o", "llama3.2"},
	}

	formatWarnings := map[string]string{
		"openrouter": "should be in vendor/model format (e.g., anthropic/claude-3.5-haiku)",
		"openai":     "should start with gpt-, o1-, or o3-",
		"anthropic":  "should start with claude-",
		"gemini":     "should contain 'gemini'",
		"ollama":     "no specific format required",
		"custom":     "no specific format required",
	}

	models := modelMap[provider]
	fmt.Println("Step 3/4: Model")
	fmt.Printf("  Suggested for %s: %s\n", provider, strings.Join(models, ", "))
	fmt.Println("  Or enter any model name directly")
	fmt.Println()
	fmt.Print("Enter model name (or 1-" + fmt.Sprint(len(models)) + " to select, Enter for default): ")
	modelChoice, _ := reader.ReadString('\n')
	modelChoice = strings.TrimSpace(modelChoice)
	fmt.Println()

	model := models[0]
	if modelChoice != "" {
		idx := 0
		fmt.Sscanf(modelChoice, "%d", &idx)
		if idx >= 1 && idx <= len(models) {
			model = models[idx-1]
		} else {
			model = modelChoice
		}
	}
	if model == "" {
		model = models[0]
	}

	warning := formatWarnings[provider]
	if warning != "" && provider != "ollama" && provider != "custom" {
		valid := validateModelFormat(provider, model)
		if !valid {
			fmt.Printf("  ⚠ Warning: Model %q doesn't match typical format for %s\n", model, provider)
			fmt.Printf("    (%s)\n", warning)
			fmt.Print("  Continue anyway? (y/n): ")
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			fmt.Println()
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Setup cancelled. Run 'brain config setup' to try again.")
				os.Exit(0)
			}
		}
	}

	fmt.Println("Step 4/4: Review Profile")
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
	cfg.LLM.Provider = provider
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
	fmt.Printf("  Provider: %s\n", cfg.LLM.Provider)
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

func validateModelFormat(provider, model string) bool {
	switch provider {
	case "openrouter":
		return strings.Contains(model, "/")
	case "openai":
		return strings.HasPrefix(model, "gpt-") ||
			strings.HasPrefix(model, "o1-") ||
			strings.HasPrefix(model, "o3-") ||
			strings.HasPrefix(model, "o4-")
	case "anthropic":
		return strings.HasPrefix(model, "claude-")
	case "gemini":
		return strings.Contains(model, "gemini")
	default:
		return true
	}
}
