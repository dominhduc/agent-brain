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


### [2026-04-05 07:57:51] Response parsing utilizes conditional field checks.


### [2026-04-05 07:57:51] HTTP client design accommodates various data formats.


### [2026-04-05 07:57:51] Flexible HTTP client design accommodating various data formats


### [2026-04-05 07:57:51] Config.Load() returns defaults with warnings instead of errors


### [2026-04-05 08:34:34] Human-in-the-loop review system supports distinct autonomy profiles for automated approvals.


### [2026-04-05 08:34:34] Multi-project daemon support isolates service names per project hash.


### [2026-04-05 08:34:34] Cross-platform file handling uses filepath.Join instead of OS string concatenation.


### [2026-04-05 08:34:34] Environment isolation ensures tests do not pollute production configuration files.


### [2026-04-05 08:34:34] Daemon processes queue items with backoff retry logic.


### [2026-04-05 08:34:34] Multiple LLM providers are supported via unified config section.


### [2026-04-05 08:34:34] Human-in-the-loop review system integrates directly with git hooks.


### [2026-04-05 10:39:47] Human-in-the-loop review system supports distinct autonomy profiles for approvals


### [2026-04-05 10:39:47] Multi-project daemon support isolates service names per project hash


### [2026-04-05 10:39:47] Cross-platform file handling uses filepath.Join instead of OS string concatenation


### [2026-04-05 10:39:47] Multiple LLM providers are supported via unified config section


### [2026-04-05 10:39:47] Human-in-the-loop review system supports distinct autonomy profiles for automated approvals


### [2026-04-05 10:39:47] Daemon processes queue items with backoff retry logic


### [2026-04-05 10:39:47] Multiple LLM providers supported via unified config section


### [2026-04-05 10:39:47] Environment isolation ensures tests do not pollute production configuration files


### [2026-04-05 10:39:47] Human-in-the-loop review integrates directly with git hooks

