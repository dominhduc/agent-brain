# 🧠 agent-brain

> **Give your AI coding agent a memory that grows smarter with every session.**

`brain` is a CLI tool that creates a per-project knowledge hub (`.brain/`) for AI coding agents. It tracks what your agent learns, auto-analyzes commits via LLM, and feeds accumulated knowledge back into every new session.

**Works with** OpenCode, Claude Code, Cursor, Windsurf — any agent that can run shell commands.

---

## Table of Contents

- [Why brain?](#why-brain)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Commands](#commands)
  - [Core Commands](#core-commands)
  - [Knowledge Commands](#knowledge-commands)
  - [Daemon Commands](#daemon-commands)
  - [Maintenance Commands](#maintenance-commands)
  - [Configuration](#configuration-commands)
- [The 8-Topic Taxonomy](#the-8-topic-taxonomy)
- [Autonomy Profiles](#autonomy-profiles)
- [File Structure](#file-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [The Daemon](#the-daemon)
- [Troubleshooting](#troubleshooting)
- [For AI Agents](#for-ai-agents)
- [Building from Source](#building-from-source)
- [License](#license)

---

## Why brain?

AI coding agents are brilliant — but they **forget everything** between sessions. Every time you start a new conversation, your agent starts from zero.

`brain` fixes this by giving your agent a **persistent memory**:

| Without brain | With brain |
|---------------|------------|
| Agent forgets project conventions every session | Agent loads accumulated knowledge automatically |
| You repeat the same corrections over and over | Corrections are recorded and remembered |
| No institutional memory across sessions | Knowledge compounds and grows smarter |
| Agent makes the same mistakes repeatedly | Past gotchas are flagged before they happen again |

**v0.20.0** adds OpenTelemetry audit logging for full observability.

---

## Quick Start

Three commands. That's all you need.

```bash
# 1. Install brain
curl -sL https://github.com/dominhduc/agent-brain/releases/latest/download/brain_$(uname -s)_$(uname -m).tar.gz | tar xz -C ~/.local/bin

# 2. Set your API key (one-time)
brain config set api-key sk-or-v1-...

# 3. Initialize in your project
cd your-project
brain init
```

That's it. Every commit is analyzed automatically. Every agent session loads accumulated knowledge. No manual configuration needed.

---

## How It Works

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  You code   │────▶│  Git commit  │────▶│  Git push       │
│  + agent    │     │  + push      │     │                 │
└─────────────┘     └──────────────┘     └────────┬────────┘
                                                  │
                                         pre-push hook
                                                  │
                                         ┌────────▼────────┐
                                         │  Queue item     │
                                         │  (.brain/.queue)│
                                         └────────┬────────┘
                                                  │
                                      brain-daemon (background)
                                                  │
                                      OpenRouter LLM analysis
                                                  │
                                      ┌──────────▼──────────┐
                                      │  .brain/pending/    │
                                      │  (review queue)     │
                                      └──────────┬──────────┘
                                                 │
                                     brain review (human approves)
                                                 │
                                      ┌──────────▼──────────┐
                                      │  .brain/ knowledge  │
                                      │  files (permanent)  │
                                      └──────────┬──────────┘
                                                 │
                                     Next session: agent loads knowledge
                                                 │
                                      Agent makes fewer mistakes
```

### The Three Layers

| Layer | What it does | Automatic? |
|-------|-------------|------------|
| **Git Hook** | Captures pushed commits and queues them | Yes — fires on every push |
| **Daemon** | Analyzes queued commits via LLM, writes to pending | Yes — runs in background |
| **Review Gate** | Human approves/rejects findings before they become permanent | Via `brain review` |
| **Agent Instructions** | Tells the agent to load knowledge, add learnings, self-evaluate | Yes — loaded every session via AGENTS.md |

### What Gets Tracked

- **Gotchas** — Errors the agent hit and how to fix them
- **Patterns** — Conventions discovered from actual code (not guesses)
- **Decisions** — Why certain choices were made, with rationale
- **Architecture** — Module relationships and key abstractions
- **Sessions** — Per-session journals with self-evaluations

---

## Commands

### Core Commands

#### `brain init`

Initialize a knowledge hub in your current project.

```bash
cd your-project
brain init
```

Creates `.brain/` directory, `AGENTS.md` (agent instructions), compatibility wrappers (`CLAUDE.md`, `.cursorrules`, `.windsurfrules`), installs git hooks, and starts the background daemon.

**Run once per project.** After that, everything is automatic.

---

### Knowledge Commands

#### `brain get <topic>`

Read accumulated knowledge. Entries show a strength indicator (`●0.82`) based on recency and usage.

```bash
brain get gotchas                          # See known pitfalls
brain get patterns                         # See discovered conventions
brain get all                              # Load everything
brain get all --focus "infrastructure"     # Filter by topic
brain get gotchas --json                   # Machine-readable output
```

**Topics:** `memory`, `gotchas`, `patterns`, `decisions`, `architecture`, `all`

Retrieval automatically strengthens memories (extends their half-life).

#### `brain search <query>`

Search across all knowledge files.

```bash
brain search "auth"
brain search "argon2" --topic "security"   # Filter by topic
brain search "database" --json
```

#### `brain add <topic> "<message>"`

Add knowledge manually (or have your agent do it).

```bash
brain add gotcha "Project uses argon2, NOT bcrypt"
brain add pattern "Tests use Vitest, not Jest"
brain add decision "Chose SQLite over PostgreSQL for simplicity"
brain add infrastructure gotcha "VPS uses Ubuntu 22.04"   # Tag with topic
brain add --wm "investigating auth bug"                    # Working memory
```

Use an 8-topic prefix (`ui`, `backend`, `infrastructure`, `database`, `security`, `testing`, `architecture`, `general`) before the entry type to tag entries.

#### `brain eval`

Create a session evaluation file and handoff.

```bash
brain eval
brain eval --good    # Apply positive outcome + flush working memory
brain eval --bad     # Apply negative outcome + flush working memory
```

Creates a session journal with git summary, recent commits, and auto-created handoff with topic detection. Use `--good` or `--bad` to provide feedback on retrieved memories.

---

### Daemon Commands

#### `brain daemon start|stop|status`

Manage the background analysis daemon.

```bash
brain daemon start    # Start background daemon
brain daemon stop     # Stop daemon
brain daemon status   # Check health, queue depth
```

The daemon watches for new commits, sends diffs to OpenRouter for analysis, and writes findings to a **pending queue** for human review via `brain review`.

#### `brain review [--all]`

Review and approve knowledge entries found by the daemon.

```bash
brain review           # Interactive TUI to approve/reject entries
brain review --all     # Import existing topic entries for re-review
```

The TUI shows entries grouped by topic. Navigate with arrows, accept with `a`, reject with `r`, press `q` to quit. Only entries you approve become permanent.

---

### Maintenance Commands

#### `brain prune [--dry-run]`

Archive stale knowledge entries.

```bash
brain prune --dry-run   # See what would be pruned
brain prune             # Actually prune
```

Reads patterns from `.brainprune` (like `.gitignore` for knowledge). Moves matching entries to `.brain/archived/`.

#### `brain sleep [--dry-run]`

Run memory consolidation — the biological "sleep" cycle.

```bash
brain sleep           # Archive decayed entries, merge related patterns
brain sleep --dry-run # Preview what would change
```

Entries with strength below threshold (from disuse) are archived. Related patterns are merged into consolidated insights.

#### `brain index rebuild`

Rebuild the metadata index from topic files.

```bash
brain index rebuild
```

Useful if the index gets out of sync. Safe to run anytime.

---

### Configuration Commands

#### `brain config [get|set|list|reset|setup]`

View or modify configuration.

```bash
brain config                              # Show current config
brain config get api-key                  # Get a single value
brain config set api-key <your-key>       # Set API key
brain config set model anthropic/claude-3.5-haiku  # Change model
brain config set profile guard            # Set review strictness
brain config list                         # List all available keys
brain config reset model                  # Reset one key to default
brain config setup                        # Interactive setup wizard
```

**Available keys:** `api-key`, `model`, `provider`, `profile`, `poll-interval`, `max-retries`, `retry-backoff`, `max-diff-lines`

#### `brain status [--json]`

Show knowledge hub statistics.

```
Knowledge Hub Status
====================
MEMORY.md:       45 lines (OK, under 200)
Topic files:     4 files
Session files:   12 sessions
Total size:      23 KB
Queue depth:     0 pending, 47 done
Pending entries: 2 (run 'brain review' to approve)
```

---

## The 8-Topic Taxonomy

Entries are tagged with one or more topics for smarter filtering:

| Topic | Keywords |
|-------|----------|
| `ui` | react, css, component, style, tailwind, html, frontend |
| `backend` | api, handler, controller, service, middleware, route |
| `infrastructure` | vps, deploy, docker, ci, cd, kubernetes, nginx |
| `database` | sql, migration, schema, query, postgres, mysql |
| `security` | auth, secret, token, jwt, oauth, encrypt |
| `testing` | test, spec, mock, assert, vitest, jest |
| `architecture` | module, layer, package, pattern, abstraction |
| `general` | catch-all for cross-cutting knowledge |

---

## Autonomy Profiles

Control how much human review is needed:

| Profile | Auto-Dedup | Auto-Accept | Best For |
|---------|-----------|-------------|----------|
| `guard` (default) | No | No | New projects, all entries reviewed |
| `assist` | Yes | No | Stable projects, less noise |
| `agent` | Yes | Yes | Trusted patterns, fully automatic |

```bash
brain config set profile assist   # Less strict
brain config set profile agent    # Fully automatic
```

---

## File Structure

```
your-project/
├── AGENTS.md                    # Instructions for coding agents
├── CLAUDE.md                    # "See AGENTS.md"
├── .cursorrules                 # "See AGENTS.md"
├── .windsurfrules               # "See AGENTS.md"
│
├── .brain/                      # Knowledge hub (git-tracked)
│   ├── MEMORY.md                # Main index (< 200 lines)
│   ├── gotchas.md               # Error patterns + fixes
│   ├── patterns.md              # Discovered conventions
│   ├── decisions.md             # Architecture decisions + rationale
│   ├── architecture.md          # Module relationships
│   ├── sessions/                # Per-session journals
│   │   └── 2026-04-02T14-30-00.md
│   ├── .queue/                  # Daemon queue (local only)
│   │   ├── commit-*.json
│   │   └── done/
│   ├── pending/                 # Entries awaiting review (local only)
│   ├── index.json               # Metadata index (local only)
│   ├── buffer/                  # Working memory (local only)
│   ├── handoffs/                # Session handoffs (local only)
│   └── archived/                # Pruned entries (local only)
│
└── .brainprune                  # Patterns for knowledge pruning (optional)
```

---

## Installation

### Quick install (recommended)

```bash
# If brain is already installed
brain update

# Or download the latest release binary
curl -sL https://github.com/dominhduc/agent-brain/releases/latest/download/brain_$(uname -s)_$(uname -m).tar.gz | tar xz -C ~/.local/bin
```

### Build from source

```bash
git clone https://github.com/dominhduc/agent-brain.git
cd agent-brain
make build && make install
```

### Verify installation

```bash
brain doctor
```

---

## Configuration

### OpenRouter API Key

The daemon needs an OpenRouter API key to analyze commits. Set it one of three ways:

```bash
# Option 1: Via brain config (recommended)
brain config set api-key sk-or-v1-...

# Option 2: Environment variable
export BRAIN_API_KEY="sk-or-v1-..."

# Option 3: Interactive prompt (during brain init)
brain init  # Will ask for your API key
```

Get your key at [openrouter.ai](https://openrouter.ai).

### Model Selection

The default model is `anthropic/claude-3.5-haiku` — fast and cheap (~$0.01 per commit). You can change it:

```bash
brain config set model anthropic/claude-3.5-haiku
brain config set model openai/gpt-4o-mini
brain config set model google/gemini-2.5-flash
```

Any OpenRouter model works. Choose based on your budget and accuracy needs.

### Config File Location

All config lives in `~/.config/brain/config.yaml`:

```yaml
llm:
  provider: openrouter
  api_key: sk-or-v1-...
  model: anthropic/claude-3.5-haiku
analysis:
  max_diff_lines: 2000
  categories: [gotchas, patterns, decisions, architecture]
daemon:
  poll_interval: 5s
  max_retries: 3
  retry_backoff: exponential
```

### OpenTelemetry Tracing

Enable distributed tracing for debugging and observability:

```bash
# Enable OTel tracing
brain config set otel.enabled true

# Set the export endpoint
brain config set otel.endpoint localhost:4317           # OTLP gRPC (default)
brain config set otel.endpoint http://localhost:4318/v1/traces  # OTLP HTTP
brain config set otel.endpoint stdout                   # Debug: print to console
```

When enabled, `brain` emits OpenTelemetry spans for CLI commands, daemon pipeline, and review sessions. Compatible with Jaeger, Grafana Tempo, Honeycomb, etc.

---

## The Daemon

### What It Does

The `brain-daemon` runs in the background and:

1. Watches `.brain/.queue/` for new commit items (added by the git hook)
2. Reads the full git diff for each commit
3. Sends the diff to OpenRouter LLM for analysis
4. Writes findings (gotchas, patterns, decisions, architecture) to `.brain/` topic files
5. Moves processed items to `.brain/.queue/done/`

### Why a Daemon?

Without the daemon, knowledge only accumulates when the agent remembers to add it. With the daemon, **every commit is analyzed automatically** — even commits you make manually, even commits from Cursor or Windsurf that don't read AGENTS.md.

The queue is your safety net: if the LLM API is down, items wait. When the API comes back, processing resumes. Nothing is lost.

### Cost Estimate

| Usage | Commits/day | Cost/day | Cost/month |
|-------|-------------|----------|------------|
| Light (side project) | 3 | ~$0.03 | ~$0.90 |
| Normal (active dev) | 10 | ~$0.10 | ~$3.00 |
| Heavy (full-time) | 30 | ~$0.30 | ~$9.00 |

Based on Claude Haiku pricing via OpenRouter. Cheaper models (Gemini Flash, GPT-4o-mini) cost less.

### Service Registration

`brain init` automatically registers the daemon as a system service:

- **macOS:** `launchd` via `~/Library/LaunchAgents/com.dominhduc.brain-daemon.plist`
- **Linux:** `systemd` via `~/.config/systemd/user/brain-daemon.service`

The daemon starts on login and restarts on crash.

---

## Troubleshooting

### "Knowledge hub not found"

Run `brain init` in your project directory first.

### "OpenRouter API key not configured"

```bash
brain config set api-key sk-or-v1-...
```

Get your key at [openrouter.ai](https://openrouter.ai).

### "Daemon is not running"

```bash
brain daemon start
```

Or run in foreground to debug:

```bash
brain daemon run
```

### "Queue has many pending items"

The daemon might be stuck. Check:

```bash
brain daemon status
```

If the API key is invalid or the model is unavailable, items will pile up. Fix the config and the daemon will catch up.

### "MEMORY.md is too large"

```bash
brain prune --dry-run   # See what can be archived
brain prune             # Archive stale entries
```

### "brain: command not found"

`~/.local/bin` is not in your PATH. Fix it:

```bash
export PATH="$HOME/.local/bin:$PATH"
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

### "dubious ownership" / git ownership error

```bash
git config --global --add safe.directory /path/to/your/project
```

### "Secret detected in diff" (daemon skipping commits)

The daemon found a potential secret and skipped the commit. The item is moved to `.brain/.queue/flagged/`.

Review the commit for exposed secrets. If it's a false positive, requeue:

```bash
mv .brain/.queue/flagged/commit-*.processing .brain/.queue/commit-TIMESTAMP.json
```

### ".brain is a symlink" error

For security, `.brain/` cannot be a symlink. Remove it and reinitialize:

```bash
rm .brain
brain init
```

### Daemon logs

- **macOS:** `/tmp/brain-daemon.log` (stdout), `/tmp/brain-daemon.err` (stderr)
- **Linux:** `journalctl --user -u brain-daemon`

---

## For AI Agents

When `brain init` creates `AGENTS.md`, it includes instructions for your coding agent:

1. **At session start:** Run `brain get all` to load accumulated knowledge
2. **During work:** Run `brain search "<topic>"` before unfamiliar code, `brain get gotchas` before debugging
3. **When corrected:** Run `brain add <topic> "<...>"` or use topic prefix: `brain add infrastructure gotcha "..."`
4. **At session end:** Run `brain eval` (or `brain eval --good` / `brain eval --bad` for feedback)

The agent doesn't need to remember anything — the instructions are in AGENTS.md, which loads every session.

### Compatibility

| Tool | Reads AGENTS.md? | Uses .brain/? |
|------|-----------------|---------------|
| OpenCode | Yes (native) | Yes |
| Claude Code | Via CLAUDE.md wrapper | Yes |
| Cursor | Via .cursorrules wrapper | Yes |
| Windsurf | Via .windsurfrules wrapper | Yes |

All tools benefit from the daemon's auto-analysis, even if they don't read AGENTS.md.

---

## Building from Source

```bash
git clone https://github.com/dominhduc/agent-brain.git
cd agent-brain
make build
./bin/brain --help
```

**Requirements:** Go 1.24+

---

## TODO Before First Use

- [ ] **Set your OpenRouter API key:** `brain config set api-key sk-or-v1-...`
- [ ] **Choose your preferred model** in `~/.config/brain/config.yaml` (default: `anthropic/claude-3.5-haiku`)
- [ ] **Review daemon configuration:** `brain daemon status`
- [ ] **Run `brain init`** in your first project
- [ ] **Optional:** After working, run `brain eval --good` to strengthen helpful memories

---

## License

MIT
