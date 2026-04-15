# 🧠 agent-brain

> **Give your AI coding agent a memory that grows smarter with every session.**

`agent-brain` (CLI command: `brain`) is a CLI tool that creates a per-project knowledge hub (`.brain/`) for AI coding agents. It tracks what your agent learns, auto-analyzes commits via LLM, and feeds accumulated knowledge back into every new session.

**Works with** OpenCode, Claude Code, Cursor, Windsurf — any agent that can run shell commands.

---

## Table of Contents

- [For AI Agents (Start Here)](#for-ai-agents-start-here)
- [Why agent-brain?](#why-agent-brain)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Commands](#commands)
  - [Core Commands](#core-commands)
  - [Knowledge Commands](#knowledge-commands)
  - [Daemon Commands](#daemon-commands)
  - [Skill Commands](#skill-commands)
  - [Maintenance Commands](#maintenance-commands)
  - [Configuration Commands](#configuration-commands)
- [The 8-Topic Taxonomy](#the-8-topic-taxonomy)
- [Autonomy Profiles](#autonomy-profiles)
- [File Structure](#file-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [The Daemon](#the-daemon)
- [Troubleshooting](#troubleshooting)
- [Building from Source](#building-from-source)
- [License](#license)
- [References & Inspiration](#references--inspiration)

---

## For AI Agents (Start Here)

> **Tell your AI coding agent to install and set up agent-brain for you.** It's the fastest way to get started.

When `brain init` runs, it creates `AGENTS.md` and installs **Agent Skills** — skill files that teach coding agents how to use `brain` automatically.

1. **At session start:** Run `brain get all` to load accumulated knowledge
2. **During work:** Run `brain get "<topic>"` before unfamiliar code, `brain get gotchas` before debugging
3. **When corrected:** Run `brain add <topic> "<...>"` or use topic prefix: `brain add infrastructure gotcha "..."`
4. **At session end:** Run `brain add --eval` (or `brain add --eval --good` / `brain add --eval --bad` for feedback)

The agent doesn't need to remember anything — the instructions are in AGENTS.md and the Agent Skill, which load every session.

### Compatibility

| Tool | Reads AGENTS.md? | Uses Agent Skill? | Uses .brain/? |
|------|-----------------|-------------------|---------------|
| OpenCode | Yes (native) | Yes (`.opencode/skills/`) | Yes |
| Claude Code | Via CLAUDE.md wrapper | Yes (`.claude/skills/`) | Yes |
| Cursor | Via .cursorrules wrapper | Yes (`.agents/skills/`) | Yes |
| Windsurf | Via .windsurfrules wrapper | Yes (`.agents/skills/`) | Yes |

All tools benefit from the daemon's auto-analysis, even if they don't read AGENTS.md or the Agent Skill.

---

## Why agent-brain?

AI coding agents are brilliant — but they **forget everything** between sessions. Every time you start a new conversation, your agent starts from zero.

`agent-brain` fixes this by giving your agent a **persistent memory**:

| Without agent-brain | With agent-brain |
|---------------|------------|
| Agent forgets project conventions every session | Agent loads accumulated knowledge automatically |
| You repeat the same corrections over and over | Corrections are recorded and remembered |
| No institutional memory across sessions | Knowledge compounds and grows smarter |
| Agent makes the same mistakes repeatedly | Past gotchas are flagged before they happen again |

**v1.4.2** adds short flag aliases (`-f`, `-d`, `-s`, `-c`, `-j`, `-m`, `-g`, `-l`, `-y`, `-h`, `-v`) and removes deprecated command aliases. **v1.4.1** reorganizes `brain help` into brief (28 lines) + `--full` reference, fixes daemon review writing paraphrased duplicates into topic files, and documents all deprecated command aliases. v1.4.0 consolidates 21 commands into 8 with backward-compatible aliases. v1.3.0 adds fuzzy deduplication (`brain clean --duplicates --fuzzy`) using trigram Jaccard similarity to catch near-duplicate entries.

---

## Quick Start

Three commands. That's all you need.

```bash
# 1. Install agent-brain
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
                                       brain daemon review (human approves)
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
| **Review Gate** | Human approves/rejects findings before they become permanent | Via `brain daemon review` |
| **Agent Instructions** | Tells the agent to load knowledge, add learnings, self-evaluate | Yes — loaded every session via AGENTS.md |
| **Self-Learning** | `brain update --skills --reflect` adapts agent instructions based on usage patterns | On-demand — run periodically |

### What Gets Tracked

- **Gotchas** — Errors the agent hit and how to fix them
- **Patterns** — Conventions discovered from actual code (not guesses)
- **Decisions** — Why certain choices were made, with rationale
- **Architecture** — Module relationships and key abstractions
- **Sessions** — Per-session journals with self-evaluations

---

## Commands

### Getting Help

```bash
brain help              # Brief: commands, topics, workflows (~28 lines)
brain help --full       # Complete reference: all flags, area taxonomy, examples
```

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

Read accumulated knowledge. Entries show a strength indicator (`1.00`) based on recency and usage.

```bash
brain get gotchas                          # See known pitfalls
brain get patterns                         # See discovered conventions
brain get all                              # Tiered view: overview + recent + top
brain get all --full                       # Complete dump (all entries)
brain get all --focus "infrastructure"     # Filter by topic
brain get gotchas --compact                # One line per entry, no blank lines
brain get gotchas --message-only           # Pure message text (no metadata)
brain get gotchas --json                   # Structured JSON output
```

**Output modes:**
- **Default:** Full entries with timestamps and strength scores
- **`--compact`:** One line per entry with relative timestamps (e.g., `Apr 26`)
- **`--message-only`:** Just the message text — ideal for piping to AI agents
- **`--json`:** Structured format `{topic, entry_count, entries: [{timestamp, message}]}`

**Topics:** `memory`, `gotchas`, `patterns`, `decisions`, `architecture`, `all`

Retrieval automatically strengthens memories (extends their half-life).

#### `brain get <query>` (search mode)

When the argument is not a known topic name, `brain get` auto-searches across all knowledge files. Results are grouped by topic with markdown stripped.

```bash
brain get "auth"
brain get "argon2" --topic "security"   # Filter by topic
brain get "database" --json
brain get "auth" --search               # Force search mode explicitly
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

#### `brain add --eval`

Create a session evaluation file and handoff.

```bash
brain add --eval
brain add --eval --good    # Apply positive outcome + flush working memory
brain add --eval --bad     # Apply negative outcome + flush working memory
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
brain daemon retry    # Requeue all failed items for retry
```

The daemon watches for new commits, sends diffs to OpenRouter for analysis, and writes findings to a **pending queue** for human review via `brain daemon review`.

If the daemon fails to process a commit (e.g., LLM timeout, API error), the item moves to the failed queue. Use `brain daemon retry` to requeue all failed items for another attempt.

#### `brain daemon review [--all]`

Review and approve knowledge entries found by the daemon.

```bash
brain daemon review           # Interactive TUI to approve/reject entries
brain daemon review --all     # Import existing topic entries for re-review
```

The TUI shows entries grouped by topic. Navigate with arrows, accept with `a`, reject with `r`, press `q` to quit. Only entries you approve become permanent.

Accepted entries are deduplicated against existing topic file content using trigram similarity (threshold 0.55). If an accepted entry paraphrases what's already in a topic file, it's skipped automatically.

---

### Skill Commands

Agent Skills teach coding agents (Claude Code, OpenCode, Cursor, etc.) how to use `brain` automatically. Skills are installed during `brain init` and live in `.claude/skills/agent-brain/`, `.opencode/skills/agent-brain/`, and `.agents/skills/agent-brain/`.

#### `brain update --skills --list`

Show installed skill locations and their status.

```bash
brain update --skills --list
```

#### `brain update --skills --diff`

Compare installed skill files against the latest embedded templates.

```bash
brain update --skills --diff
```

#### `brain update --skills`

Update skill files to the latest version. Warns about uncommitted git changes. Preserves adaptation markers generated by `brain update --skills --reflect`.

```bash
brain update --skills
```

#### `brain update --skills --reflect [--dry-run]`

Generate skill adaptations from accumulated behavior data (command usage, search queries, eval outcomes). Adaptations are appended to installed skill files and preserved across updates.

```bash
brain update --skills --reflect           # Generate and apply adaptations
brain update --skills --reflect --dry-run # Preview what would be added
```

#### `brain update --skills --install [--global]`

Install skill files to project directories (or global with `--global`).

```bash
brain update --skills --install           # Project-local
brain update --skills --install --global  # Global (~/.claude/skills/, etc.)
```

---

### Maintenance Commands

#### `brain clean --patterns [--dry-run]`

Archive stale knowledge entries.

```bash
brain clean --patterns --dry-run   # See what would be pruned
brain clean --patterns             # Actually prune
```

Reads patterns from `.brainprune` (like `.gitignore` for knowledge). Moves matching entries to `.brain/archived/`.

#### `brain clean --duplicates [--dry-run]`

Find and remove duplicate entries across all topic files.

```bash
brain clean --duplicates --dry-run   # See what duplicates would be removed
brain clean --duplicates             # Actually remove duplicates
brain clean --duplicates --fuzzy     # Also catch near-duplicates (trigram Jaccard, threshold 0.55)
```

**Exact mode** (default): Uses SHA-256 fingerprinting of normalized content to detect duplicates, including cross-topic duplicates (same entry in multiple files). Keeps the first occurrence, archives removed entries to `.brain/archived/dedup-YYYY-MM-DD.md`, and shows a detailed summary report.

**Fuzzy mode** (`--fuzzy`): Uses character trigram Jaccard similarity (threshold 0.55) to catch near-duplicates — entries that paraphrase the same concept with different wording. In active projects, this typically finds 50%+ more duplicates than exact mode alone.

Run when `brain get --summary` shows `⚠️ duplicates detected`.

#### `brain clean --decay [--dry-run]`

Run memory consolidation — the biological "sleep" cycle.

```bash
brain clean --decay           # Archive decayed entries, merge related patterns
brain clean --decay --dry-run # Preview what would change
```

Entries with strength below threshold (from disuse) are archived. Related patterns are merged into consolidated insights.

#### `brain clean --rebuild`

Rebuild the metadata index from topic files.

```bash
brain clean --rebuild
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

#### `brain doctor [--json]`

Show knowledge hub statistics with TTY-aware color indicators.

```
brain v1.2.0  linux/amd64  abc1234

Hub
  .brain/      found ✓
  Topics       5 files (63 KB)
  Sessions     10
  MEMORY.md    43 lines (OK)

Config
  Provider     openrouter
  Model        qwen/qwen3.5-9b
  API Key      configured
  Profile      agent

Daemon
  Status       running ●
  Queue        0 pending, 90 done, 0 failed
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
│   ├── behavior/                # Usage signals for self-learning (local only)
│   │   └── signals.json
│   ├── handoffs/                # Session handoffs (local only)
│   └── archived/                # Pruned entries (local only)
│
└── .brainprune                  # Patterns for knowledge pruning (optional)
```

---

## Installation

### Quick install (recommended)

```bash
# If agent-brain is already installed
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

Configuration can be **global** (shared across all projects) or **project-specific** (isolated to one project).

During `brain init`, you'll be prompted to choose:
- **Global config** (`~/.config/brain/config.yaml`) — Share LLM settings (provider, model, API key) across all projects. Best if you use the same LLM setup everywhere.
- **Project config** (`.brain/config.yaml`) — Isolate settings to this project only. Best if different projects need different providers, models, or API keys.

You can mix and match: some projects with global config, others with project-specific config. When running `brain config` inside a project, the project config takes precedence over the global config if it exists.

```bash
# Check which config is active
brain config

# Set a project-specific value (uses project config if it exists, otherwise global)
brain config set model anthropic/claude-3.5-haiku

# Switch a project from global to project-specific config
brain config set provider openai   # This creates .brain/config.yaml if it doesn't exist
```

Global config file (`~/.config/brain/config.yaml`):

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

Enable distributed tracing for debugging and observability via environment variables:

```bash
# Enable OTel tracing
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

# Or use stdout for debugging
export OTEL_EXPORTER_OTLP_ENDPOINT=stdout
```

When enabled, `brain` emits OpenTelemetry spans for CLI commands, daemon pipeline, and review sessions. Compatible with Jaeger, Grafana Tempo, Honeycomb, etc. See the [OpenTelemetry specification](https://opentelemetry.io/docs/) for details.

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
brain clean --patterns --dry-run   # See what can be archived
brain clean --patterns             # Archive stale entries
```

### "Duplicates detected in topic files"

```bash
brain clean --duplicates --dry-run   # Preview duplicate cleanup
brain clean --duplicates             # Remove duplicates
```

The `brain get --summary` command shows `⚠️ duplicates detected` when the same entry appears multiple times in a topic file or across files. `brain clean --duplicates` removes duplicates while keeping the first occurrence.

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
- [ ] **Optional:** After working, run `brain add --eval --good` to strengthen helpful memories

---

## License

MIT

---

## References & Inspiration

agent-brain draws from established research and best practices in software engineering, cognitive science, and AI systems:

### Knowledge Management & Organizational Memory

- **Knowledge-Centered Support (KCS)** — The practice of capturing knowledge at the point of use, rather than as a separate activity. agent-brain's "capture while coding" philosophy follows this principle. See the [Consortium for Service Innovation](https://www.thinkkm.org/kcs-adoption/kcs-principles/) for KCS principles.
- **Organizational Memory Systems** — Research on how teams retain and transfer knowledge over time. agent-brain's `.brain/` directory serves as a project-level organizational memory, preventing knowledge loss when team members (or agents) rotate. Key work: Walsh & Ungson (1991), ["Organizational Memory"](https://doi.org/10.2307/258607).

### Spaced Repetition & Memory Consolidation

- **Spaced Repetition** — The psychological finding that information is better retained when review is spaced over time. agent-brain's memory strength/decay system (`brain clean --decay`, half-life extension on retrieval) implements this digitally. See [Wozniak's research on spaced repetition](https://www.supermemo.com/en/archives1990-2015/en/tech/sm5).
- **Memory Consolidation Theory** — The neuroscience concept that memories are strengthened and reorganized during "offline" periods (sleep). agent-brain's `brain clean --decay` command and `brain add --eval --good/--bad` feedback loop mirror this biological process. See [Rasch & Born (2013)](https://doi.org/10.1152/physrev.00032.2012).

### Human-in-the-Loop AI

- **Human-in-the-Loop Machine Learning** — The practice of keeping humans in the decision loop for AI systems, especially for high-stakes or ambiguous cases. agent-brain's `brain daemon review` TUI and autonomy profiles (`guard`, `assist`, `agent`) implement this pattern. See [Amershi et al. (2014)](https://doi.org/10.1145/2556831) on software engineering guidelines for HCI.
- **Explainable AI (XAI)** — agent-brain's session journals, topic-tagged entries, and strength indicators make the system's "thinking" transparent and auditable. See the [DARPA XAI program](https://www.darpa.mil/program/explainable-artificial-intelligence) for foundational work.

### Developer Experience & Tool Design

- **Progressive Disclosure** — A UX principle where advanced features are hidden until needed. agent-brain's three-command surface (`init`, `get`, `add`) with optional advanced commands (`clean`, `daemon review`) follows this pattern. See [Miller's "Progressive Disclosure"](https://www.nngroup.com/articles/progressive-disclosure/) from Nielsen Norman Group.
- **Developer Flow State** — Research on minimizing interruptions during deep work. agent-brain's background daemon and automatic knowledge capture are designed to avoid pulling developers out of flow. See [Csikszentmihalyi's Flow Theory](https://doi.org/10.1037/0003-066X.54.10.824) and [Forsgren et al.'s DORA research](https://www.dora.dev/) on software delivery performance.

### Git & Version Control Best Practices

- **Post-Receive Hooks** — Using git hooks for automated analysis is a well-established pattern in CI/CD. agent-brain's pre-push hook follows the same design as tools like [pre-commit](https://pre-commit.com/) and [githooks](https://git-scm.com/docs/githooks).
- **Trunk-Based Development** — agent-brain is optimized for teams practicing trunk-based development with frequent commits, as recommended by [DORA research](https://www.dora.dev/four-keys/) and the [Accelerate book](https://www.oreilly.com/library/view/accelerate/9781484203439/).
