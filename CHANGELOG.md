# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.5.0] - 2026-04-04

### Fixed
- CLI help text reorganized for better UX consistency
- Config keys now in logical order (LLM → Review → Daemon → Analysis)
- Removed redundant verbose flags ([--json], [--dry-run], [--all]) from help text

## [v0.4.0] - 2026-04-04

### Added
- Config key registry: friendly keys like `api-key` instead of `llm.api_key`
- New config subcommands: `get`, `set`, `list`, `reset`, `setup`
- Interactive setup wizard: `brain config setup`
- Backward compatibility with dot-path notation

### Changed
- Split 1392-line main.go into 13 per-command files
- Service tests: 0% → 35.7% coverage
- Command tests: 4.8% → 13.4% coverage
- GitHub Actions CI/CD workflows

### Fixed
- BSD terminal support
- stdin timeout issue
- Config reload interval

## [v0.3.1] - 2026-04-03

### Added
- Human-in-the-loop review system (`brain review`) with interactive TUI
- Autonomy profiles: `guard`, `assist`, `agent` for configurable auto-accept
- Pending entry queue — daemon writes to `.brain/pending/` instead of topic files directly
- Pre-push git hook for analyzing stable code
- `brain review --all` to retroactively import existing topic entries into review queue
- Self-update via `brain update` with GitHub Releases binary assets
- Platform-specific binaries: Linux/Darwin/Windows × amd64/arm64

### Fixed
- Windows service stub signature mismatch (1→2 params)
- TUI `r` key (reject) now functional
- Agent profile auto-accept now respects config
- Self-update archive detection (was passing URL as filename)
- Git history anonymized (personal email removed)

### Changed
- goreleaser `draft: false` for auto-published releases
- README updated with review flow diagram and autonomy profiles

## [v0.2.0] - 2026-04-02

### Added
- Daemon with queue processing and backoff retry
- `brain init`, `brain get`, `brain search`, `brain add`, `brain status`
- `brain daemon start|stop|status|run`
- `brain config` for LLM and daemon settings
- `brain prune` for archiving stale entries
- `brain eval` for session evaluations
- LLM-powered commit analysis via OpenRouter
- Secret scanning before sending diffs
- Cross-platform support (Linux, macOS, Windows)

### Changed
- Extracted LLM analysis into `internal/analyzer`
- Dependency injection for daemon process testing
