package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v1.1.0"

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
|  agent-brain  --  Persistent Memory for AI Coding Agents  (%s)%s|
|  https://github.com/dominhduc/agent-brain                        |
+--------------------------------------------------------------+

WHAT IS AGENT-BRAIN?

  agent-brain gives your AI coding agent (Claude Code, OpenCode, Cursor, etc.)
  a persistent memory. It creates a .brain/ knowledge hub that:

  - Remembers project conventions, gotchas, and decisions across sessions
  - Auto-analyzes every git commit via LLM to discover patterns
  - Feeds accumulated knowledge back into every new agent session
  - Lets agents record learnings so the next session starts smarter

QUICK START

  brain init                    Initialize knowledge hub in current project
  brain get all                 Load all accumulated knowledge
  brain add <topic> "<msg>"     Record a new learning or decision
  brain search <query>          Search across all knowledge
  brain eval                    End session with self-evaluation + handoff

COMMON WORKFLOWS

  Session start     brain get all
  Before debugging  brain get gotchas
  When corrected    brain add gotcha "The fix"
  Session end       brain eval

FULL REFERENCE

  INIT & UPDATE
    brain init                 Create .brain/ hub, AGENTS.md, git hooks, daemon
    brain skill install        Install Agent Skills for coding agents
    brain skill update         Update Agent Skills to latest version
    brain update               Update agent-brain to latest version

  GET & SEARCH
    brain get <topic>          Topics: all, gotchas, patterns, decisions, architecture
                               Flags: --summary, --json, --focus "<topic>"
    brain search <query>       Search all knowledge
                               Flags: --json, --topic "<topic>"

  ADD & EVAL
    brain add <topic> "<msg>"         Add entry to a topic
    brain add <area> <topic> "<msg>"  Add entry with area tag
    brain add --wm "<msg>"            Add to working memory (temporary)
    brain eval                        Session evaluation + handoff
                                      Flags: --good, --bad

  SKILLS (for coding agents)
    brain skill list           Show installed Agent Skill locations
    brain skill diff           Compare installed vs latest templates
    brain skill update         Update Agent Skills (overwrites with confirmation)
    brain skill reflect [--dry-run]  Generate skill adaptations from usage data
    brain skill install        Install to project directories
    brain skill install --global   Install to global directories

  DAEMON
    brain daemon <action>      Actions: start, stop, restart, status, failed, retry, run
    brain review               Interactive TUI to approve/reject pending entries

  MAINTENANCE
    brain status               Hub statistics & health
    brain prune [--dry-run]    Archive stale entries
    brain dedup [--dry-run]    Remove duplicate entries
    brain sleep [--dry-run]    Consolidate memory (decay + archive)
    brain index rebuild        Rebuild metadata index
    brain doctor               Full health check & diagnostics

  CONFIG
    brain config list          List all settings
    brain config get <key>     Get a value
    brain config set <key> <value>  Set a value
    brain config reset <key>   Reset to default
    brain config setup         Interactive setup wizard
                               Config can be global (~/.config/brain/)
                               or project-specific (.brain/config.yaml)

AREA TAXONOMY (8 topics)

  ui            Frontend, styling, components, accessibility
  backend       API, services, middleware, business logic
  infrastructure Deploy, CI/CD, Docker, cloud, monitoring
  database      Schemas, migrations, queries, indexes
  security      Auth, secrets, permissions, encryption
  testing       Unit/integration/e2e tests, mocks, fixtures
  architecture  Module structure, design patterns, data flow
  general       Cross-cutting conventions, tooling, guidelines

EXAMPLES

  brain init
  brain get gotchas
  brain get all --focus "security"
  brain search "auth" --topic "security"
  brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  brain add pattern "All handlers use middleware chain: logging -> auth -> rate-limit"
  brain eval --good
  brain skill diff
  brain skill update
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
