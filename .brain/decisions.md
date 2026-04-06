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


### [2026-04-05 07:57:51] Support non-JSON and pre-encoded byte slices in HTTP POST requests


### [2026-04-05 07:57:51] Require named custom providers instead of generic 'custom' value


### [2026-04-05 08:34:34] Unified API key configuration key renamed from nested llm.api_key to flat api-key.


### [2026-04-05 08:34:34] Named custom providers replaced single custom option to improve configurability.


### [2026-04-05 08:34:34] Auto-accept flag implemented to replace interactive TUI in CI/automated workflows.


### [2026-04-05 08:34:34] Project-specific daemon service names generated via unique hash lookup.


### [2026-04-05 08:34:34] Config setup wizard flow prioritized provider selection over base URL.


### [2026-04-05 08:34:34] Per-project daemon locks include project hashes for parallelization.


### [2026-04-05 08:34:34] XDG_CONFIG_HOME is used to isolate test config writes from production.


### [2026-04-05 10:39:47] Unified API key configuration key renamed from nested llm.api_key to flat api-key


### [2026-04-05 10:39:47] Auto-accept flag implemented to replace interactive TUI in CI/automated workflows


### [2026-04-05 10:39:47] XDG_CONFIG_HOME is used to isolate test config writes from production configuration


### [2026-04-05 10:39:47] Named custom providers replaced single custom option to improve configurability


### [2026-04-05 10:39:47] Unified API key configuration key renamed to flat api-key structure


### [2026-04-05 10:39:47] Named custom providers replaced single custom option for better configurability


### [2026-04-05 10:39:47] Project-specific daemon service names generated via unique hash lookup


### [2026-04-05 10:39:47] Per-project daemon locks include project hashes for parallelization support


### [2026-04-05 10:39:47] Config setup wizard prioritizes provider selection over base URL setup


### [2026-04-06 11:46:28] Auto-accept flag implemented to replace interactive TUI in CI/automated workflows.


### [2026-04-06 11:46:28] Unified API key configuration renamed from nested structure to flat api-key.


### [2026-04-06 11:46:28] Named custom providers replaced single custom option to improve configurability.


### [2026-04-06 11:46:28] Project-specific daemon service names generated via unique hash lookup.


### [2026-04-06 11:46:28] Added gitleaks allowlist for scanner/config test files


### [2026-04-06 11:46:28] Added --summary flag for compact view with entry counts and duplicate warnings


### [2026-04-06 11:46:28] Duplicate detection adds entries to a set before committing to prevent redundancy


### [2026-04-06 11:46:28] TopicSummary struct separates metadata (counts, duplicates) from actual content


### [2026-04-06 11:46:28] Normalize function uses strings.NewReplacer for whitespace normalization


### [2026-04-06 11:46:28] Split SPL|ALS gate between planning and execution phases with human approval


### [2026-04-06 11:46:28] Architecture decisions are documented in a dedicated decisions.md file for future reference


### [2026-04-06 11:46:28] Tool interface updated to support JSON output for machine-readable command parsing


### [2026-04-06 11:46:28] Added auto-deduplication to prevent duplicate entries on save.


### [2026-04-06 11:46:28] Integrated topic statistics into the main list view.

