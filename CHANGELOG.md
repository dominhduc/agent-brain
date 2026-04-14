# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.2.0] - 2026-04-14

### Added
- `--compact` flag for `brain get`: one line per entry with relative timestamps, no blank lines (~70% output reduction)
- `--message-only` flag for `brain get`: pure message text with no timestamps, scores, or metadata — ideal for AI agent context windows
- `--full` flag for `brain get all`: complete dump (default is now tiered view)
- Structured JSON output: `brain get <topic> --json` now returns `{topic, entry_count, entries: [{timestamp, message}]}` instead of raw file content
- Tiered `brain get all` display: overview → recent entries (7-day window) → top entries, with total count
- Search results grouped by topic with markdown syntax stripped
- Strength score legend printed at top of default output
- TTY-aware color indicators in `brain status` (green ✓, red ✗, yellow ●)

### Changed
- Stripped `●` prefix from strength scores in all output
- Stripped `### [timestamp]` markdown prefix from search and focused output
- `brain get all` default now shows scannable tiered view instead of full dump
- Help text updated with all new flags

### Fixed
- JSON timestamp parsing: timestamps now correctly formatted as `YYYY-MM-DD HH:MM:SS` without bracket artifacts

## [v1.1.1] - 2026-04-14

### Added
- `brain skill update` now prompts for confirmation before overwriting modified skill files, listing which files have local changes

### Changed
- Help banner uses `fmt.Printf` with `%-64s` format for proper alignment, truncates long version strings to 64 chars
- Makefile now injects `version`, `commit`, and `date` via `-ldflags` using `git describe --tags` for automatic versioning

## [v1.1.0] - 2026-04-14

### Changed
- **Consolidated internal packages**: eliminated 8 duplicate packages (22→14), removing ~1,800 lines of redundant code. All imports now use `internal/knowledge/` and `internal/session/` exclusively
- Deleted packages: `internal/analyzer/`, `internal/secrets/`, `internal/index/`, `internal/review/`, `internal/handoff/`, `internal/wm/`, `internal/outcome/`, `internal/brain/`
- Added standalone function wrappers in `knowledge/` and `session/` packages for backward-compatible API without requiring Hub instantiation

### Fixed
- Skill template inaccuracies: wrong prune config path (`.brainprune` not `.brain/.brainprune`), removed nonexistent `--area` flag, fixed deprecated command docs
- README OTel configuration: use environment variables instead of `brain config set otel.*`
- Skill template autonomy profiles section referenced wrong command syntax

### Added
- Tests for `internal/knowledge/behavior.go`: load/save, track command/topic/search/eval, corrupt file handling (10 tests)
- Tests for `internal/skill/skill.go`: install, skip existing, detect installed/modified, extract adaptations, generate diff (11 tests)

## [v1.0.0] - 2026-04-14

### Added
- **Self-learning skill adaptation**: `brain skill reflect` generates usage-based adaptations from behavior signals, search queries, and eval outcomes. Adaptations are appended to Agent Skills and preserved across `brain skill update`
- **Unified knowledge core**: `internal/knowledge/` package consolidates brain, index, review, outcome, and working memory into a single Hub struct
- **Session lifecycle package**: `internal/session/` for handoff and eval management
- **Behavior signal tracking**: `internal/knowledge/behavior.go` tracks command counts, topic access, search queries, and eval outcomes in `.brain/behavior/signals.json`
- **Collapsed daemon ecosystem**: LLM analysis and secret scanning moved into `internal/daemon/` — eliminates deep dependency chains across analyzer/secrets/review packages
- Config scope choice during `brain init`: choose between global config (shared across projects) or project-specific config (isolated in `.brain/config.yaml`)
- `brain config` now shows which config source is active (global or project)
- Project config takes precedence over global config when both exist

### Changed
- `brain skill update` now uses merge-based updates that preserve adaptation markers during template updates
- `brain config set/get/list/reset` now check for project config first, fall back to global
- Daemon uses local `Analyze()`, `Finding`, and `AnalyzeRequest` types instead of importing from separate packages

## [v0.17.3] - 2026-04-06

### Added
- `brain status` now shows commit hash and update available notification

## [v0.17.2] - 2026-04-06

### Fixed
- TUI EOF fallback: when TUI can't read stdin, falls back to line-buffered review instead of showing error

## [v0.17.1] - 2026-04-06

### Fixed
- `--tty` flag no longer forces TUI when stdin is not a terminal
- Line-buffered fallback is now the default for non-TTY sessions (no `--tty` needed)
- `brain status` and `brain review` now use the same counting method for pending entries

## [v0.17.0] - 2026-04-06

### Added
- Line-buffered fallback for `brain review` when TUI is not available (stdin is /dev/null)
  - Prompts user per-entry with y/n/q/a options
  - Works in non-TTY environments where TUI can't run
  - `y` = accept, `n` = reject, `a` = accept all, `q` = quit (leave rest pending)
  - Empty input defaults to `y` (accept), stdin EOF defaults to `y` (accept)
- Added `--tty` flag to force TUI mode in `brain review`

## [v0.16.6] - 2026-04-06

### Fixed
- `brain review` TTY detection: add `os.Stdin.Stat()` check for `ModeCharDevice` to correctly detect `/dev/null` stdin in tmux/SSH sessions

## [v0.16.5] - 2026-04-06

### Fixed
- `brain review` TTY detection: use `golang.org/x/term.IsTerminal()` instead of non-blocking read (which returned false when no key was pressed)

## [v0.16.4] - 2026-04-06

### Fixed
- `brain review` TTY detection: use `select()` to verify stdin is actually readable before entering TUI mode

## [v0.16.3] - 2026-04-06

### Fixed
- `brain review` now auto-accepts entries when stdin is closed without interaction (SSH edge case)

## [v0.16.2] - 2026-04-06

### Fixed
- Use `golang.org/x/term.IsTerminal()` for reliable TTY detection in SSH/non-TTY environments
- Silent exit on EOF instead of printing error message

## [v0.16.1] - 2026-04-06

### Fixed
- `brain review` EOF error: Handle `io.EOF` gracefully in TUI input loop instead of treating it as an error
- Improved `CanUseRawMode()` to check stdin is a character device before attempting TCGETS
- Graceful exit message when stdin is closed during interactive review

## [v0.16.0] - 2026-04-06

### Added
- `brain get --summary` flag for compact topic overview with entry counts and duplicate warnings
- `brain get --json` now works with both summary and full output modes
- Automatic deduplication on `brain add` — skips adding entries with duplicate normalized content

### Changed
- `brain get all` now includes overview section with topic stats and deduplicates displayed content

## [v0.15.0] - 2026-04-05

### Fixed
- PostJSON double-marshal: BuildBody() returned []byte (already JSON), PostJSON() re-marshaled it, sending base64-encoded garbage to LLM APIs
- Explicit JSON schema in LLM prompt for reliable structured output from models like qwen
- OpenRouter ParseResponse now handles reasoning and refusal fields from thinking models
- Config wiping: tests no longer write to real config file (XDG_CONFIG_HOME isolation)
- `brain review` detects non-TTY upfront, auto-accepts gracefully without TCGETS errors
- Added `--yes` flag to `brain review` for explicit non-interactive auto-accept

## [v0.14.0] - 2026-04-04

### Changed
- Simplified LLM prompts further to avoid JSON parsing failures

## [v0.13.1] - 2026-04-04

### Fixed
- Config Load() handles corrupted/invalid config files gracefully (returns defaults with warning)

## [v0.13.0] - 2026-04-04

### Fixed
- Config setup wizard step numbering fixed for ollama
- LLM JSON parsing: simplified prompts, clearer format instructions, added fallback for missing confidence
- Response parsing returns error if no JSON found

### Changed
- Streamlined system/user prompts for better LLM compliance

## [v0.12.0] - 2026-04-04

### Fixed
- AGENTS.md template now matches repo
- Config key name unified: all messages use `api-key` (not `llm.api_key`)
- First commit now works: daemon uses empty-tree fallback when `HEAD~1` doesn't exist
- `brain doctor` now checks daemon status

### Changed
- `brain daemon run` documented in help text

## [v0.11.0] - 2026-04-04

### Changed
- `brain status` now shows unified view: hub stats, config, daemon state, and health warnings

## [v0.10.0] - 2026-04-04

### Added
- `brain daemon restart` and `brain daemon retry`

## [v0.9.0] - 2026-04-04

### Added
- `brain daemon failed` — list failed queue items with error reasons
- Failed items now store error reason in JSON

### Fixed
- Failed items no longer keep `.processing` suffix in failed directory
- Config setup preserves existing API key when user skips key prompt

## [v0.8.2] - 2026-04-04

### Fixed
- Config setup now preserves existing API key when user skips the key prompt
- Previously `brain config setup` started from DefaultConfig(), wiping the API key

## [v0.8.1] - 2026-04-04

### Fixed
- Per-project daemon lock: lock file includes project hash for parallel daemons
- Queue counting filters to `commit-*.json` only
- `findCurrentProjectBrainDir` uses filepath.Join for cross-platform safety

## [v0.8.0] - 2026-04-04

### Changed
- Config setup wizard flow: Provider → Base URL → API Key → Model → Profile
- Named custom providers with custom_providers config section

### Fixed
- Config setup wizard: prompt for base-url when custom provider selected

## [v0.7.0] - 2026-04-04

### Added
- `brain doctor` health check command
- Multi-project daemon support

## [v0.6.0] - 2026-04-04

### Added
- Multi-provider support: OpenAI, Anthropic, Google Gemini, Ollama, Custom
- Provider-specific model suggestions in setup wizard

### Changed
- `provider` config key is now enum type

## [v0.5.0] - 2026-04-04

### Fixed
- CLI help text reorganized for better UX consistency
- Config keys in logical order

## [v0.4.0] - 2026-04-04

### Added
- Config key registry with friendly keys
- Interactive setup wizard: `brain config setup`
- Config subcommands: `get`, `set`, `list`, `reset`

### Changed
- Split 1392-line main.go into 13 per-command files

## [v0.3.1] - 2026-04-03

### Added
- Human-in-the-loop review system (`brain review`) with interactive TUI
- Autonomy profiles: `guard`, `assist`, `agent`
- Self-update via `brain update` with GitHub Releases binary assets
- Platform-specific binaries: Linux/Darwin/Windows x amd64/arm64

## [v0.2.0] - 2026-04-02

### Added
- Daemon with queue processing and backoff retry
- `brain init`, `brain get`, `brain search`, `brain add`, `brain status`
- LLM-powered commit analysis via OpenRouter
- Secret scanning before sending diffs
- Cross-platform support (Linux, macOS, Windows)

### Changed
- Extracted LLM analysis into `internal/analyzer`
- Dependency injection for daemon process testing
