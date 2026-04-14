# Decisions
<!-- Entries added by brain add decision or daemon analysis -->

### [2026-04-03 08:42:47] Extracted LLM analysis into internal/analyzer for testability — Analyze() takes AnalyzeRequest struct with configurable APIBaseURL, enabling httptest-based tests without real API calls


### [2026-04-03 08:42:47] Dependency injection for daemon.ProcessItemWithDeps — takes DiffGetter and AnalyzeFunc as function params, allowing tests to mock git diff and LLM calls


### [2026-04-03 08:42:47] Daemon PID lock uses flock on POSIX (auto-released on crash) and file-existence on Windows (best-effort, stale lock possible on crash)


### [2026-04-03 08:42:47] Self-update uses GitHub Releases API — Downloads platform-specific archive, extracts binary, replaces in-place with .bak backup. Supports both tar.gz and zip (Windows).


### [2026-04-03 08:42:47] Changed version from const to var so goreleaser ldflags can override it at build time — -X main.version={{.Version}}


### [2026-04-04 16:19:17] Removed 'custom' as a direct provider value. Users must name their custom providers (e.g. groq, together). This makes config more readable and allows switching between multiple custom endpoints.


### [2026-04-05 07:12:27] Support for non-JSON request bodies


### [2026-04-05 07:12:27] Support for pre-encoded byte slices in POST requests


### [2026-04-05 07:57:51] Users must name custom providers explicitly instead of using a generic 'custom' value.




### [2026-04-05 07:57:51] Require named custom providers instead of generic 'custom' value


### [2026-04-05 08:34:34] Unified API key configuration key renamed from nested llm.api_key to flat api-key.


### [2026-04-05 08:34:34] Named custom providers replaced single custom option to improve configurability.


### [2026-04-05 08:34:34] Auto-accept flag implemented to replace interactive TUI in CI/automated workflows.


### [2026-04-05 08:34:34] Project-specific daemon service names generated via unique hash lookup.


### [2026-04-05 08:34:34] Config setup wizard flow prioritized provider selection over base URL.


### [2026-04-05 08:34:34] Per-project daemon locks include project hashes for parallelization.






























### [2026-04-06 11:46:28] Added gitleaks allowlist for scanner/config test files




### [2026-04-06 11:46:28] Duplicate detection adds entries to a set before committing to prevent redundancy


### [2026-04-06 11:46:28] TopicSummary struct separates metadata (counts, duplicates) from actual content


### [2026-04-06 11:46:28] Normalize function uses strings.NewReplacer for whitespace normalization


### [2026-04-06 11:46:28] Split SPL|ALS gate between planning and execution phases with human approval


### [2026-04-06 11:46:28] Architecture decisions are documented in a dedicated decisions.md file for future reference


### [2026-04-06 11:46:28] Tool interface updated to support JSON output for machine-readable command parsing


### [2026-04-06 11:46:28] Added auto-deduplication to prevent duplicate entries on save.


### [2026-04-06 11:46:28] Integrated topic statistics into the main list view.






### [2026-04-06 13:43:38] Knowledge base uses separate MEMORY.md and PATTERNS.md files within .brain/
















### [2026-04-06 13:43:38] Silent exit on EOF instead of printing error messages for cleaner UX


### [2026-04-06 13:43:38] Removed 'exited cleanly' console output to reduce noise when review completes without actions




### [2026-04-06 14:06:58] Developer inserted test comment before variable block


### [2026-04-06 15:41:55] Developer logged the test comment insertion event.


### [2026-04-06 15:41:55] Adopted golang.org/x/term for reliable TTY detection.










### [2026-04-06 15:47:20] Line-buffered fallback is now default for non-TTY sessions (no --tty needed)




### [2026-04-06 15:47:20] Changed TUI condition to remove ttyFlag forcing TUI mode when stdin is not terminal






### [2026-04-06 15:47:20] TUI mode now only activates when !ttyFlag && canUseTTY








### [2026-04-06 15:47:20] --tty flag no longer forces TUI when stdin is not terminal






### [2026-04-06 15:48:51] Modified useTUI logic to disable forced TUI via flag.












### [2026-04-06 16:02:42] Switched from manual error reporting to automatic mode switching


### [2026-04-09 14:56:21] Implemented stdout exporter for debug visibility of emitted spans


### [2026-04-09 14:56:21] Recorded attributes only after completion (deferring measurements)


### [2026-04-09 14:56:21] Added fallback modes for TUI unavailability (auto, line-buffered)


### [2026-04-09 14:56:21] Minor version increment designated for release tracking


### [2026-04-09 14:56:21] Project renamed to `agent-brain` while keeping CLI command alias as `brain`.


### [2026-04-09 14:56:21] Background daemon pushes insights to a pending queue rather than direct permanent storage.


### [2026-04-09 14:56:21] OpenTelemetry tracing added to CLI commands and daemon pipeline for observability.


### [2026-04-09 14:56:21] Renamed CLI tool identifier from 'brain' to 'agent-brain' across files.


### [2026-04-09 14:56:21] Updated version string output to include 'v0.20.0' for tracking.


### [2026-04-09 14:56:21] Incremented patch version from v0.20.0 to v0.20.1


### [2026-04-09 14:56:21] Renamed tool identifier from 'agent-brain' to 'brain' for shorter display.


### [2026-04-09 14:56:21] Updated version string in code and banners during release management.


### [2026-04-09 14:56:21] CLI command name shortened from agent-brain to brain.




### [2026-04-09 14:56:21] Add dedup command as maintenance tool similar to prune


### [2026-04-09 14:56:21] Sorting by topic then line number provides deterministic duplicate removal


### [2026-04-09 14:56:21] Keep entry from alphabetically first topic in cross-topic scenarios


### [2026-04-09 14:56:21] Archive files with timestamp for audit trail


### [2026-04-09 14:56:21] Exit codes are strict (1 on error, 0 on success)


### [2026-04-09 14:56:21] Bumped version variable to reflect release v0.21.0


### [2026-04-09 14:56:21] Remove duplicates while maintaining the first occurrence to preserve context.


### [2026-04-09 14:56:21] Implement dry-run flags to preview changes before modifying the persistent store.


### [2026-04-12 06:20:41] Config scope choice during brain init: choose between global config (shared across projects) or project-specific config (isolated)
























### [2026-04-12 06:20:41] Brain initialize now checks for project config first before global config


### [2026-04-12 06:20:41] Config values can be mixed - some projects with global config, others with project-specific














### [2026-04-12 06:20:41] Automated config scope choice during brain init - global vs project


### [2026-04-12 06:20:41] Lazy config loading with resolveConfig() and resolveConfigForWrite() patterns


### [2026-04-12 06:20:41] Config source indication added to CLI output for clarity










### [2026-04-12 06:20:41] TUI mode activation conditional (!ttyFlag && canUseTTY)


### [2026-04-14 06:46:30] Dual-config system: global (~/.config/brain/) or project-specific (.brain/) configuration










### [2026-04-14 06:46:30] Config resolves project-local first, then falls back to global






### [2026-04-14 06:46:30] Config scope is resolved during initialize: global shared or project-specific isolated.




### [2026-04-14 06:46:30] TUI mode activates only when explicit TTY flag is unset and stdin is a terminal.


### [2026-04-14 06:46:30] Project config takes precedence over global config when both exist.


### [2026-04-14 06:46:30] Version variable consistent with CHANGELOG release version.
























### [2026-04-14 06:46:30] Internal config layer resolves config sources and handles reading/writing appropriately






### [2026-04-14 06:46:30] Config scope choice during init: users can choose global or project-specific configuration










### [2026-04-14 06:46:30] Use JSON format for persisting behavioral signals


### [2026-04-14 06:46:30] Tag adaptations with start/end markers for later extraction


### [2026-04-14 06:46:30] Rate-based filtering for top gotchas (strength < 0.5 or retrievals < 2)


### [2026-04-14 06:46:30] Thresholds for strong/weak entries (0.7 and 0.3 strength)


### [2026-04-14 06:46:30] Different counters for good/bad eval outcomes stored per session


### [2026-04-14 06:46:30] Daemon adopted local types for analyze and secrets to simplify dependency chains


### [2026-04-14 06:46:30] Session lifecycle functions are separated into internal/session/ package


### [2026-04-14 06:46:30] Daemon types are local to eliminate deep dependency chains across packages.


### [2026-04-14 06:46:30] Project config prevails over global config when both sources exist during init.


### [2026-04-14 06:46:30] Skill adaptations are preserved using merge-based updates during versioning.








### [2026-04-14 19:13:33] Added confirmation step for skill updates to prevent data loss


### [2026-04-14 19:13:33] Chose Semantic Versioning for project release tracking


### [2026-04-14 19:13:33] Consoildlated architecture by eliminating duplicate packages


### [2026-04-14 19:13:33] Reserve 64 chars for help banner truncation to keep UI consistent










### [2026-04-14 20:59:17] Structured JSON output format for machine-readable retrieval








### [2026-04-14 22:53:12] Tiered view display strategy for organized information presentation




### [2026-04-15 05:54:00] Merge legacy commands (search, eval, prune) into unified commands (get, clean, doctor).


### [2026-04-15 05:54:00] Automated version injection via ldflags using git describe tags for consistency.


### [2026-04-15 05:54:00] Consolidate duplicate packages into unified knowledge/ and session/ modules.


### [2026-04-15 05:54:00] Safe overwrite confirmation for skill updates to prevent accidental data loss.


### [2026-04-15 05:54:00] TTY-aware color output in doctor/status commands for improved UX.


### [2026-04-15 05:54:00] Tiered view display strategy for organized information presentation.


### [2026-04-15 05:54:00] Auto-detect missing sessions by comparing git HEAD~1 diff content.


### [2026-04-15 05:54:00] Isolation of project-specific daemon service names using hashes.


### [2026-04-15 05:54:00] Behavior signal tracking using JSON for machine-readable usage patterns.

