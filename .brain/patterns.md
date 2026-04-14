# Patterns
<!-- Entries added by brain add pattern or daemon analysis -->

### [2026-04-03 08:42:34] Error messages follow 'What to do:' guidance format — always tell user how to resolve the error, not just what went wrong


### [2026-04-03 08:42:34] TDD for internal packages — write test file first in internal/<pkg>/<pkg>_test.go, verify it fails, then implement


### [2026-04-03 08:42:34] Cross-platform files use //go:build tags — lock_posix.go (!windows), lock_windows.go (windows), signal_posix.go, signal_windows.go


### [2026-04-03 08:42:34] Package extraction pattern: create internal/<pkg>/<pkg>.go + _test.go, add import to main.go, replace inline code with package calls, delete old code from main.go


### [2026-04-03 08:42:34] Service registration uses runtime.GOOS switch — platform-specific implementations in service_posix.go and service_windows.go


### [2026-04-03 09:06:05] Public repo readiness checklist: (1) Remove dev-only files plans from tracking,2) gitignore dev notes, (3) verify no real secrets/paths/email, (4) verify no hardcoded IP/internal URLs, (5) run secret scanner on all tracked files


### [2026-04-04 06:43:44] After releasing new CLI features: 1) Bump version in CHANGELOG.md 2) Update printUsage() in cmd.go 3) Don't forget either - both need updating together


### [2026-04-04 06:50:20] After code push with new version: 1) Create GitHub release with git tag push 2) Wait for goreleaser workflow to complete 3) Verify release appears in gh release list 4) Then run brain update to test


### [2026-04-04 07:00:13] CLI help consistency: When adding config keys, update ALL of: 1) registry.go allKeys slice order 2) printUsage() config section 3) cmdConfigShow() output order 4) cmdConfigList() uses registry order automatically - keep them in sync


### [2026-04-04 07:02:07] After any code push that changes CLI behavior or help text: ALWAYS tag a new version (minor for UX fixes, patch for bug fixes) and push tag. Then verify with 'gh release list' and run 'brain update' to test. Never skip this step.


### [2026-04-04 11:28:09] CLI setup wizard model validation: For provider selection, allow both number input (1,2,3) OR direct model name input. Validate model format for each provider and warn if it doesn't match typical patterns, but let user override with confirmation.


### [2026-04-04 15:27:33] Global vs local installation: brain binary is fully self-contained (CGO_ENABLED=0 static binary), uses os.Getwd() to find project context, walks up from cwd to find .brain/. Can be installed anywhere in PATH, global install works perfectly.


### [2026-04-04 16:19:17] Named custom providers: option 6 in setup wizard now asks for a name (e.g. 'groq'), base-url, api-key, model. Stored in custom_providers map in config.yaml. Provider field stores the name, not 'custom'. Analyzer resolves unknown provider names from custom_providers config.


### [2026-04-04 17:21:11] Per-project daemon isolation: systemd service files (brain-daemon.<hash>.service) are per-project with WorkingDirectory set. PID lock file must also be per-project (brain-daemon-<hash>.pid) to allow parallel daemons. Data is inherently per-project (each project's .brain/.queue/ directory).


### [2026-04-04 19:08:24] Daemon failure inspection: 'brain daemon failed' lists each failed item with error reason, attempts count, and changed files. ErrorReason field is stored in the queue item JSON when moved to failed/.


### [2026-04-04 19:48:51] Daemon management commands: start, stop, restart, status, failed, retry. restart = stop + start (useful after config changes). retry = move all failed/ items back to .queue/, reset attempts to 0 and clear error_reason. Failed items retain their original queue JSON data.


### [2026-04-04 20:13:19] brain status is the single command for full project state: hub stats, config (provider/model/key/profile), daemon (running/queue), and health warnings. The Health section only appears when there are issues. Subcommands (brain daemon status, brain config, brain doctor) still work for detailed views.


### [2026-04-04 21:08:55] brain prune workflow: create .brainprune at project root with one pattern per line. Lines in topic files matching patterns get archived to .brain/archived/. Run 'brain prune --dry-run' first to preview. Use for outdated entries: old versions, resolved issues, deprecated patterns.


### [2026-04-04 22:05:35] brain doctor checks: version, git, .brain/, config (with API key), daemon (running state), preflight. All health checks should be in one command for quick diagnosis.


### [2026-04-04 22:05:35] Config key names: use 'api-key' (friendly) in user-facing messages, not 'llm.api_key' (dot-path). The ResolveKey function accepts both, but user messages should be consistent.


### [2026-04-04 22:16:55] Analyzer LLM prompts: keep simple. System prompt rules, user prompt just says 'analyze this diff'. Output format in system prompt only. Avoid markdown examples in user prompt - models try to format response as code blocks.


### [2026-04-05 07:12:27] Handling multiple body types in HTTP requests


### [2026-04-05 07:12:27] Type switch used to handle different body types


### [2026-04-05 07:57:51] HTTP client uses a type switch to handle various body types.


### [2026-04-05 07:57:51] Tests set XDG_CONFIG_HOME for configuration isolation.




### [2026-04-05 07:57:51] XDG_CONFIG_HOME setperfor test config directory isolation




### [2026-04-05 08:34:34] LLM prompts utilize explicit JSON schema for better structured output compliance.


### [2026-04-05 08:34:34] Modular CLI structure splits monolithic main.go into per-command files.


### [2026-04-05 08:34:34] Queue-based daemon processing includes retry logic and error reason tracking.


### [2026-04-05 08:34:34] Version string is defined as a package variable in cmd.go.


### [2026-04-05 08:34:34] --yes flag is used to force explicit non-interactive auto-accept.


### [2026-04-05 08:34:34] Semantic Versioning requirements are documented in the header.












### [2026-04-05 10:39:47] Platform-specific CanUseRawMode checks for terminal capabilities


### [2026-04-05 10:39:47] XDG_CONFIG_HOME used to isolate test config writes from production


### [2026-04-06 08:36:28] SUPER principles for agent-friendly code: 5 constraints — S: Side Effects at Edge (I/O in thin outer layer, never in business logic), U: Uncoupled Logic (dependencies as parameters, not globals), P: Pure & Total Functions (deterministic, handle every input, return Result not throw), E: Explicit Data Flow (linear pipeline, each step returns new value), R: Replaceable by Value (referential transparency — can swap function call with its result). Agents working on SUPER-compliant code pass tests 3x more often on first try


### [2026-04-06 08:36:30] SPIRALS process loop for human-agent collaboration: 7-step cycle — S: Sense (gather context), P: Plan (draft approach, define done), I: Inquire (identify knowledge gaps), R: Refine (simplify, 80/20, split >3pt tickets), A: Act (write code following SUPER), L: Learn (run tests, record failures), S: Scan (zoom out, check regressions, prevent infinite loops). Split into SPIR|ALS with human approval gate between planning and execution phases


### [2026-04-06 08:36:33] Agent code must be machine-reasonable: mutable state, hidden dependencies, and entangled side effects make agent output non-deterministic and impossible to debug. Code should follow: pure functions (same input → same output, no globals/DB/logging inside), explicit data flow (trace linearly), side effects at boundaries (I/O in thin outer layer), composition over coupling (small replaceable functions). When agent modifies function, scope of breakage must be exactly one function


### [2026-04-06 10:30:43] brain get now supports --summary flag for compact view with entry counts and duplicate warnings


### [2026-04-06 10:30:47] brain add now automatically deduplicates entries - checks normalized message content before adding, skips if duplicate exists








### [2026-04-06 11:46:28] Platform-specific terminal capability checks prevent issues with raw mode usage.


### [2026-04-06 11:46:28] path-based allowlist for specific test files


### [2026-04-06 11:46:28] SUPER principles: S=Side Effects at Edge, U=Uncoupled Logic, P=Pure & Total Functions, E=Explicit Data Flow, R=Replaceable by Value


### [2026-04-06 11:46:28] SPIRALS loop: Sense, Plan, Inquire, Refine, Act, Learn, Scan for agent collaboration


### [2026-04-06 11:46:28] Entry deduplication via normalizeEntry() with strings.ToLower() and whitespace collapsing


### [2026-04-06 11:46:28] Machine-reasonable code: pure functions with explicit data flow and side effects at boundaries


### [2026-04-06 11:46:28] Agent functions should be pure with side effects isolated to a thin outer layer


### [2026-04-06 11:46:28] TDD is required for internal packages to maintain code integrity


### [2026-04-06 11:46:28] Add command now deduplicates entries based on normalized message content




### [2026-04-06 11:46:28] New CLI flags support JSON and compact summary output.


### [2026-04-06 11:46:28] Content deduplication relies on normalized string comparison.


### [2026-04-06 12:07:42] brain review now handles EOF gracefully - stdin closed during review exits cleanly instead of erroring


















### [2026-04-06 13:43:38] Content deduplication via map[string]bool tracking before committing to set


### [2026-04-06 13:43:38] Use golang.org/x/term.IsTerminal() for reliable TTY detection instead of manual syscall-based checks


### [2026-04-06 13:43:38] State variable tracks loop phase to differentiate entry states.


### [2026-04-06 13:43:38] Default fallback action implemented to auto-accept entries when interaction fails.


### [2026-04-06 15:41:55] Test comments removed from binary source code.


### [2026-04-06 15:41:55] Decisions documented with timestamps in a separate log.


### [2026-04-06 15:41:55] Use standard library packages like golang.org/x/term instead of writing custom syscall wrappers.


### [2026-04-06 15:41:55] Version bump in code matches entry in CHANGELOG.md.










### [2026-04-06 15:47:20] Simplified conditional logic for TUI vs line-buffered review selection














### [2026-04-06 15:48:51] Adopts external packages like golang.org/x/term over custom syscalls.


### [2026-04-06 15:48:51] Auto-accept pending entries to avoid fatal silent closure treatment.
















### [2026-04-06 16:02:42] UI failure triggers fallback to line-buffered mode


### [2026-04-09 14:56:21] Context propagation and cancellation throughout daemon processing pipeline


### [2026-04-09 14:56:21] Structured attributes with semantic conventions for vendor-agnostic telemetry


### [2026-04-09 14:56:21] Separate span per logical unit of work (diff, analyze, review)


### [2026-04-09 14:56:21] No-op tracer fallback when tracing is disabled for safer configuration


### [2026-04-09 14:56:21] Version number incremented synchronously across command and internal packages


### [2026-04-09 14:56:21] Knowledge-Centered Support implementation captures context during active coding sessions.


### [2026-04-09 14:56:21] Human-in-the-loop approval via TUI validates automated analysis before storage.


### [2026-04-09 14:56:21] Spaced repetition logic mimics biological memory consolidation with sleep commands.


### [2026-04-09 14:56:21] Reorganized help text into structured sections for better navigation.


### [2026-04-09 14:56:21] Documented legacy commands within help text to aid user migration.


### [2026-04-09 14:56:21] Version bumping


### [2026-04-09 14:56:21] Banner padding calculated dynamically based on title string length.


### [2026-04-09 14:56:21] Project name abbreviated consistently across help text and version outputs.


### [2026-04-09 14:56:21] Uses SHA256 hash for content fingerprinting with normalization for case-insensitive matching


### [2026-04-09 14:56:21] Drives if filename not already ending with .md


### [2026-04-09 14:56:21] Architecture uses alphabetical first topic for cross-topic duplicate resolution


### [2026-04-09 14:56:21] Testing with same-topic and cross-topic duplicates covers different scenarios


### [2026-04-09 14:56:21] Dry-run mode implemented before actual modification for safety


### [2026-04-09 14:56:21] Archives removed entries rather than deleting completely


### [2026-04-09 14:56:21] Version number incremented following semantic versioning rules


### [2026-04-09 14:56:21] Normalize content to calculate SHA-256 fingerprints ensuring cross-topic duplicate detection.


### [2026-04-09 14:56:21] Entry archival follows a specific directory structure with date-based filenames.


























### [2026-04-12 06:20:41] Limited external dependencies to Go standard library (golang.org/x/term)








### [2026-04-12 06:20:41] Use standard library packages like golang.org/x/term for reliable TTY detection


### [2026-04-12 06:20:41] Config resolves to project-local if exists, falling back to global


### [2026-04-12 06:20:41] API key can be set via environment variable as fallback




### [2026-04-14 01:32:14] Self-learning uses behavior signals (.brain/behavior/signals.json) + index strength data to generate skill adaptations via brain skill reflect


















### [2026-04-14 06:46:30] Duplicate detection normalizes content before committing to a set.


### [2026-04-14 06:46:30] Terminal introspection favors standard library packages over native syscalls.














### [2026-04-14 06:46:30] Layered config system with global and project-specific sources




### [2026-04-14 06:46:30] Lazily batch config resolution with separate read/write paths


















### [2026-04-14 06:46:30] Two-tier config system with project-first precedence over global config


### [2026-04-14 06:46:30] Dry-run flag for previewing changes before writing


### [2026-04-14 06:46:30] Marker-based section insertion to preserve existing adaptations


### [2026-04-14 06:46:30] Behavior signals tracking for personalized adaptation generation


### [2026-04-14 06:46:30] Command-based behavioral feedback loops with eval outcomes


### [2026-04-14 06:46:30] Topic access metrics for identifying knowledge gaps




### [2026-04-14 06:46:30] Unified knowledge Hub consolidates state operations to eliminate deep dependencies






### [2026-04-14 06:46:30] Unified knowledge.Hub acts as the single core for all state operations.


### [2026-04-14 13:05:03] v1.1.0 consolidated 8 duplicate packages into unified knowledge/ and session/ packages










### [2026-04-14 19:13:33] Use Makefile ldflags with git describe for automated version injection


### [2026-04-14 19:13:33] Store version at build-time via compiler flags, not compile-time


### [2026-04-14 19:13:33] Show all modified files in confirmation prompt before overwriting








### [2026-04-14 20:59:17] Safe overwrite confirmation for skill updates


### [2026-04-14 20:59:17] TTY-aware output with color indicators


### [2026-04-14 20:59:17] Tiered view display strategy


### [2026-04-14 20:59:17] Modular CLI structure with per-command files


### [2026-04-14 22:53:12] Trigram Jaccard similarity for near-duplicate detection (threshold 0.55)


### [2026-04-14 22:53:12] Union-Find clustering for grouping similar entries


### [2026-04-14 22:53:12] Character normalization before trigram extraction


### [2026-04-14 22:53:12] Dry-run mode for safe duplicate removal operations


### [2026-04-14 22:53:12] Global threshold fallback when provided invalid values


### [2026-04-15 05:54:00] Consolidate CLI commands into unified modules (borrowing existing logic like cmdClean covering old prune/dedup).


### [2026-04-15 05:54:00] Queue-based daemon processing with backoff retry logic for resilience.


### [2026-04-15 05:54:00] Use standard library golang.org/x/term for TTY detection across platforms.


### [2026-04-15 05:54:00] Auto-accept pending entries in non-TTY environments to prevent fatal exits.


### [2026-04-15 05:54:00] Fuzzy deduplication using trigram Jaccard similarity with thresholding (0.55).


### [2026-04-15 05:54:00] Union-Find clustering groups similar entries for cross-topic deduplication.


### [2026-04-15 05:54:00] Semantic Versioning for consistent release tracking and changelog generation.


### [2026-04-15 05:54:00] Modular CLI structure with per-command files to improve maintenance.


### [2026-04-15 05:54:00] Session evaluation handoffs with topic detection from git diff content.

