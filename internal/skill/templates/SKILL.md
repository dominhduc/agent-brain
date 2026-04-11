---
name: agent-brain
description: |
  Load project knowledge, record learnings, and manage AI agent memory.
  Use at session start (brain get all), before debugging (brain get gotchas),
  when corrected (brain add gotcha "..."), and at session end (brain eval).
license: MIT
compatibility: Requires the brain CLI. Works with OpenCode, Claude Code, Cursor, and other Agent Skills-compatible tools.
allowed-tools: Bash Read Grep Glob Edit Write
---

# agent-brain Skill

This skill gives you persistent memory across AI agent sessions via the `.brain/` knowledge hub.

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
brain search "auth" --topic "security"
brain search "database migration"
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
brain eval
```

This writes a self-evaluation to the current session file, records what you did, what worked, what failed, and creates a handoff for the next session.

With outcome feedback:
```bash
brain eval --good    # Recommendation was successful
brain eval --bad     # Recommendation caused issues
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

### Get & Search
```bash
brain get <topic>              # Topics: memory, gotchas, patterns, decisions, architecture, all
brain get all --focus "<topic>" # Load knowledge filtered by topic
brain search "<query>"          # Search all knowledge
brain search "<query>" --topic "<topic>"  # Search within a topic
```

### Add & Eval
```bash
brain add <topic> "<message>"         # Add entry to a topic
brain add <area> <topic> "<message>"  # Add entry with area tag (ui/backend/infrastructure/etc.)
brain add --wm "<message>"            # Add to working memory (temporary, decays)
brain eval                            # Session evaluation + handoff
brain eval --good                     # Mark recommendation as successful
brain eval --bad                      # Mark recommendation as problematic
```

### Maintenance
```bash
brain status               # Hub statistics & health
brain review               # Review pending daemon entries
brain prune [--dry-run]    # Archive stale entries
brain dedup [--dry-run]    # Find and remove duplicate entries
brain sleep                # Consolidate memory (decay + archive)
```

### Config
```bash
brain config list          # List all settings
brain config get <key>     # Get a value
brain config set <key> <value>  # Set a value
brain config reset <key>   # Reset to default
brain config setup         # Interactive setup wizard
```

### Advanced
```bash
brain daemon <action>      # Actions: start, stop, restart, status, failed, retry, run
brain doctor               # Health check & diagnostics
brain index rebuild        # Rebuild metadata index
brain update               # Update to latest version
brain version              # Show version info
brain skill diff           # Show skill updates vs templates
brain skill update         # Update skill files to latest version
```

## Autonomy Profiles

The skill supports different autonomy levels:

- **Guard mode** (`brain guard`): Maximum safety — destructive command warnings + directory-scoped edits
- **Careful mode** (`brain careful`): Safety warnings for destructive commands only
- **Assist mode** (default): Normal operation with standard safeguards
- **Agent mode**: Full autonomy for trusted workflows

## Skill Management

```bash
brain skill list           # Show installed skill locations and versions
brain skill diff           # Compare installed files vs latest templates
brain skill update         # Update skill files (overwrites with confirmation)
brain skill install --global  # Install to global directories
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
4. **Run `brain eval` every session** — Even for small sessions. The handoff is invaluable for continuity.
5. **Search before writing** — `brain search "pagination"` before implementing pagination from scratch.
6. **Review pending entries** — Run `brain review` periodically to approve/reject daemon-analyzed entries.
