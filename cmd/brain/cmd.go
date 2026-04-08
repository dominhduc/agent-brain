package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v0.19.0"

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
	ctx := context.Background()

	cfg := otel.LoadConfig()
	if err := otel.Init(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to init OTel: %v\n", err)
	}
	defer otel.Shutdown(ctx)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	jsonFlag := hasFlag("--json")
	summaryFlag := hasFlag("--summary")
	dryRun := hasFlag("--dry-run")
	cmdName := os.Args[1]

	start := time.Now()
	ctx, span := otel.StartSpan(ctx, "brain."+cmdName)
	defer func() {
		otel.SetAttributes(span,
			otel.BrainCommand.String(cmdName),
			otel.BrainVersion.String(version),
			otel.BrainDurationMs.Int64(time.Since(start).Milliseconds()),
		)
		span.End()
	}()

	switch cmdName {
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
	case "sleep":
		cmdSleep(dryRun)
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
		ttyFlag := hasFlag("--tty")
		cmdReview(allFlag, yesFlag, ttyFlag)
	case "update":
		cmdUpdate()
	case "doctor":
		cmdDoctor()
	case "index":
		cmdIndex()
	case "wm":
		cmdWM()
	case "handoff":
		cmdHandoff()
	case "outcome":
		cmdOutcome()
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
                                      Flags: --summary (compact view), --json, --focus "<topic>"
  brain search <query>                Search across all knowledge
                                      Flags: --json, --topic "<topic>"
  brain add <topic> "<message>"       Add entry to topic
  brain add <8-topic> <topic> "<msg>" Add entry with topic tag
  brain add --wm "<message>"          Add to working memory

  brain eval                          Create session evaluation + handoff
  brain eval --good                   Apply positive outcome + flush WM
  brain eval --bad                    Apply negative outcome + flush WM
  brain status                        Show hub statistics
  brain review                        Review pending entries
  brain prune                         Archive stale entries
  brain sleep                         Consolidate memory (decay + archive)

  brain config <subcommand>           Configure brain
  brain wm push|read|clear|flush      Working memory (deprecated: use 'brain add --wm')
  brain handoff create|latest|resume  Session handoffs (deprecated: use 'brain eval')
  brain outcome --good|--bad          Feedback on retrieved memories (deprecated: use 'brain eval')
  brain daemon <start|stop|restart|status|failed|retry|run>   Manage background daemon
  brain doctor                        Run health check
  brain index rebuild                 Rebuild metadata index
  brain version                       Show version info
  brain update                        Update to latest version

8-Topic taxonomy: ui, backend, infrastructure, database, security, testing, architecture, general

Config subcommands:
  brain config list                   List all settings
  brain config get <key>              Get a value (e.g., api-key, model)
  brain config set <key> <value>      Set a value
  brain config reset <key>            Reset to default
  brain config setup                 Interactive setup wizard

Examples:
  brain init
  brain get gotchas
  brain get all --focus "infrastructure"
  brain search "auth"
  brain search "auth" --topic "security"
  brain add gotcha "Project uses argon2 NOT bcrypt"
  brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  brain add --wm "investigating auth bug"
  brain eval --good
  brain config list
  brain config set api-key sk-...
  brain config setup
  brain doctor`)
}
