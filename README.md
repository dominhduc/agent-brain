# 🧠 agent-brain

> **Give your AI coding agent a memory that grows smarter with every session.**

`agent-brain` (CLI command: `brain`) is a CLI tool that creates a per-project knowledge hub (`.brain/`) for AI coding agents. It tracks what your agent learns, auto-analyzes commits via LLM, and feeds accumulated knowledge back into every new session.

**Works with** OpenCode, Claude Code, Cursor, Windsurf — any agent that can run shell commands.

---

## Table of Contents

- [For AI Agents](#for-ai-agents-start-here)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Commands](#commands)
- [Topic Taxonomy](#the-8-topic-taxonomy)
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

### What's New in v3

| Version | Highlights |
|---------|-----------|
| **v3.0.1** | Restored `brain search` alias, fixed trace dry-run wording, grade progress output |
| **v3.0.4** | Fix: `brain update` on Termux, hardened Termux detection, clearer `--once` output |
| **v3.0.3** | Termux support: auto-process on `brain get all`, `daemon run --once`, Termux-aware init |
| **v3.0.0** | `brain grade`, `brain trace`, LLM consolidation (`--llm`), contrastive extraction, memory feedback loops, adaptive daemon guidance |

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
                                          (auto brain sync)
                                                   │
                                          ┌────────▼────────┐
                                          │  docs/brain/    │
                                          │  (tracked)      │
                                          └────────┬────────┘
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
                                        │  files (local)      │
                                        └──────────┬──────────┘
                                                   │
                                      Next push: brain sync exports to docs/brain/
                                                   │
                                      Teammates pull and get shared knowledge
```

### The Three Layers

| Layer | What it does | Automatic? |
|-------|-------------|------------|
| **Git Hook** | Captures pushed commits and queues them | Yes — fires on every push |
| **Daemon** | Analyzes queued commits via LLM, writes to pending | Yes — runs in background |
| **Review Gate** | Human approves/rejects findings before they become permanent | Via `brain daemon review` |
| **Knowledge Sync** | Exports topic files to `docs/brain/` for sharing across machines | Yes — pre-push hook |
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
brain help              # Brief: commands, topics, workflows
brain help --full       # Complete reference: all flags, examples
```

### `brain init`

Initialize a knowledge hub in your project. Creates `.brain/` (gitignored), `AGENTS.md`, compatibility wrappers, git hooks (pre-push sync), and starts the daemon. If `docs/brain/` exists, imports teammate knowledge automatically.

```bash
cd your-project && brain init
```

### `brain get <topic|query>`

Retrieve accumulated knowledge. Entries show a strength indicator based on recency and usage. When the argument is not a known topic, auto-searches across all knowledge files.

```bash
brain get gotchas                          # Known pitfalls
brain get all                              # Budget-aware tiered view
brain get all --full                       # Everything
brain get all --context                    # Boost entries matching git diff
brain get all --budget 5000                # Custom token budget
brain get all --focus "infrastructure"     # Filter by topic
brain get gotchas --compact                # One line per entry
brain get gotchas --message-only           # No metadata
brain get gotchas --json                   # Structured JSON
brain get "auth"                           # Search across topics
brain get "database" --topic "security"    # Search with topic filter
```

**Output modes:** Default (budget-aware), `--full` (complete), `--compact` (one-liner), `--message-only` (bare text), `--json` (structured). Retrieval automatically strengthens memories.

**Topics:** `gotchas`, `patterns`, `decisions`, `architecture`, `memory`, `all`, `wm`

### `brain add <topic> "<message>"`

Add knowledge manually. Supports fuzzy dedup (skips near-duplicates).

```bash
brain add gotcha "Project uses argon2, NOT bcrypt"
brain add pattern "Tests use Vitest, not Jest"
brain add infrastructure gotcha "VPS uses Ubuntu 22.04"   # With area tag
brain add --wm "investigating auth bug"                    # Working memory
brain add --global pattern "Always use filepath.Join"      # Global store
brain add --eval                                           # Session evaluation
brain add --eval --good                                    # Positive feedback
brain add --eval --bad                                     # Negative feedback
```

Use an area prefix (`ui`, `backend`, `infrastructure`, `database`, `security`, `testing`, `architecture`, `general`) before the topic to tag entries.

### `brain grade`

Evaluate knowledge entries for accuracy, specificity, and generality via LLM. Returns keep/rewrite/archive verdicts.

```bash
brain grade                # Grade and apply verdicts
brain grade --dry-run      # Preview without modifying
```

### `brain trace <action>`

Capture reasoning traces from agent sessions. Subcommands: `step`, `save`, `extract`, `list`.

```bash
brain trace step --action "debug" --target "auth.go" --reasoning "Found nil pointer"
brain trace save --outcome success --goal "Fix auth nil pointer bug"
brain trace extract          # Extract knowledge via LLM
brain trace extract --dry-run
brain trace list
```

### `brain consolidate`

Find and merge related entries. Deterministic by default, LLM-powered with `--llm`.

```bash
brain consolidate --dry-run              # Preview
brain consolidate --apply                # Apply
brain consolidate --llm --dry-run        # LLM-powered semantic merge
brain consolidate --topic gotchas        # Filter by topic
```

### `brain clean`

Run cleanup operations: dedup, prune, decay, rebuild.

```bash
brain clean --duplicates --dry-run       # Preview exact duplicate removal
brain clean --duplicates --fuzzy         # Near-duplicate removal
brain clean --patterns --dry-run         # Prune via .brainprune patterns
brain clean --decay --dry-run            # Archive decayed entries
brain clean --rebuild                    # Rebuild metadata index
```

### `brain edit` / `brain supersede`

Manage entry lifecycle.

```bash
brain edit gotchas "2026-04-15 10" --message "Updated message"
brain supersede gotchas "2026-04-15 10:00:00" "2026-04-18 12:00:00"
```

### `brain daemon <action>`

Manage the background analysis daemon. Actions: `start`, `stop`, `restart`, `status`, `retry`, `run`, `review`.

```bash
brain daemon start              # Start background daemon
brain daemon stop               # Stop
brain daemon status             # Health check + queue depth
brain daemon review             # Interactive TUI to approve/reject
brain daemon review --all       # Re-review existing entries
brain daemon review --yes       # Auto-accept all
brain daemon retry              # Requeue failed items
brain daemon run --once         # Process queue, then exit
```

### `brain embed` / `brain sync`

Embedding search and cross-machine sharing.

```bash
brain embed                     # Embed new/stale entries
brain embed --all               # Re-embed everything
brain embed --status            # Coverage report
brain sync                      # Export to docs/brain/
brain sync --import             # Import from docs/brain/
```

### `brain config`

View or modify configuration.

```bash
brain config                    # Show config + source
brain config get api-key        # Single value (keys masked)
brain config set api-key <key>  # Set (project scope if .brain/ exists)
brain config set model anthropic/claude-3.5-haiku --global  # Force global
brain config list               # All available keys
brain config reset model        # Reset to default
brain config setup              # Interactive wizard
```

**Available keys:** `api-key`, `model`, `provider`, `profile`, `poll-interval`, `max-retries`, `retry-backoff`, `max-diff-lines`, `retrieval.max_tokens`, `retrieval.min_strength`, `retrieval.max_entries`, `retrieval.include_recent_days`

### `brain doctor [--json]`

Knowledge hub health check with TTY-aware output.

```
brain v3.0.1  linux/amd64  abc1234

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

Warnings
  ⚠ MEMORY.md not updated in 12 days
  ⚠ 3 pending entries awaiting review
```

### `brain update`

Update brain binary or manage Agent Skills.

```bash
brain update                            # Update to latest version
brain update --skills                   # Update skill files
brain update --skills --list            # Show installed skills
brain update --skills --diff            # Compare vs templates
brain update --skills --install         # Install to project
brain update --skills --install --global  # Install globally
brain update --skills --reflect         # Generate adaptations from usage
brain update --skills --reflect --dry-run
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

Control how much human review is needed when running `brain daemon review`:

| Profile | Auto-Dedup | Auto-Accept | Behavior |
|---------|-----------|-------------|----------|
| `guard` (default) | No | No | Every entry prompts for approval |
| `assist` | Yes | No | Deduplicates automatically, prompts for new entries |
| `agent` | Yes | Yes | Fully automatic — accepts all entries, deduplicates, no prompts |

When `profile` is set to `agent`, `brain daemon review` skips interactive prompts entirely and auto-accepts all pending entries (with deduplication). The `guard` and `assist` profiles require interactive review via TUI or line-buffered mode.

```bash
brain config set profile assist   # Auto-dedup, manual approval
brain config set profile agent    # Fully automatic (no prompts)
```

Config resolution follows project-first priority: `brain config set` writes to `.brain/config.yaml` when inside a project. All commands (`brain doctor`, `brain daemon review`, `brain config get`) read from the project config first, falling back to the global config at `~/.config/brain/config.yaml`.

---

## File Structure

```
your-project/
├── AGENTS.md                    # Instructions for coding agents
├── CLAUDE.md                    # "See AGENTS.md"
├── .cursorrules                 # "See AGENTS.md"
├── .windsurfrules               # "See AGENTS.md"
│
├── docs/brain/                  # Tracked knowledge (shared via git)
│   ├── gotchas.md               # Error patterns + fixes
│   ├── patterns.md              # Discovered conventions
│   ├── decisions.md             # Architecture decisions + rationale
│   └── architecture.md          # Module relationships
│
├── .brain/                      # Knowledge hub (gitignored, local only)
│   ├── MEMORY.md                # Main index (< 200 lines)
│   ├── gotchas.md               # Working copy (edited by daemon/agent)
│   ├── patterns.md              # Working copy
│   ├── decisions.md             # Working copy
│   ├── architecture.md          # Working copy
│   ├── sessions/                # Per-session journals
│   │   └── 2026-04-02T14-30-00.md
│   ├── .queue/                  # Daemon queue
│   │   ├── commit-*.json
│   │   └── done/
│   ├── pending/                 # Entries awaiting review
│   ├── traces/                  # Session reasoning traces
│   │   ├── current.json         # Active trace being built
│   │   └── extracted/           # Traces already extracted
│   ├── index.json               # Metadata index
│   ├── buffer/                  # Working memory
│   ├── behavior/                # Usage signals for self-learning
│   │   └── signals.json
│   ├── handoffs/                # Session handoffs
│   └── archived/                # Pruned entries
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

**Resolution priority:** All commands read project config first (`.brain/config.yaml` if it exists), then fall back to global config (`~/.config/brain/config.yaml`). This applies to `brain doctor`, `brain daemon review`, `brain config get`, and all other commands.

You can mix and match: some projects with global config, others with project-specific config. When running `brain config` inside a project, the project config takes precedence over the global config if it exists.

```bash
# Check which config is active
brain config

# Set a value — auto-detects project config if .brain/ exists
brain config set model anthropic/claude-3.5-haiku

# Force write to global config from inside a project
brain config set model anthropic/claude-3.5-haiku --global
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
- **Linux with systemd:** `systemd` via `~/.config/systemd/user/brain-daemon.service`
- **Linux without systemd** (proot, containers): Background process via `nohup` with log at `~/.cache/brain/brain-daemon-*.log`
- **Android/Termux:** Daemon registration is skipped. Use one of the Termux alternatives below.

The daemon starts on login and restarts on crash. On non-systemd Linux, `brain daemon start` backgrounds the process directly.

### Termux & Android

Android aggressively kills background processes, making persistent daemons unreliable. agent-brain handles this with three alternatives:

| Method | When | How |
|--------|------|-----|
| **Auto-process on get** | Every session start | `brain get all` automatically processes queued commits before returning knowledge |
| **One-shot processing** | On demand | `brain daemon run --once` processes all queued commits, then exits |
| **Foreground polling** | In a tmux session | `brain daemon run` polls continuously like on desktop |

The auto-process method is recommended — it requires zero extra commands. Since AI agents already run `brain get all` at session start, the queue is processed transparently.

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

On Termux, the daemon can't run persistently in the background. Instead:

```bash
brain daemon run --once    # Process queue and exit
brain get all              # Auto-processes queue before showing knowledge
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
- **Linux with systemd:** `journalctl --user -u brain-daemon`
- **Linux without systemd:** `~/.cache/brain/brain-daemon-*.log`

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
