# Architecture
<!-- Entries added by brain add architecture or daemon analysis -->



### [2026-04-03 08:43:06] internal/analyzer — LLM analysis: Analyze(AnalyzeRequest) → Finding, WriteFindings(Finding, brainDir). Prompt construction, OpenRouter API call via httpclient.PostJSON, JSON response parsing.




### [2026-04-03 08:43:06] internal/daemon — Queue processing: QueueItem struct, ProcessItemWithDeps() with DI (DiffGetter, AnalyzeFunc). CalcBackoff(attempt), ParsePollInterval(input). RecoverStaleProcessing(brainDir).




### [2026-04-03 08:43:06] internal/service — Service registration: Register(execPath), Start(), Stop(), IsRunning(). Platform-specific: launchd (macOS), systemd (Linux). Windows stubs.




### [2026-04-03 08:43:06] internal/updater — Self-update: FetchLatestRelease(FetchOptions), FindAssetForPlatform(release, goos, goarch), IsNewerVersion(current, latest), DownloadAndReplace(url, binPath). Supports tar.gz and zip extraction. DownloadAsset() for private repo asset API.




### [2026-04-03 08:43:06] cmd/brain/main.go — CLI router with thin switch statement. Uses internal/* packages for daemon start/stop, add, version, update, Platform files: lock_posix.go, lock_windows.go, signal_posix.go. signal_windows.go




### [2026-04-03 08:43:06] Cross-platform strategy: Use //go:build tags for platform-specific files. runtime.GOOS for shared code for service routing. filepath.Join() everywhere (no string concatenation with /). os.UserConfigDir() for config paths.




### [2026-04-04 16:19:23] Config.CustomProviders map[string]CustomProviderConfig stores named custom provider definitions (base_url, api_key, model). provider.NewCustom() returns a Custom provider without registry lookup. Analyzer falls back to NewCustom() when provider name is unknown but BaseURL is provided or config has a matching custom provider.




### [2026-04-05 07:12:27] Response parsing with conditional field checks




### [2026-04-05 07:12:27] Flexible HTTP client design accommodating various data formats










### [2026-04-05 07:57:51] Config.Load() returns defaults with warnings instead of errors




### [2026-04-05 08:34:34] Human-in-the-loop review system supports distinct autonomy profiles for automated approvals.




### [2026-04-05 08:34:34] Multi-project daemon support isolates service names per project hash.




### [2026-04-05 08:34:34] Cross-platform file handling uses filepath.Join instead of OS string concatenation.




### [2026-04-05 08:34:34] Environment isolation ensures tests do not pollute production configuration files.




### [2026-04-05 08:34:34] Daemon processes queue items with backoff retry logic.








### [2026-04-05 08:34:34] Human-in-the-loop review system integrates directly with git hooks.


























### [2026-04-06 11:46:28] Environment isolation uses XDG_CONFIG_HOME to prevent test pollution.






### [2026-04-06 11:46:28] Gitleaks integration with path-based exclusions for test fixtures




### [2026-04-06 11:46:28] Stratified knowledge hub with separate topic files under .brain/




### [2026-04-06 11:46:28] Centralized GetTopicSummary() generates TopicSummary structs per file






### [2026-04-06 11:46:28] summaryFlag conditional redirect to brain.GetAllSummaries() before main logic




### [2026-04-06 11:46:28] Test coverage includes cross-file entry types (gotchas, patterns, memory) for deduplication








### [2026-04-06 11:46:28] CLI command logic isolated in cmd/brain/cmd.go.






















### [2026-04-06 13:43:38] Simplified term_posix.go by replacing manual syscall with standard library term.IsTerminal() for cross-platform compatibility




### [2026-04-06 13:43:38] TUI input handling adds specific logic to differentiate between initial and mid-loop stdin closure.




### [2026-04-06 15:41:55] Terminal state is managed within the internal/tui module.




### [2026-04-06 15:41:55] Terminals are identified via stdin file descriptors using wrapper functions.








### [2026-04-06 15:47:20] cmd_review.go: Auto-accept for non-TTY when !yesFlag and !prof.AutoAccept












### [2026-04-06 15:47:20] TUI logic differentiates initial vs mid-loop stdin closure




### [2026-04-06 15:47:20] Review command handles TTY/auto-accept affinity logic










### [2026-04-06 16:02:42] Pending entries counting unified across cmd_status.go and cmd_review.go using review.LoadPendingEntries()








### [2026-04-06 16:02:42] Review command supports multiple UI modes within same structure




### [2026-04-09 14:56:21] Components now expose context.Context for span propagation




### [2026-04-09 14:56:21] Daemon queue processing traces from start to LLM analysis




### [2026-04-09 14:56:21] Review session traced through TUI, line-buffered, and auto modes




### [2026-04-09 14:56:21] OTel configuration loaded from user brain config.yaml directory




### [2026-04-09 14:56:21] Structured directory hierarchy organizes Git hooks, knowledge files, and pending queues.




### [2026-04-09 14:56:21] Pipeline integrates CLI commands, background daemon, and LLM providers for ingestion.








### [2026-04-09 14:56:21] CLI tool structure with cmd_*.go command handlers




### [2026-04-09 14:56:21] internal package abstraction for core logic




### [2026-04-09 14:56:21] Multiple md topic files maintained (architecture, decisions, gotchas, memory, patterns)




### [2026-04-09 14:56:21] Cross-topic deduplication by scanning all topics together




### [2026-04-09 14:56:21] Archived storage in brainDir/archived/ with date suffix




### [2026-04-09 14:56:21] Application entry point maintains version string globally




### [2026-04-09 14:56:21] The tool uses a file-system based memory architecture with command-line interfaces.




### [2026-04-09 14:56:21] A separate storage location handles archived entries specifically for deduplication logs.


























### [2026-04-12 06:20:41] Multi-project daemon support with project isolation




### [2026-04-12 06:20:41] Config scope choice: global shared or project-specific isolated in .brain/config.yaml




### [2026-04-12 06:20:41] Static TTY state checked at startup and maintained via stdin file descriptors






### [2026-04-12 06:20:41] Unified entry statistics tracked via review.LoadPendingEntries()














### [2026-04-12 06:20:41] Config dual-source resolver checks project first, falls back to global












### [2026-04-12 06:20:41] Embedded skill templates managed via fs/embed for version control




### [2026-04-12 06:20:41] Skill system supports platform-specific directory locations (.opencode/.claude/.agents)






### [2026-04-14 01:32:14] knowledge.Hub is the unified core — all state operations go through it. New packages: knowledge/ (topics+index+pending+wm+retrieval+behavior+adapt), session/ (handoffs+git stats). Daemon now has local analyze.go + secrets.go instead of importing external packages.






### [2026-04-14 06:46:30] Two-tier config system with dual-source resolver








### [2026-04-14 06:46:30] Agent skill files installed to .claude/, .opencode/, .agents/ directories


















































### [2026-04-14 06:46:30] Git-integrated knowledge persistence through session captures










### [2026-04-14 06:46:30] New internal/daemon package replaces internal/analyzer for code analysis functionality




### [2026-04-14 06:46:30] Behavioral learning layer using user-generated signals




### [2026-04-14 06:46:30] Knowledge hub generates adaptations from retrieval patterns




### [2026-04-14 06:46:30] Multi-directory synchronization for skill files




### [2026-04-14 06:46:30] Command-driven reflection workflow for project integration






### [2026-04-14 06:46:30] Behavior signals tracked in .brain/behavior/signals.json for self-learning




### [2026-04-14 06:46:30] Core logic consolidated into integrated internal/knowledge package structure.




### [2026-04-14 06:46:30] Directory layout expanded to include behavior/ signals for self-learning tracking.






### [2026-04-14 19:13:33] Unified knowledge.Hub is the core all state operations go through it centrally.










### [2026-04-14 19:13:33] Makefile builds version from source control timestamps






### [2026-04-14 20:59:17] Two-tier config system architecture




### [2026-04-14 20:59:17] Tiered view display architecture




### [2026-04-14 20:59:17] Isolated daemon service naming convention




### [2026-04-14 20:59:17] Automated version injection via build flags
















### [2026-04-15 05:54:00] .brain/ knowledge hub with separate topic files (gotchas, patterns, memory) for modularity.




### [2026-04-15 05:54:00] Stratified knowledge base with MEMORY.md and PATTERNS.md as core knowledge vectors.




### [2026-04-15 05:54:00] Background daemon service with isolated queue processing per project.






### [2026-04-15 05:54:00] CLI command routing via main.go with backward compatibility aliases for legacy commands.






### [2026-04-15 05:54:00] Behavior-driven skill adaptations stored in .brain/behavior/signals.json.
















### [2026-04-15 13:29:10] Session tracking and self-evaluation with handoff to next agent




### [2026-04-15 13:29:10] Knowledge entries grouped by topic with trigram-based duplicate detection








### [2026-04-15 13:29:10] Behavioral metrics are persisted in a separate JSON configuration file.




### [2026-04-15 16:24:55] Knowledge hub acts as single core for all state operations, eliminating deep dependencies



### [2026-04-15 16:24:55] Modular CLI structure with per-command files consolidaing logic (get, add, doctor, clean)



### [2026-04-15 16:24:55] Terminal state managed within internal/tui module using stdin file descriptors






### [2026-04-15 16:24:55] Knowledge hub as unified core state manager



### [2026-04-15 16:24:55] Multiple topic files (gotchas, patterns, memory) stratified






### [2026-04-15 16:24:55] Session tracking and handoffs via git commits



### [2026-04-15 16:24:55] OpenTelemetry tracing throughout pipeline



### [2026-04-15 16:24:55] CLI command handlers modularized as cmd_*.go



### [2026-04-15 16:24:55] Daemon types local to internal/daemon package



### [2026-04-15 16:24:55] Behavioral signal tracking in JSON for ML adaptation



### [2026-04-15 16:24:55] File-system based memory architecture with separate topic files under .brain/






### [2026-04-15 16:24:55] OTel configuration for CLI commands and daemon pipeline tracing



### [2026-04-15 16:24:55] Unified CLI routing consolidates legacy commands into single entry points



### [2026-04-15 16:24:55] Centralized behavior signal tracking in JSON for ML pattern recognition



### [2026-04-15 16:24:55] Modular Go packages with internal abstractions for domain logic separation









### [2026-04-15 16:24:55] Cross-platform strategy: runtime.GOOS checks and platform-specific files (lock_posix.go, signal_posix.go)



### [2026-04-15 16:24:55] OpenTelemetry tracing with stdout exporter for span visibility and observability



### [2026-04-15 16:24:55] Behavioral metrics tracked separately in .brain/behavior/signals.json



### [2026-04-15 16:24:55] Self-learning layer generates adaptations from retrieval patterns and behavioral signals






### [2026-04-15 16:24:55] Two-tier display architecture with tiered view display strategy for organized retrieval



### [2026-04-15 16:24:55] Comprehensive testing includes cross-file entry types for deduplication coverage



### [2026-04-15 16:24:55] Daemon worker retry logic with exponential backoff (calcBackoff, attempt counting)



### [2026-04-15 16:24:55] Binary name 'brain' with backward compliance aliases for legacy commands






### [2026-04-15 16:24:55] Modular CLI structure isolates command handling in cmd_*.go files routed via main.go.



### [2026-04-15 16:24:55] Project isolation uses service name hashing and unique lock files for parallel daemons.



### [2026-04-15 16:24:55] Signal handling uses platform-specific wrappers using //go:build tags.



### [2026-04-15 16:24:55] Modular CLI routed through main.go with backward compatibility aliases consolidated



### [2026-04-15 16:24:55] Stratified knowledge base preserves separate topic files for memory, patterns, and gotchas






### [2026-04-15 16:24:55] Per-project daemon queue processing isolates service naming by project hash



### [2026-04-15 16:24:55] Session tracking with handoff mechanism captures context during active coding activities



### [2026-04-15 16:24:55] Background analyzer processes LLM analysis results and writes findings to designated storage






### [2026-04-15 16:24:55] Behavioral metrics persisted in separate JSON file for machine-readable usage patterns



### [2026-04-15 16:24:55] Stratified knowledge hub organized with separate topic files for modular retrieval



### [2026-04-15 16:24:55] Dual-source config system resolves project-local sources before global fallbacks



### [2026-04-19 13:14:07] Layer-based label scanning with topic hierarchy


### [2026-04-19 13:14:07] Buffer-based config resolution with project-first precedence


### [2026-04-19 13:14:07] Memory hub architecture with topic files for modularity


### [2026-04-19 13:14:07] Budget-aware retrieval with context support (git diff context)


### [2026-04-19 13:14:07] Split plane of evolution: planning and execution phases


### [2026-04-19 13:14:07] Layer hierarchies preserve entries with source timelines


### [2026-04-19 13:14:07] Multi-project daemon isolation via hash-based service names


### [2026-04-19 13:14:07] Two-tier architecture: project-shared and global isolation


### [2026-04-19 13:14:07] Per-project daemon queue processing with exponential backoff retry


### [2026-04-19 13:14:07] Daemon adopted local types to simplify dependency chains


### [2026-04-19 13:14:07] Session lifecycle separated into internal/session/ package


### [2026-04-19 13:14:07] Two-tier display architecture separates metadata from content


### [2026-04-19 13:14:07] Filesystem knowledge base with separate topic files for modularity


### [2026-04-19 13:14:07] Cross-platform strategy using runtime.GOOS checks and wrappers


### [2026-04-19 13:14:07] Isolated daemon service naming via project hash for parallelization


### [2026-04-19 13:14:07] Modular CLI structure with unified routing via main.go


### [2026-04-19 13:14:07] Tiered view display strategy for knowledge retrieval


### [2026-04-19 13:14:07] Agent-friendly code principles (SUPER/SPIRALS) documented

