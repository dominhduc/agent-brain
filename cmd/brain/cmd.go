package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v3.0.3"

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
	case "supersede":
		cmdSupersedeEntry()
	case "edit":
		cmdUpdateEntry()
	case "consolidate":
		cmdConsolidate()
	case "embed":
		cmdEmbed()
	case "sync":
		cmdSync()
	case "grade":
		cmdGrade()
	case "trace":
		cmdTrace()
	case "search":
		cmdGet(jsonFlag, false, false, false, false)
	case "--help", "-h", "help":
		printUsage(hasFlag("--full"))


	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage(false)
		os.Exit(1)
	}
}

func isTermux() bool {
	return os.Getenv("TERMUX_VERSION") != ""
}

var flagAliases = map[string]string{
	"-f": "--full",
	"-g": "--global",
	"-l": "--list",
	"-d": "--dry-run",
	"-s": "--summary",
	"-c": "--compact",
	"-j": "--json",
	"-m": "--message-only",
	"-y": "--yes",
	"-h": "--help",
	"-v": "--version",
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[2:] {
		if arg == flag {
			return true
		}
		if alias, ok := flagAliases[arg]; ok && alias == flag {
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

CORE
  init                      Initialize .brain/ knowledge hub
  get <topic|query>         Retrieve knowledge or search
  add <topic> "<msg>"       Record a learning or decision
  add --eval                End session with self-evaluation

REASONING (v3)
  grade                     Grade entries for accuracy and usefulness
  trace <action>            Capture session reasoning (step|save|extract|list)

MAINTENANCE
  clean                     Run cleanup (dedup, prune, decay, rebuild)
  consolidate               Find and merge duplicate entries
  doctor                    Hub health check & diagnostics
  edit | supersede          Manage entry lifecycle

DAEMON & CONFIG
  daemon <action>           start|stop|status|review|failed|retry
  config <action>           list|get|set|setup|reset
  update                    Update brain or skill files
  embed | sync              Embeddings and cross-machine sharing

TOPICS   gotcha, pattern, decision, architecture, memory, wm
AREAS    ui, backend, infrastructure, database, security, testing, architecture, general

WORKFLOWS
  Session start     brain get all
  When corrected    brain add gotcha "..."
  Session end       brain add --eval

FLAGS   --dry-run/-d  --json/-j  --summary/-s  --compact/-c  --full/-f  --yes/-y

Run 'brain help --full' for complete reference with flags and examples.
`)
}

func printUsageFull(v string) {
	fmt.Printf("brain %s — Persistent Memory for AI Coding Agents\n", v)
	fmt.Printf("https://github.com/dominhduc/agent-brain\n\n")
	fmt.Print(`USAGE
  brain <command> [arguments] [flags]

QUICK START
  brain init                    Initialize knowledge hub
  brain get all                 Load accumulated knowledge
  brain add <topic> "<msg>"     Record a learning or decision
  brain add --eval              End session with self-evaluation

WORKFLOWS
  Session start     brain get all
  Before debugging  brain get gotchas
  When corrected    brain add gotcha "The fix"
  Session end       brain add --eval  (auto-adapts skills)

COMMANDS

  CORE
    init                       Create .brain/ hub, AGENTS.md, git hooks, daemon
    get <topic|query>          Retrieve knowledge or search
                                 Topics: all, gotchas, patterns, decisions,
                                         architecture, memory, wm
                                 Flags: --search  --json/-j  --summary/-s
                                        --compact/-c  --message-only/-m  --full/-f
                                        --focus "<topic>"  --context  --budget N
    add <topic> "<msg>"        Add entry (fuzzy dedup at add time)
                                 Flags: --global/-g
    add <area> <topic> "<msg>" Add entry with area tag
    add --wm "<msg>"           Add to working memory (temporary)
    add --eval                 Session evaluation + handoff
                                 Flags: --good  --bad

  REASONING (v3)
    grade                      Grade entries for accuracy, specificity, generality
                                 Flags: --dry-run/-d
    trace step                 Append a reasoning step
                                 Flags: --action  --target  --outcome  --reasoning
    trace save                 Finalize current trace
                                 Flags: --outcome  --goal
    trace extract              Extract knowledge from traces via LLM
                                 Flags: --dry-run/-d
    trace list                 Show all traces

  MAINTENANCE
    clean                      Run cleanup (prune + dedup + decay + rebuild)
                                 Flags: --dry-run/-d  --patterns  --duplicates
                                        --decay  --rebuild  --fuzzy
    doctor                     Hub health check & diagnostics
                                 Flags: --json/-j  --fix  --conflicts
    consolidate                Find and merge related entries
                                 Flags: --dry-run  --apply  --topic  --llm
    edit <topic> <ts>          Update an entry in-place
                                 Flags: --message "<new text>"
    supersede <topic> <old> <new>  Mark entry as superseded

  EMBEDDINGS & SHARING
    embed                      Embed entries for semantic search
                                 Flags: --all  --status
    sync                       Export to docs/brain/ for sharing
                                 Flags: --import  --dry-run/-d

  DAEMON
    daemon start|stop|restart  Manage background daemon
    daemon status              Health check + queue depth
    daemon review              Approve/reject pending entries
                                 Flags: --all  --yes/-y  --tty
    daemon failed              List failed queue items
    daemon retry               Requeue failed items
    daemon run                 Run in foreground (debugging)
    daemon run --once           Process queued commits, then exit

  CONFIG
    config list                List all settings
    config get <key>           Get a value (API keys masked)
    config set <key> <value>   Set a value (auto-detects project scope)
    config set <key> <value> --global  Force write to global config
    config reset <key>         Reset to default
    config setup               Interactive setup wizard

  SKILLS & UPDATE
    update                     Update agent-brain to latest version
    update --skills            Update skill files (preserves adaptations)
    update --skills --list     Show installed skill locations
    update --skills --diff     Compare installed vs latest templates
    update --skills --install  Install skill files to project directories
    update --skills --install --global  Install globally
    update --skills --reflect  Generate adaptations from usage data
                                 Flags: --dry-run/-d

AREA TAXONOMY
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
  brain get "auth error"                   # Auto-searches
  brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  brain add pattern "All handlers use middleware chain"
  brain add --eval --good
  brain grade --dry-run
  brain trace step --action "debug" --target "auth.go" --reasoning "Found nil pointer"
  brain trace save --outcome success --goal "Fix auth bug"
  brain clean --dry-run
  brain doctor
  brain daemon review
`)
}
