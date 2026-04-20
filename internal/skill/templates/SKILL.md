---
name: agent-brain
description: |
  Load project knowledge, record learnings, and manage AI agent memory.
  Use at session start (brain get all), before debugging (brain get gotchas),
  when corrected (brain add gotcha "..."), and at session end (brain add --eval).
license: MIT
compatibility: Requires the brain CLI. Works with OpenCode, Claude Code, Cursor, and other Agent Skills-compatible tools.
allowed-tools: Bash Read Grep Glob Edit Write
---

# agent-brain Skill

This skill gives you persistent memory across AI agent sessions via the `.brain/` knowledge hub.

## How Knowledge Is Stored

- `.brain/` is your **local working copy** — gitignored, never committed
- `docs/brain/` is the **tracked shareable copy** — committed to git for teammates
- `brain sync` exports knowledge from `.brain/` to `docs/brain/` (auto-runs on push)
- `brain init` imports from `docs/brain/` on fresh clones

## Session Workflow

### 1. Session Start — Load Knowledge

```bash
brain get all
```

This loads accumulated project knowledge: gotchas, patterns, architecture decisions, and active memory. Read the output before writing any code.

For focused loading:
```bash
brain get all --focus "security"
brain get gotchas
brain get patterns
```

### 2. During Work — Search Before Writing

Before writing code against unfamiliar patterns:
```bash
brain get "auth" --topic "security"
brain get "database migration"
```

When you discover something new, record it immediately:
```bash
brain add gotcha "Redis connection pool defaults to 10; set max_connections in config"
brain add pattern "All API handlers use the middleware chain: logging -> auth -> rate-limit"
brain add architecture "User service communicates with auth via gRPC, not HTTP"
```

### 3. When Corrected — Persist Learnings

When the user corrects you or you hit a repeated mistake:
```bash
brain add gotcha "The fix for issue #42"
brain add pattern "Always use the repository pattern for database access"
```

This ensures the next agent session won't repeat the same mistake.

### 4. Session End — Evaluate and Handoff

```bash
brain add --eval
```

This writes a self-evaluation to the current session file, records what you did, what worked, what failed, and creates a handoff for the next session.

With outcome feedback:
```bash
brain add --eval --good    # Recommendation was successful
brain add --eval --bad     # Recommendation caused issues
```

## Topic Taxonomy

Use these 8 topics when adding entries:

| Topic | Keywords | Examples |
|-------|----------|----------|
| `ui` | frontend, styling, UX, components, responsive, accessibility | React components, CSS patterns, form validation |
| `backend` | API, business logic, services, handlers, middleware | REST endpoints, auth middleware, rate limiting |
| `infrastructure` | deployment, VPS, CI/CD, Docker, cloud, monitoring | Docker configs, GitHub Actions, Nginx setup |
| `database` | schemas, migrations, queries, indexes, ORM | PostgreSQL migrations, query optimization |
| `security` | auth, secrets, permissions, OWASP, encryption | JWT handling, API key rotation, CORS config |
| `testing` | tests, mocks, fixtures, e2e, unit | Test patterns, mock strategies, CI test runs |
| `architecture` | module structure, patterns, data flow, design | Service boundaries, event-driven patterns |
| `general` | cross-cutting knowledge, project-wide conventions | Code style, git workflow, naming conventions |

## Commands

### Core
```bash
brain get <topic>              # Topics: all, gotchas, patterns, decisions, architecture, memory
                               # Auto-searches if not a known topic
brain get all --focus "<topic>" # Load knowledge filtered by topic
brain get "<query>"            # Search if not a known topic
brain add <topic> "<message>"  # Add entry to a topic
brain add <area> <topic> "<message>"  # Add entry with area tag
brain add --wm "<message>"     # Add to working memory (temporary)
brain add --eval               # Session evaluation + handoff
brain add --eval --good        # Mark recommendation as successful
brain add --eval --bad         # Mark recommendation as problematic
```

### Maintenance
```bash
brain clean                # Run all cleanup (prune + dedup + decay + rebuild)
brain clean --dry-run      # Preview all cleanup actions
brain clean --patterns     # Archive entries matching .brainprune patterns
brain clean --duplicates   # Remove exact duplicate entries
brain clean --duplicates --fuzzy  # Also catch near-duplicates
brain clean --decay        # Archive strength-decayed entries
brain clean --rebuild      # Rebuild metadata index
brain doctor               # Hub statistics, health check & diagnostics
brain doctor --json        # Machine-readable output
brain doctor --fix         # Auto-repair (rebuild index, requeue failed items)
```

### Sync
```bash
brain sync                 # Export topic files to docs/brain/ (for sharing)
brain sync --import        # Import from docs/brain/ into .brain/
brain sync --dry-run       # Preview without writing files
```

`.brain/` is gitignored. `brain sync` copies topic files to `docs/brain/` so they're tracked in git. The pre-push hook auto-syncs before every push. On fresh clones, `brain init` imports from `docs/brain/` automatically.

### Daemon
```bash
brain daemon start         # Register and start the background daemon
brain daemon stop          # Stop the daemon
brain daemon status        # Show daemon state and queue counts
brain daemon review        # Interactive review of pending entries
brain daemon review --yes  # Auto-accept all pending entries
brain daemon failed        # List failed queue items
brain daemon retry         # Requeue all failed items
brain daemon run           # Run daemon in foreground
```

### Config
```bash
brain config list          # List all settings
brain config get <key>     # Get a value
brain config set <key> <value>  # Set a value
brain config reset <key>   # Reset to default
brain config setup         # Interactive setup wizard
```

### Update
```bash
brain update               # Update brain binary to latest version
brain update --skills      # Update skill files (preserves adaptations)
brain update --skills --list   # Show installed skill locations
brain update --skills --diff   # Compare installed vs latest templates
brain update --skills --install  # Install skill files to project directories
brain update --skills --install --global  # Install to global directories
brain update --skills --reflect [--dry-run]  # Generate skill adaptations
```

## Autonomy Profiles

The skill supports different autonomy levels, configured via `brain config set profile <name>`:

- **Guard** (default): No auto-accept, no auto-dedup — all entries reviewed
- **Assist**: Auto-dedup yes, auto-accept no — less noise
- **Agent**: Auto-dedup yes, auto-accept yes — fully automatic

## Skill Management

```bash
brain update --skills --list    # Show installed skill locations
brain update --skills --diff    # Compare installed files vs latest templates
brain update --skills           # Update skill files to latest version
brain update --skills --install --global  # Install to global directories
```

## Supporting Files

For detailed reference material, see the bundled files:

- [Command Reference](references/commands.md) — Complete command documentation with all flags
- [Troubleshooting](references/troubleshooting.md) — Error diagnostics and fixes
- [Topic Taxonomy](references/taxonomy.md) — Detailed topic descriptions and examples

## Best Practices

1. **Always start with `brain get all`** — Never skip this. The accumulated knowledge prevents repeated mistakes.
2. **Record learnings immediately** — Don't wait until session end. Add gotchas and patterns as you discover them.
3. **Use specific topic tags** — `brain add security "JWT expires in 15min"` is better than `brain add general "auth stuff"`.
4. **Run `brain add --eval` every session** — Even for small sessions. The handoff is invaluable for continuity.
5. **Search before writing** — `brain get "pagination"` before implementing pagination from scratch.
6. **Review pending entries** — Run `brain daemon review` periodically to approve/reject daemon-analyzed entries.
7. **Run `brain sync` before important work** — Ensures `docs/brain/` is up to date so teammates get your latest knowledge.
