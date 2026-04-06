package main

import (
	"fmt"
	"os"
)

var version = "v0.15.1"

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
	summaryFlag := hasFlag("--summary")
	dryRun := hasFlag("--dry-run")

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "get":
		cmdGet(jsonFlag, summaryFlag)
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
		yesFlag := hasFlag("--yes") || hasFlag("-y")
		cmdReview(allFlag, yesFlag)
	case "update":
		cmdUpdate()
	case "doctor":
		cmdDoctor()
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
  brain init                          Initialize knowledge hub in project
  brain get <topic>                   Get knowledge (memory|gotchas|patterns|decisions|architecture|all)
                                      Flags: --summary (compact view), --json
  brain search <query>                Search across all knowledge
  brain add <topic> "<message>"      Add entry to topic

  brain eval                          Create session evaluation
  brain status                        Show hub statistics
  brain review                        Review pending entries
  brain prune                         Archive stale entries

  brain config <subcommand>           Configure brain
  brain daemon <start|stop|restart|status|failed|retry|run>   Manage background daemon
  brain doctor                        Run health check
  brain version                       Show version info
  brain update                        Update to latest version

Config subcommands:
  brain config list                   List all settings
  brain config get <key>              Get a value (e.g., api-key, model)
  brain config set <key> <value>      Set a value
  brain config reset <key>            Reset to default
  brain config setup                 Interactive setup wizard

Examples:
  brain init
  brain get gotchas
  brain search "auth"
  brain add gotcha "Project uses argon2 NOT bcrypt"
  brain config list
  brain config set api-key sk-...
  brain config setup
  brain doctor`)
}
