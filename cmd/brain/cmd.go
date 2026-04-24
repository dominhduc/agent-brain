package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dominhduc/agent-brain/internal/otel"
)

var version = "v3.0.1"

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

COMMANDS
  init                Initialize .brain/ knowledge hub
  get <topic|query>   Retrieve knowledge or search
  add <topic> "<msg>" Record a learning or decision
  add --eval          End session with self-evaluation
  clean               Run all cleanup (dedup, prune, decay, rebuild)
  grade               Grade entries for accuracy and usefulness
  trace <action>      Capture session reasoning (save|step|extract|list)
  consolidate         Find and merge duplicate entries
  doctor              Hub health check & diagnostics
  daemon <action>     Manage daemon (start|stop|status|review|failed|retry)
  config <action>     Manage settings (list|get|set|setup|reset)
  update              Update brain or skill files

TOPICS   gotcha, pattern, decision, architecture, memory, wm
AREAS    ui, backend, infrastructure, database, security, testing, architecture, general

WORKFLOWS
  Session start     brain get all
  When corrected    brain add gotcha "..."
  Session end       brain add --eval  (auto-adapts skills)

FLAGS  --dry-run/-d  --json/-j  --summary/-s  --compact/-c  --full/-f  --yes/-y

Run 'brain help --full' or 'brain help -f' for complete reference with all flags and examples.
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
  brain add --eval              End session with self-evaluation + handoff (auto-adapts skills)
  brain get <query>             Search if not a known topic

WORKFLOWS
  Session start     brain get all
  Before debugging  brain get gotchas
  When corrected    brain add gotcha "The fix"
  Session end       brain add --eval  (auto-adapts skills)

COMMANDS
  CORE
    brain init                 Create .brain/ hub, AGENTS.md, git hooks, daemon
    brain get <topic>          Topics: all, gotchas, patterns, decisions, architecture, memory, wm
                                  Or search if not a known topic
                                 Flags: --search (force search), --json/-j, --summary/-s, --compact/-c,
                                        --message-only/-m, --full/-f, --focus "<topic>",
                                        --context (boost entries matching git diff),
                                        --budget N (custom token limit)
    brain add <topic> "<msg>"  Add entry to a topic (fuzzy dedup at add time)
                                Flags: --global/-g (also add to global store)
    brain add <area> <topic> "<msg>"  Add entry with area tag
    brain add --wm "<msg>"     Add to working memory (temporary)
    brain add --eval           Session evaluation + handoff (auto-adapts skills)
                               Flags: --good, --bad

  REASONING (v3)
    brain grade                Grade entries for accuracy, specificity, generality
                                Flags: --dry-run/-d (preview without applying)
    brain trace <action>       Capture session reasoning traces
                                 save    Finalize current trace
                                           Flags: --outcome <success|partial|failure>, --goal "<desc>"
                                 step    Append a reasoning step to current trace
                                           Flags: --action, --target, --outcome, --reasoning
                                 extract Extract knowledge from traces via LLM
                                           Flags: --dry-run/-d
                                 list    Show all traces

  MAINTENANCE
    brain clean                Run all cleanup (prune + dedup + decay + rebuild)
                                Flags: --dry-run/-d, --patterns, --duplicates, --decay, --rebuild, --fuzzy
    brain doctor               Hub statistics, health check & diagnostics
                                Flags: --json/-j, --fix, --conflicts
    brain consolidate          Find and merge duplicate entries
                                Flags: --dry-run, --apply, --topic, --llm (semantic merge via LLM)
    brain edit <topic> <ts>    Update an entry in-place
                                Flags: --message "<new text>"
    brain supersede <topic> <old-ts> <new-ts>  Mark entry as superseded

  EMBEDDINGS
    brain embed                Embed entries for semantic search
                                Flags: --all, --status
    brain sync                 Export topic files to docs/brain/ for sharing
                                Flags: --import (import from docs/brain/), --dry-run/-d

  DAEMON
    brain daemon <action>      Actions: start, stop, restart, status, failed, retry, run, review
    brain daemon review        Approve/reject pending entries (respects profile)
                                 Flags: --all, --yes/-y (auto-accept), --tty (force interactive)
                                 Profile=agent auto-accepts; guard/assist prompts interactively
                                 Uses systemd on Linux, launchd on macOS,
                                 nohup on systems without systemd (Termux, proot)

  CONFIG
    brain config list          List all settings
    brain config get <key>     Get a value
    brain config set <key> <value>  Set a value (auto-detects project scope)
    brain config set <key> <value> --global  Force write to global config
    brain config reset <key>   Reset to default
    brain config setup         Interactive setup wizard
                                 Project config: .brain/config.yaml (auto-created)
                                 Global config: ~/.config/brain/config.yaml

  SKILLS & UPDATE
    brain update               Update agent-brain to latest version
    brain update --skills      Update skill files (preserves adaptations)
    brain update --skills -l/--list   Show installed skill locations
    brain update --skills --diff   Compare installed vs latest templates
    brain update --skills --install  Install skill files to project directories
    brain update --skills --install -g/--global  Install to global directories
    brain update --skills --reflect [--dry-run/-d]  Generate skill adaptations from usage data

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
  brain get "auth error"           # Auto-searches
  brain add infrastructure gotcha "VPS uses Ubuntu 22.04"
  brain add pattern "All handlers use middleware chain: logging -> auth -> rate-limit"
  brain add --eval --good
  brain grade --dry-run
  brain trace step --action "debug" --target "auth.go" --reasoning "Found nil pointer"
  brain trace save --outcome success --goal "Fix auth bug"
  brain clean --dry-run
  brain doctor
  brain daemon review
`)
}
