package main

import (
	"fmt"
	"os"
)

var version = "v0.4.0"

var (
	commit string
	date   string
)

const (
	maxMessageLen = 10240
	maxDiffChars  = 200000
	maxPerCycle   = 5
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	jsonFlag := hasFlag("--json")
	dryRun := hasFlag("--dry-run")

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "get":
		cmdGet(jsonFlag)
	case "search":
		cmdSearch(jsonFlag)
	case "add":
		cmdAdd()
	case "eval":
		cmdEval()
	case "prune":
		cmdPrune(dryRun)
	case "status":
		cmdStatus(jsonFlag)
	case "daemon":
		cmdDaemon()
	case "config":
		cmdConfig()
	case "version", "--version", "-v":
		cmdVersion()
	case "review":
		allFlag := hasFlag("--all")
		cmdReview(allFlag)
	case "update":
		cmdUpdate()
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[2:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Println(`brain - AI Agent Knowledge Hub CLI

Usage:
  brain init                          Initialize knowledge hub in current project
  brain get <topic> [--json]          Get topic content (memory, gotchas, patterns, decisions, architecture, all)
  brain search <query> [--json]       Search across all knowledge files
  brain add <topic> "<message>"       Add knowledge entry to a topic
  brain eval                          Create session evaluation file
  brain prune [--dry-run]             Archive stale knowledge entries
  brain status [--json]               Show knowledge hub statistics
  brain review [--all]                 Review and approve pending knowledge entries
  brain daemon start|stop|status      Manage background daemon
  brain config [set <key> <value>]    View or set configuration
  brain version                       Show version and build info
  brain update                       Self-update to latest release

Topics: memory, gotchas, patterns, decisions, architecture, all

Examples:
  brain init
  brain get gotchas
  brain search "auth"
  brain add gotcha "Project uses argon2, NOT bcrypt"
  brain eval
  brain status
  brain daemon status
  brain config set llm.api_key <your-openrouter-key>`)
}
