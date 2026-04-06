# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
