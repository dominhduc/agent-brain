package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v1.4.2"

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
		printUsage(false)
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
		cmdGet(jsonFlag, summaryFlag, hasFlag("--compact"), hasFlag("--message-only"), hasFlag("--full"))
	case "add":
		cmdAdd()
	case "clean":
		cmdClean(dryRun, hasFlag("--fuzzy"), hasFlag("--patterns"), hasFlag("--duplicates"), hasFlag("--decay"), hasFlag("--rebuild"))
	case "doctor":
		cmdDoctor(jsonFlag, hasFlag("--fix"))
	case "daemon":
		cmdDaemon()
	case "config":
		cmdConfig()
	case "version", "--version", "-v":
		cmdVersion()
	case "update":
		cmdUpdate()
	case "--help", "-h", "help":
		printUsage(hasFlag("--full"))

	// Backward compatibility aliases
	case "search":
		fmt.Fprintln(os.Stderr, "Note: 'brain search' merged into 'brain get'. Use: brain get <query>")
		cmdGetSearch(os.Args[2], jsonFlag)
	case "eval":
		fmt.Fprintln(os.Stderr, "Note: 'brain eval' merged into 'brain add --eval'. Use: brain add --eval")
		os.Args = append([]string{"brain", "add", "--eval"}, os.Args[2:]...)
		cmdAdd()
	case "prune":
		fmt.Fprintln(os.Stderr, "Note: 'brain prune' merged into 'brain clean'. Use: brain clean --patterns")
		os.Args = append([]string{"brain", "clean", "--patterns"}, os.Args[2:]...)
		cmdClean(dryRun, false, true, false, false, false)
	case "dedup":
		fmt.Fprintln(os.Stderr, "Note: 'brain dedup' merged into 'brain clean'. Use: brain clean --duplicates")
		os.Args = append([]string{"brain", "clean", "--duplicates"}, os.Args[2:]...)
		cmdClean(dryRun, hasFlag("--fuzzy"), false, true, false, false)
	case "sleep":
		fmt.Fprintln(os.Stderr, "Note: 'brain sleep' merged into 'brain clean'. Use: brain clean --decay")
		os.Args = append([]string{"brain", "clean", "--decay"}, os.Args[2:]...)
		cmdClean(dryRun, false, false, false, true, false)
	case "status":
		fmt.Fprintln(os.Stderr, "Note: 'brain status' merged into 'brain doctor'. Use: brain doctor")
		cmdDoctor(jsonFlag, false)
	case "review":
		fmt.Fprintln(os.Stderr, "Note: 'brain review' merged into 'brain daemon review'. Use: brain daemon review")
		os.Args = append([]string{"brain", "daemon", "review"}, os.Args[2:]...)
		cmdDaemon()
	case "index":
		fmt.Fprintln(os.Stderr, "Note: 'brain index rebuild' merged into 'brain clean'. Use: brain clean --rebuild")
		os.Args = append([]string{"brain", "clean", "--rebuild"}, os.Args[2:]...)
		cmdClean(false, false, false, false, false, true)
	case "wm":
		fmt.Fprintln(os.Stderr, "Note: 'brain wm' is deprecated. Use 'brain add --wm'.")
		os.Args = append([]string{"brain", "add", "--wm"}, os.Args[2:]...)
		cmdAdd()
	case "handoff":
		fmt.Fprintln(os.Stderr, "Note: 'brain handoff' is deprecated. Use 'brain add --eval'.")
		os.Exit(1)
	case "outcome":
		fmt.Fprintln(os.Stderr, "Note: 'brain outcome' is deprecated. Use 'brain add --eval --good/--bad'.")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage(false)
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

func printUsage(full bool) {
	v := version
	if v == "" {
		v = "dev"
	}
	if full {
		printUsageFull(v)
	} else {
		printUsageBrief(v)
	}
}

func printUsageBrief(v string) {
	fmt.Printf("brain %s — Persistent Memory for AI Coding Agents\n", v)
	fmt.Printf("https://github.com/dominhduc/agent-brain\n\n")
	fmt.Print(`USAGE
  brain <command> [arguments] [flags]

COMMANDS
  init                Initialize .brain/ knowledge hub
  get <topic|query>   Retrieve knowledge or search
  add <topic> "<msg>" Record a learning or decision
  add --eval          End session with self-evaluation
  clean               Run all cleanup (dedup, prune, decay, rebuild)
  doctor              Hub health check & diagnostics
  daemon <action>     Manage daemon (start|stop|status|review|failed|retry)
  config <action>     Manage settings (list|get|set|setup|reset)
  update              Update brain or skill files

TOPICS   gotcha, pattern, decision, architecture, memory
AREAS    ui, backend, infrastructure, database, security, testing, architecture, general

WORKFLOWS
  Session start     brain get all
  When corrected    brain add gotcha "..."
  Session end       brain add --eval

FLAGS  --dry-run  --json  --summary  --compact  --full  --yes/-y

Run 'brain help --full' for complete reference with all flags and examples.
`)
}

func printUsageFull(v string) {
	fmt.Printf("brain %s — Persistent Memory for AI Coding Agents\n", v)
	fmt.Printf("https://github.com/dominhduc/agent-brain\n\n")
	fmt.Print(`USAGE
  brain <command> [arguments] [flags]

QUICK START
  brain init                    Initialize knowledge hub in current project
  brain get all                 Load all accumulated knowledge
  brain add <topic> "<msg>"     Record a new learning or decision
  brain add --eval              End session with self-evaluation + handoff
  brain get <query>             Search if not a known topic

WORKFLOWS
  Session start     brain get all
  Before debugging  brain get gotchas
  When corrected    brain add gotcha "The fix"
  Session end       brain add --eval

COMMANDS
  CORE
    brain init                 Create .brain/ hub, AGENTS.md, git hooks, daemon
    brain get <topic>          Topics: all, gotchas, patterns, decisions, architecture, memory
                               Or search if not a known topic
                               Flags: --search (force search), --json, --summary, --compact,
                                      --message-only, --full, --focus "<topic>"
    brain add <topic> "<msg>"  Add entry to a topic
    brain add <area> <topic> "<msg>"  Add entry with area tag
    brain add --wm "<msg>"     Add to working memory (temporary)
    brain add --eval           Session evaluation + handoff
                               Flags: --good, --bad

  MAINTENANCE
    brain clean                Run all cleanup (prune + dedup + decay + rebuild)
                               Flags: --dry-run, --patterns, --duplicates, --decay, --rebuild, --fuzzy
    brain doctor               Hub statistics, health check & diagnostics
                               Flags: --json, --fix

  DAEMON
    brain daemon <action>      Actions: start, stop, restart, status, failed, retry, run, review
    brain daemon review        Interactive TUI to approve/reject pending entries
                               Flags: --all, --yes/-y, --tty

  CONFIG
    brain config list          List all settings
    brain config get <key>     Get a value
    brain config set <key> <value>  Set a value
    brain config reset <key>   Reset to default
    brain config setup         Interactive setup wizard
                               Config can be global (~/.config/brain/)
                               or project-specific (.brain/config.yaml)

  SKILLS & UPDATE
    brain update               Update agent-brain to latest version
    brain update --skills      Update skill files (preserves adaptations)
    brain update --skills --list   Show installed skill locations
    brain update --skills --diff   Compare installed vs latest templates
    brain update --skills --install  Install skill files to project directories
    brain update --skills --install --global  Install to global directories
    brain update --skills --reflect [--dry-run]  Generate skill adaptations from usage data

AREA TAXONOMY (8 topics)
  ui            Frontend, styling, components, accessibility
  backend       API, services, middleware, business logic
  infrastructure Deploy, CI/CD, Docker, cloud, monitoring
  database      Schemas, migrations, queries, indexes
  security      Auth, secrets, permissions, encryption
  testing       Unit/integration/e2e tests, mocks, fixtures
  architecture  Module structure, design patterns, data flow
  general       Cross-cutting conventions, tooling, guidelines

DEPRECATED ALIASES
  search  → brain get <query>       eval     → brain add --eval
  prune   → brain clean --patterns  dedup    → brain clean --duplicates
  sleep   → brain clean --decay     status   → brain doctor
  review  → brain daemon review     index    → brain clean --rebuild
  wm      → brain add --wm          handoff  → brain add --eval
  outcome → brain add --eval --good/--bad

EXAMPLES
  brain init
  brain get gotchas
  brain get all --focus "security"
  brain get "auth error"           # Auto-searches
  brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  brain add pattern "All handlers use middleware chain: logging -> auth -> rate-limit"
  brain add --eval --good
  brain clean --dry-run
  brain doctor
  brain daemon review
`)
}
