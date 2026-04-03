# CLI UX Improvements Design

## Problem

The current CLI has several usability issues:
1. Config keys use internal YAML paths (`llm.api_key`, `daemon.poll_interval`) that users can't discover
2. No way to list available config keys or see what can be configured
3. No interactive setup for first-time users
4. No way to reset config to defaults
5. `brain config set` without args gives a cryptic usage message

## Solution

Full restructuring of `brain config` with friendly keys, discoverability, and interactive setup.

## Architecture

### New Config Key Registry

A single source of truth mapping friendly keys to their internal paths, types, defaults, and descriptions. Lives in `internal/config/registry.go`.

```go
type ConfigKey struct {
    Friendly    string  // "api-key"
    DotPath     string  // "llm.api_key"
    Type        string  // "string", "duration", "int", "enum"
    Default     string  // ""
    Description string  // "OpenRouter API key"
    EnvVar      string  // "BRAIN_API_KEY" (optional)
    Options     []string // ["guard", "assist", "agent"] (for enum types)
}
```

### New Commands

| Command | Description |
|---------|-------------|
| `brain config` | Show current config (unchanged) |
| `brain config get <key>` | Get a single config value |
| `brain config set <key> <value>` | Set a config value (friendly or dot notation) |
| `brain config list` | List all available keys with descriptions and current values |
| `brain config reset [key]` | Reset one key or all to defaults |
| `brain config setup` | Interactive wizard for first-time setup |

### Friendly Key Mapping

| Friendly Key | Dot Path | Type | Default | Description |
|---|---|---|---|---|
| `api-key` | `llm.api_key` | string | "" | OpenRouter API key |
| `model` | `llm.model` | string | "anthropic/claude-3.5-haiku" | LLM model name |
| `provider` | `llm.provider` | string | "openrouter" | LLM provider |
| `profile` | `review.profile` | enum | "guard" | Review profile (guard/assist/agent) |
| `poll-interval` | `daemon.poll_interval` | duration | "5s" | Daemon poll interval |
| `max-retries` | `daemon.max_retries` | int | 3 | Daemon max retries |
| `retry-backoff` | `daemon.retry_backoff` | string | "exponential" | Retry backoff strategy |
| `max-diff-lines` | `analysis.max_diff_lines` | int | 2000 | Max diff lines for analysis |

### Backward Compatibility

- Old dot notation (`llm.api_key`) still works in `config set` and `config get`
- Key resolution: try friendly key first, fall back to dot path
- `brain config set` with 3+ args auto-detects as set command (no need for explicit `set`)

### `brain config list` Output

```
Configuration Keys
==================
  api-key          OpenRouter API key                    current: ****or-v1-xxx
  model            LLM model name                        current: anthropic/claude-3.5-haiku
  provider         LLM provider                          current: openrouter
  profile          Review profile (guard/assist/agent)   current: guard
  poll-interval    Daemon poll interval                  current: 5s
  max-retries      Daemon max retries                    current: 3
  retry-backoff    Daemon retry backoff                  current: exponential
  max-diff-lines   Max diff lines for analysis           current: 2000
```

### `brain config setup` Flow

1. Ask for API key (with option to skip)
2. Ask for model (numbered list with defaults, option for custom)
3. Ask for review profile (numbered list with descriptions)
4. Save and confirm

### Files to Create
- `internal/config/registry.go` — ConfigKey struct, registry, lookup functions
- `internal/config/registry_test.go` — Tests for key resolution

### Files to Modify
- `cmd/brain/cmd_config.go` — Add get/list/reset/setup subcommands, friendly key support
- `cmd/brain/cmd.go` — Update printUsage() with new commands
- `internal/config/config.go` — GetValue() function for dot path reads
- `README.md` — Update config examples

## Testing

- Unit tests for config key registry (lookup, validation)
- Unit tests for config get/set with friendly keys
- Unit tests for backward compatibility (dot notation still works)
- Integration test for `brain config setup` (mock stdin)
- Test `brain config list` output format
- Test `brain config reset` (single key and all)
