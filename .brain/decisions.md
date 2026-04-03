# Decisions
<!-- Entries added by brain add decision or daemon analysis -->

### [2026-04-03 08:42:47] Extracted LLM analysis into internal/analyzer for testability — Analyze() takes AnalyzeRequest struct with configurable APIBaseURL, enabling httptest-based tests without real API calls


### [2026-04-03 08:42:47] Dependency injection for daemon.ProcessItemWithDeps — takes DiffGetter and AnalyzeFunc as function params, allowing tests to mock git diff and LLM calls


### [2026-04-03 08:42:47] Daemon PID lock uses flock on POSIX (auto-released on crash) and file-existence on Windows (best-effort, stale lock possible on crash)


### [2026-04-03 08:42:47] Self-update uses GitHub Releases API — Downloads platform-specific archive, extracts binary, replaces in-place with .bak backup. Supports both tar.gz and zip (Windows).


### [2026-04-03 08:42:47] Changed version from const to var so goreleaser ldflags can override it at build time — -X main.version={{.Version}}

