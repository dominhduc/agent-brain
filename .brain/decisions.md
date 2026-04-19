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




### [2026-04-09 14:56:21] Renamed tool identifier from 'agent-brain' to 'brain' for shorter display.




### [2026-04-09 14:56:21] Updated version string in code and banners during release management.




### [2026-04-09 14:56:21] CLI command name shortened from agent-brain to brain.






### [2026-04-09 14:56:21] Add dedup command as maintenance tool similar to prune




### [2026-04-09 14:56:21] Sorting by topic then line number provides deterministic duplicate removal




### [2026-04-09 14:56:21] Keep entry from alphabetically first topic in cross-topic scenarios




### [2026-04-09 14:56:21] Archive files with timestamp for audit trail




### [2026-04-09 14:56:21] Exit codes are strict (1 on error, 0 on success)




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




























### [2026-04-15 05:54:00] Merge legacy commands (search, eval, prune) into unified commands (get, clean, doctor).




### [2026-04-15 05:54:00] TTY-aware color output in doctor/status commands for improved UX.






### [2026-04-15 05:54:00] Auto-detect missing sessions by comparing git HEAD~1 diff content.




### [2026-04-15 05:54:00] Isolation of project-specific daemon service names using hashes.




### [2026-04-15 05:54:00] Behavior signal tracking using JSON for machine-readable usage patterns.




### [2026-04-15 06:08:16] TTY-aware color output provides improved user experience in status commands.








### [2026-04-15 13:29:10] Separated brief/Full-help usage to improve user experience




### [2026-04-15 13:29:10] Use relative time format for recent timestamps with fallback to full format for future dates




### [2026-04-15 13:29:10] Custom version output including OS/arch and short commit hash for diagnostic clarity




### [2026-04-15 13:29:10] Daemon auto-restart to restore functionality after updates without manual intervention




### [2026-04-15 13:29:10] Replace 'agent-brain' with 'brain' for CLI name consistency




### [2026-04-15 13:29:10] Track behavior signals in JSON for data analysis and ML pattern recognition




### [2026-04-15 13:29:10] Command help formatting was adjusted to standardize indentation.




### [2026-04-15 13:29:10] Decided to introduce single-letter flag aliases for common command parameters




### [2026-04-15 16:24:55] AddEntry now returns (bool, error) — true if duplicate detected, prints 'Already exists — skipped'



### [2026-04-15 16:24:55] Working memory was write-only, now readable via 'brain get wm' or 'brain get working-memory'






### [2026-04-15 16:24:55] Version variable as var instead of const so goreleaser ldflags can override at build time



### [2026-04-15 16:24:55] Auto-accept flag for CI/automated workflows



### [2026-04-15 16:24:55] Dual-source config resolver with project precedence



### [2026-04-15 16:24:55] Per-project daemon service names via hash isolation






### [2026-04-15 16:24:55] Exit codes strict 0 success 1 error



### [2026-04-15 16:24:55] Archived storage instead of deletion for audit trail



### [2026-04-15 16:24:55] Human-in-the-loop approval before permanent storage



### [2026-04-15 16:24:55] Version injected via ldflags from git describe instead of const declaration



### [2026-04-15 16:24:55] All daemon services isolated to specific project stacks using hash-based service names






### [2026-04-15 16:24:55] Unified command routing via main.go consolidates legacy commands into unified modules



### [2026-04-15 16:24:55] Two-tier config system separates global and project-specific settings for flexibility.



### [2026-04-15 16:24:55] Versioning implemented via static string variable accessible across CLI modules.



### [2026-04-15 16:24:55] Dual-config system allows fallback from project-local to global settings.



### [2026-04-15 16:24:55] Extracted LLM analysis into internal analyer package for testability without real API



### [2026-04-15 16:24:55] Named custom providers replaced generic custom for better configuration clarity



### [2026-04-15 16:24:55] Unified knowledge.Hub consolidated all state operations to reduce dependencies



### [2026-04-15 16:24:55] Self-update downloads platform-specific archives and replaces with .bak backup



### [2026-04-15 16:24:55] Tiered view display separates entry metadata from actual content storage



### [2026-04-15 16:24:55] JSON format chosen for behavior signals to enable machine-readable analysis



### [2026-04-15 16:24:55] Project-specific daemon services use hash-isolated names for parallelization



### [2026-04-15 16:24:55] Dual-source config: project-local has priority over global fallback









### [2026-04-15 16:24:55] Self-learning core uses behavior signals + index strength for skill adaptations



### [2026-04-15 16:24:55] Version string defined as var to allow goreleaser ldflags override at build time.



### [2026-04-15 16:24:55] CLI alias documentation added to all commands for backward compatibility reference



### [2026-04-15 16:24:55] Daemon uses per-project service names generated via unique hash to enable parallel processing



### [2026-04-15 16:24:55] Fuzzy deduplication integrated alongside exact dedup with threshold of 0.55



### [2026-04-15 16:24:55] Version injection uses git describe with ldflags for build-time stamping



### [2026-04-15 16:24:55] Terminal detection standardized through golang.org/x/term across platforms



### [2026-04-15 16:24:55] Testing configuration isolation via XDG_CONFIG_HOME to prevent production config pollution



### [2026-04-15 16:24:55] Project rename from agent-brain to brain for CLI while keeping trace tokens



### [2026-04-15 16:24:55] Tered review default for non-TTY sessions to avoid fatal silent closure



### [2026-04-15 16:24:55] Multiple LLM providers supported via unified config section for flexibility



### [2026-04-15 16:24:55] Self-update uses GitHub Releases API with platform-specific archive extraction



### [2026-04-15 16:24:55] Version injection uses git describe --tags with -X ldflags for automated tracking



### [2026-04-15 16:24:55] Version string moved from const to var to enable build-time ldflags override



### [2026-04-15 16:24:55] Human-in-the-loop review validates LLM analysis before permanent storage



### [2026-04-15 16:24:55] Separate daemon namespace per project using unique hashes allows parallel processing



### [2026-04-15 16:24:55] Self-update uses GitHub Releases API and supports zip/tar.gz binary extraction



### [2026-04-15 16:24:55] Machine-readable JSON output formatted for consistent CLI parsing



### [2026-04-19 13:14:07] Human-in-the-loop review system supports multiple TTY/autocut modes


### [2026-04-19 13:14:07] Auto-restart daemon after update with error handling


### [2026-04-19 13:14:07] CLI tools maintain backward compatibility for legacy commands


### [2026-04-19 13:14:07] Dual-source config resolver checks project first, falls back to global


### [2026-04-19 13:14:07] Trigram-Jaccard similarity threshold 0.55 for near-duplicate detection


### [2026-04-19 13:14:07] ROI-based extraction proven more accurate than keyword matching


### [2026-04-19 13:14:07] Embedded vector store for semantic search using chromem-go


### [2026-04-19 13:14:07] Text normalization before trigram extraction ensures case-insensitive matching


### [2026-04-19 13:14:07] Session tracking with handoffs via git commits


### [2026-04-19 13:14:07] Modular CLI structure with per-command handler files


### [2026-04-19 13:14:07] Behavioral signals persisted in JSON for ML pattern recognition


### [2026-04-19 13:14:07] Knowledge hub prioritizes file-system based storage for modularity


### [2026-04-19 13:14:07] Self-update uses GitHub Releases API with checksum verification


### [2026-04-19 13:14:07] Secret scanning centralized in internal/secrets package


### [2026-04-19 13:14:07] CLI aliases deprecated but maintained for backward compatibility


### [2026-04-19 13:14:07] API key configuration moved to header format to avoid 'key' in logs

