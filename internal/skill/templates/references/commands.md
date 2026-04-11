# Command Reference

Complete documentation for all `brain` CLI commands.

## Session Commands

### `brain get <topic>`

Retrieves knowledge from the `.brain/` hub.

**Topics:** `memory`, `gotchas`, `patterns`, `decisions`, `architecture`, `all`

**Flags:**
- `--json` — Output as JSON
- `--summary` — Compact output (fewer tokens)
- `--focus "<topic>"` — Filter by topic (ui/backend/infrastructure/database/security/testing/architecture/general)

**Examples:**
```bash
brain get all
brain get gotchas
brain get all --focus "security"
brain get patterns --json
```

### `brain search <query>`

Searches across all knowledge entries.

**Flags:**
- `--json` — Output as JSON
- `--topic "<topic>"` — Search within a specific topic
- `--area "<area>"` — Filter by area tag

**Examples:**
```bash
brain search "auth" --topic "security"
brain search "database migration"
brain search "pagination" --json
```

### `brain add <topic> "<message>"`

Adds a new entry to the knowledge hub.

**Topics:** `gotcha`, `pattern`, `decision`, `architecture`, or any area tag (ui/backend/infrastructure/database/security/testing/architecture/general)

**Flags:**
- `--wm` — Add to working memory (temporary, decays over time)

**Examples:**
```bash
brain add gotcha "Redis pool defaults to 10 connections"
brain add pattern "All handlers use logging -> auth -> rate-limit middleware"
brain add infrastructure "Deployed on Ubuntu 22.04 LTS"
brain add security gotcha "JWT secret rotated weekly via cron"
brain add --wm "Investigating the auth timeout issue"
```

### `brain eval`

Ends the session, writes a self-evaluation, and creates a handoff.

**Flags:**
- `--good` — Mark the session's recommendations as successful (positive outcome)
- `--bad` — Mark the session's recommendations as problematic (negative outcome)

**Examples:**
```bash
brain eval
brain eval --good
brain eval --bad
```

## Maintenance Commands

### `brain status`

Shows hub statistics and health metrics.

**Flags:**
- `--json` — Output as JSON

**Output includes:**
- Total entries by topic
- Session count and handoff chain length
- Memory decay status
- Daemon health
- Knowledge hub size

### `brain review`

Opens an interactive TUI to review pending daemon-analyzed entries.

**Flags:**
- `--all` — Review all pending entries (not just latest)
- `--yes` / `-y` — Auto-approve all entries
- `--tty` — Force TUI mode

**Workflow:**
1. Entries appear as "pending" after daemon analysis
2. Run `brain review` to approve or reject each entry
3. Approved entries are added to topic files
4. Rejected entries are archived

### `brain prune [--dry-run]`

Archives stale entries based on configured prune patterns.

**Flags:**
- `--dry-run` — Preview what would be archived without making changes

**Configuration:**
Patterns are defined in `.brain/prune.conf` (one pattern per line). Entries matching any pattern are archived to `.brain/archived/`.

**Examples:**
```bash
brain prune --dry-run   # Preview
brain prune             # Execute
```

### `brain dedup [--dry-run]`

Finds and removes duplicate entries.

**Flags:**
- `--dry-run` — Preview duplicates without removing

**Deduplication logic:**
- Exact text matches are merged
- Near-duplicates (90%+ similarity) are flagged for review
- Older entries are archived; newer entries are kept

### `brain sleep`

Consolidates memory: applies decay to working memory entries and archives old sessions.

**Flags:**
- `--dry-run` — Preview consolidation without making changes

## Config Commands

### `brain config list`

Lists all configuration settings (global and project-specific).

### `brain config get <key>`

Gets a specific configuration value.

### `brain config set <key> <value>`

Sets a configuration value.

**Common keys:**
- `llm.api-key` — OpenRouter API key
- `llm.model` — Model name for analysis
- `daemon.enabled` — Enable/disable daemon
- `daemon.interval` — Queue processing interval

### `brain config reset <key>`

Resets a configuration key to its default value.

### `brain config setup`

Interactive setup wizard for initial configuration.

## Advanced Commands

### `brain daemon <action>`

Controls the background daemon service.

**Actions:** `start`, `stop`, `restart`, `status`, `failed`, `retry`, `run`

**Examples:**
```bash
brain daemon start
brain daemon status
brain daemon failed    # Show failed queue items
brain daemon retry     # Retry failed items
brain daemon run       # Force one processing cycle
```

### `brain doctor`

Runs a full health check and diagnostics.

**Checks:**
- Git repository status
- `.brain/` directory structure
- Daemon service status
- Config validity
- LLM API connectivity
- Queue health
- File permissions

### `brain index rebuild`

Rebuilds the metadata index from scratch.

Use when the index becomes corrupted or after manual file edits.

### `brain update`

Updates the `brain` binary to the latest version.

**Process:**
1. Fetches latest release from GitHub
2. Downloads the platform-specific archive
3. Replaces the binary atomically
4. Stops the current daemon (restart manually)

**Note:** Skill files are NOT auto-updated. Run `brain skill diff` to check for updates, then `brain skill update` to apply.

### `brain version`

Shows version information including build date and commit hash.

### `brain wm`

Working memory operations.

**Subcommands:**
- `brain wm list` — List working memory entries
- `brain wm clear` — Clear all working memory
- `brain wm add "<message>"` — Add to working memory

### `brain handoff`

Manages session handoffs.

**Subcommands:**
- `brain handoff list` — List recent handoffs
- `brain handoff show <session-id>` — Show a specific handoff
- `brain handoff export` — Export handoffs to a file

### `brain outcome`

Tracks outcomes of recommendations.

**Subcommands:**
- `brain outcome list` — List tracked outcomes
- `brain outcome add <session-id> <rating>` — Record an outcome (good/bad)

## Skill Commands

### `brain skill list`

Shows installed skill locations and their versions.

**Output:**
- Project-local paths (`.opencode/skills/`, `.claude/skills/`, `.agents/skills/`)
- Global paths (`~/.config/opencode/skills/`, `~/.claude/skills/`, `~/.agents/skills/`)
- Version of each installed skill
- Git status (modified/uncommitted)

### `brain skill diff`

Compares installed skill files against the embedded templates (latest version).

**Output:**
- Unified diff for each changed file
- Summary of added/removed/modified sections
- Version comparison (installed vs available)

### `brain skill update`

Updates skill files from the embedded templates.

**Behavior:**
- Warns about uncommitted git changes
- Requires confirmation before overwriting
- Preserves git history (changes appear in `git diff`)

### `brain skill install [--global]`

Installs skill files to project-local or global directories.

**Flags:**
- `--global` — Install to global directories instead of project-local

**Default locations:**
- Project: `.opencode/skills/agent-brain/`, `.claude/skills/agent-brain/`, `.agents/skills/agent-brain/`
- Global: `~/.config/opencode/skills/agent-brain/`, `~/.claude/skills/agent-brain/`, `~/.agents/skills/agent-brain/`

## Area Taxonomy

When using `brain add`, use these area tags:

| Area | Use For |
|------|---------|
| `ui` | Frontend, styling, UX, components, responsive design, accessibility |
| `backend` | API, business logic, services, handlers, middleware, server-side |
| `infrastructure` | Deployment, VPS, CI/CD, Docker, cloud, monitoring, networking |
| `database` | Schemas, migrations, queries, indexes, ORM, data modeling |
| `security` | Auth, secrets, permissions, OWASP, encryption, input validation |
| `testing` | Unit tests, integration tests, e2e, mocks, fixtures, test runners |
| `architecture` | Module structure, design patterns, data flow, system design |
| `general` | Cross-cutting knowledge, project-wide conventions, tooling |
