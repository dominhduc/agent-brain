package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dominhduc/agent-brain/internal/knowledge"
	"github.com/dominhduc/agent-brain/internal/config"
	"github.com/dominhduc/agent-brain/internal/service"
)

func cmdDoctor() {
	checks := []func() (string, bool, string){
		checkVersion,
		checkGit,
		checkBrainDir,
		checkConfig,
		checkDaemon,
		checkPreflight,
	}

	fmt.Println("brain doctor - Health Check")
	fmt.Println("===========================")
	fmt.Println()

	allPassed := true
	for _, check := range checks {
		name, passed, detail := check()
		status := "✓"
		if !passed {
			status = "✗"
			allPassed = false
		}
		fmt.Printf("  %s %s", status, name)
		if detail != "" {
			fmt.Printf(" — %s", detail)
		}
		fmt.Println()
	}

	fmt.Println()
	if allPassed {
		fmt.Println("All checks passed!")
	} else {
		fmt.Println("Some checks failed. See details above.")
		os.Exit(1)
	}
}

func checkVersion() (string, bool, string) {
	return fmt.Sprintf("Version: %s", version), true, fmt.Sprintf("OS: %s/%s", runtime.GOOS, runtime.GOARCH)
}

func checkGit() (string, bool, string) {
	_, err := exec.LookPath("git")
	if err != nil {
		return "Git", false, "not found in PATH"
	}
	out, _ := exec.Command("git", "--version").Output()
	return "Git", true, strings.TrimSpace(string(out))
}

func checkBrainDir() (string, bool, string) {
	cwd, _ := os.Getwd()
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		return "Project .brain/", false, fmt.Sprintf("not found (cwd: %s)", cwd)
	}
	entries, _ := os.ReadDir(brainDir)
	return "Project .brain/", true, fmt.Sprintf("found at %s (%d entries)", brainDir, len(entries))
}

func checkConfig() (string, bool, string) {
	cfg, err := config.Load()
	if err != nil {
		return "Config", false, fmt.Sprintf("error: %v", err)
	}
	provider := cfg.LLM.Provider
	model := cfg.LLM.Model
	if cfg.LLM.APIKey == "" {
		return "Config", false, fmt.Sprintf("provider: %s, model: %s, API key: not set", provider, model)
	}
	return "Config", true, fmt.Sprintf("provider: %s, model: %s", provider, model)
}

func checkPreflight() (string, bool, string) {
	_, err := exec.LookPath("git")
	if err != nil {
		return "Preflight", false, "git required but not found"
	}
	return "Preflight", true, "all dependencies available"
}

func checkDaemon() (string, bool, string) {
	brainDir, err := knowledge.FindBrainDir()
	if err != nil {
		return "Daemon", false, "no project .brain/ found"
	}
	workDir := filepath.Dir(brainDir)
	running := service.IsRunning(workDir)
	if running {
		return "Daemon", true, "running"
	}
	return "Daemon", false, "not running (run 'brain daemon start')"
}
