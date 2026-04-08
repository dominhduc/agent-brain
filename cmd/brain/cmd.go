package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v0.20.2"

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
	v := version
	if v == "" {
		v = "dev"
	}
	padding := 13 - len(v)
	if padding < 0 {
		padding = 0
	}
	fmt.Printf(`+--------------------------------------------------------------+
|  agent-brain  --  AI Agent Knowledge Hub CLI  (%s)%s|
|  https://github.com/dominhduc/agent-brain                    |
+--------------------------------------------------------------+

QUICK START
  agent-brain init                 Initialize knowledge hub in current project
  agent-brain get <topic>          Retrieve knowledge (memory, gotchas, patterns, etc.)
  agent-brain add <topic> "<msg>"  Record a new learning or decision
  agent-brain search <query>       Search across all knowledge

COMMON WORKFLOWS
  Session start    agent-brain get all
  Debugging        agent-brain get gotchas
  Session end      agent-brain eval
  Record learning  agent-brain add gotcha "The fix"

FULL REFERENCE

  GET & SEARCH
    agent-brain get <topic>          Topics: memory, gotchas, patterns, decisions, architecture, all
                                     Flags: --summary (compact), --json, --focus "<topic>"
    agent-brain search <query>       Search all knowledge
                                     Flags: --json, --topic "<topic>"

  ADD & EVAL
    agent-brain add <topic> "<msg>"         Add entry to a topic
    agent-brain add <area> <topic> "<msg>"  Add entry with area tag
    agent-brain add --wm "<msg>"            Add to working memory
    agent-brain eval                        Session evaluation + handoff
                                            Flags: --good, --bad

  MAINTENANCE
    agent-brain status               Hub statistics & health
    agent-brain review               Review pending daemon entries
    agent-brain prune                Archive stale entries (--dry-run to preview)
    agent-brain sleep                Consolidate memory (decay + archive)

  CONFIG
    agent-brain config list          List all settings
    agent-brain config get <key>     Get a value
    agent-brain config set <key> <value>  Set a value
    agent-brain config reset <key>   Reset to default
    agent-brain config setup         Interactive setup wizard

  ADVANCED
    agent-brain daemon <action>      Actions: start, stop, restart, status, failed, retry, run
    agent-brain doctor               Health check & diagnostics
    agent-brain index rebuild        Rebuild metadata index
    agent-brain update               Update to latest version
    agent-brain version              Show version info

AREA TAXONOMY
  ui, backend, infrastructure, database, security, testing, architecture, general

EXAMPLES
  agent-brain init
  agent-brain get gotchas
  agent-brain search "auth" --topic "security"
  agent-brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  agent-brain eval --good
`, v, spaces(padding))
}

func spaces(n int) string {
	if n < 0 {
		n = 0
	}
	s := make([]byte, n)
	for i := range s {
		s[i] = ' '
	}
	return string(s)
}
