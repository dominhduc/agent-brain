package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v0.22.0"

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
	case "dedup":
		cmdDedup(dryRun)
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
	case "skill":
		cmdSkill(os.Args[1:])
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
	padding := 19 - len(v)
	if padding < 0 {
		padding = 0
	}
	fmt.Printf(`+--------------------------------------------------------------+
|  brain  --  AI Agent Knowledge Hub CLI  (%s)%s|
|  https://github.com/dominhduc/agent-brain                    |
+--------------------------------------------------------------+

QUICK START
  brain init                 Initialize knowledge hub in current project
  brain get <topic>          Retrieve knowledge (memory, gotchas, patterns, etc.)
  brain add <topic> "<msg>"  Record a new learning or decision
  brain search <query>       Search across all knowledge

COMMON WORKFLOWS
  Session start    brain get all
  Debugging        brain get gotchas
  Session end      brain eval
  Record learning  brain add gotcha "The fix"

FULL REFERENCE

  GET & SEARCH
    brain get <topic>          Topics: memory, gotchas, patterns, decisions, architecture, all
                               Flags: --summary (compact), --json, --focus "<topic>"
    brain search <query>       Search all knowledge
                               Flags: --json, --topic "<topic>"

  ADD & EVAL
    brain add <topic> "<msg>"         Add entry to a topic
    brain add <area> <topic> "<msg>"  Add entry with area tag
    brain add --wm "<msg>"            Add to working memory
    brain eval                        Session evaluation + handoff
                                      Flags: --good, --bad

  MAINTENANCE
    brain status               Hub statistics & health
    brain review               Review pending daemon entries
  brain prune                Archive stale entries (--dry-run to preview)
  brain dedup                Find and remove duplicate entries (--dry-run to preview)
  brain sleep                Consolidate memory (decay + archive)

  CONFIG
    brain config list          List all settings
    brain config get <key>     Get a value
    brain config set <key> <value>  Set a value
    brain config reset <key>   Reset to default
    brain config setup         Interactive setup wizard
                               Note: config can be global (~/.config/brain/)
                               or project-specific (.brain/config.yaml)

  ADVANCED
    brain daemon <action>      Actions: start, stop, restart, status, failed, retry, run
    brain doctor               Health check & diagnostics
    brain index rebuild        Rebuild metadata index
    brain update               Update to latest version
    brain skill list           List installed skill locations
    brain skill diff           Show skill updates vs templates
    brain skill update         Update skill files
    brain version              Show version info

AREA TAXONOMY
  ui, backend, infrastructure, database, security, testing, architecture, general

EXAMPLES
  brain init
  brain get gotchas
  brain search "auth" --topic "security"
  brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  brain eval --good
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
