# Gotchas
<!-- Entries added by brain add gotcha or daemon analysis -->

### [2026-04-03 08:42:10] Go const cannot be overridden by ldflags — use var instead. The version field must be 'var version = ...' not 'const version = ...' for goreleaser -X ldflags to work.


### [2026-04-03 08:42:10] GitHub private repo releases require GITHUB_TOKEN for both API calls and asset downloads. Use Assets API with Accept: application/octet-stream header for private repo binary downloads. Public repos work fine without token.


### [2026-04-03 08:42:10] goreleaser names binaries as brain_Linux_x86_64 (with underscore prefix), not just 'brain'. Archive extraction must match this naming pattern.


### [2026-04-03 08:42:10] os.UserHomeDir() errors are silently discarded with ',_' pattern — always handle the error, especially in CLI tools where HOME might be unset.


### [2026-04-03 08:42:10] sync.Once cannot be reset — if you need ResetCache(), add a sync.Mutex to serialize both FindBrainDir() and ResetCache(), replacing the Once on reset.


### [2026-04-03 08:42:21] brain update needs GITHUB_TOKEN for private repos — uses GitHub Assets API (accept: application/octet-stream) for private repo downloads, For public repos, BrowserDownloadURL works without token.


### [2026-04-03 08:42:21] syscall.Flock doesn't exist on Windows — use build tags with platform-specific files (//go:build !windows / //go:build windows)


### [2026-04-03 08:42:21] os.UserConfigDir() returns platform-appropriate config paths: Windows=%AppData%, macOS=~/Library/Application Support, Linux=~/.config. Prefer over os.UserHomeDir() for config directories.


### [2026-04-03 08:42:21] SetValue in config must validate numeric fields with strconv.Atoi() and durations with time.ParseDuration() — previously silently discarded errors via fmt.Sscanf


### [2026-04-03 09:05:53] Before making repo public: remove EXECUTION_PLAN.md, REPO_AGENTS.md, docs/plans/ from git tracking. Add to .gitignore.


### [2026-04-04 07:00:15] CLI help inconsistency: Don't just update one help text - check brain help, brain config, brain config list all show consistent commands and order. Use registry.go as source of truth.


### [2026-04-04 15:45:59] Config setup wizard flow: When provider = custom, Step 3 should first ask for base-url, THEN Step 3b ask for model. Other providers skip base-url and go directly to model selection.


### [2026-04-04 16:19:17] Config setup wizard order matters: Provider → Base URL → API Key → Model → Profile. Never ask for API key before knowing WHERE it goes (the endpoint/base URL).


### [2026-04-04 17:21:06] Per-project daemon lock file must include project hash in filename (brain-daemon-<hash>.pid). A global lock file (brain-daemon.pid) prevents multiple project daemons from running simultaneously. The lock path should try brain.FindBrainDir() first, then fall back to os.Getwd().


### [2026-04-04 17:21:06] Queue counting in daemon status and status commands must filter to commit-*.json (matching the daemon loop's processing filter). Without the prefix filter, non-commit JSON files in .queue/ are counted as pending but never processed.


### [2026-04-04 17:21:06] findCurrentProjectBrainDir must use filepath.Join() and filepath.Dir() for cross-platform safety. String concat (dir + '/.brain' and dir + '/..') breaks on Windows and can loop forever without a proper termination check.


### [2026-04-04 17:39:30] Config setup wizard must load existing config (config.Load()), not start from DefaultConfig(). DefaultConfig() has empty API key — starting from it and saving wipes any existing key. Always load first, update selectively.


### [2026-04-04 19:08:24] Failed queue items must store error reason. moveToFailed() must accept a reason string, update the item's ErrorReason field, and strip .processing suffix before moving to failed/.


### [2026-04-04 19:48:56] Config changes require daemon restart to take effect. The daemon reloads config every 10 cycles (line 288: if cycleCount%10 == 0), but model/provider changes only affect the next analyze call. Use 'brain daemon restart' after changing provider, model, or api-key.


### [2026-04-04 22:05:35] Daemon getDiff must handle first commit (HEAD~1 doesn't exist). Use empty-tree fallback: git hash-object -t tree /dev/null to get empty tree hash, then diff empty_tree..HEAD. Without this, first commit in any repo fails processing.


### [2026-04-04 22:16:55] LLM JSON parsing fails with 'JSON parsing failed' even when content looks valid. Root cause: complex prompts with markdown-style formatting confuse models. Fix: simplify prompts - short system prompt, minimal user prompt, explicit 'start with { and end with }' instruction.


### [2026-04-04 22:35:31] Config.Load() must not fail on read/parse errors - return defaults with warning instead of error. If Load() returns an error, any caller that ignores it will use DefaultConfig() and overwrite the file via Save(). Always make Load() fail-safe.


### [2026-04-05 07:12:27] Debug output may be enabled in production


### [2026-04-05 07:57:51] Config.Load returns defaults with warnings to prevent file corruption on parse errors.






### [2026-04-05 08:34:34] PostJSON double-marshaling sends base64-encoded garbage to LLM APIs instead of raw JSON.


### [2026-04-05 08:34:34] Direct TTY operations cause TCGETS errors when not in interactive terminal.


### [2026-04-05 08:34:34] Testing config writes can corrupt live settings without proper isolation.


### [2026-04-05 08:34:34] String-based path concatenation breaks on cross-platform environments.




### [2026-04-05 08:34:34] Non-TTY inputs initially caused TCGETS errors in review commands.


### [2026-04-05 08:34:34] Corrupted config files required specific logic to return defaults gracefully.


### [2026-04-05 10:33:34] Test entry 1














### [2026-04-05 10:39:47] Corrupted config files require graceful defaults instead of crashes


### [2026-04-06 08:36:31] Hidden dependencies trap agents: functions that use GLOBAL_CONFIG, Database.get_instance(), or singletons look pure by signature but fail in production. Agents can't see implicit state. Every function dependency must be explicit in parameters. This is the #1 reason agent projects degrade in production — each iteration introduces subtle state corruption from invisible blast radius










### [2026-04-06 11:46:28] Test fixture secrets cause false positives without proper exclusions


### [2026-04-06 11:46:28] Hidden dependencies trap agents — every function dependency must be explicit in parameters


### [2026-04-06 11:46:28] Duplicate entries across iterations degrade production code over time


### [2026-04-06 11:46:28] Go constants cannot be overridden by ldflags — use var instead


### [2026-04-06 11:46:28] Mutable state and hidden dependencies make agent output non-deterministic and impossible to debug


### [2026-04-06 11:46:28] Skipping normalization checks risks adding duplicate entries to the knowledge base


### [2026-04-06 11:46:28] Changelog entry date is set to future year 2026.


### [2026-04-06 12:07:46] brain review TCGETS ioctl can succeed on non-TTY stdin in some environments - check stdin is character device before raw mode








### [2026-04-06 13:43:38] Hidden dependencies trap agents: functions using GLOBAL_CONFIG or singletons look pure but fail in production






### [2026-04-06 13:43:38] EOF errors in TUI review needed graceful handling instead of error treatment


### [2026-04-06 13:43:38] stdin device checking required before TCGETS calls




### [2026-04-06 13:43:38] SSH sessions can close stdin without explicit user interaction, triggering unintended errors.


### [2026-04-06 13:43:38] EOF detection logic changed to distinguish between initial and sub-sequence reads.


### [2026-04-06 15:41:55] Non-blocking reads for terminal detection fail when no input is pending.


### [2026-04-06 15:41:55] Manual file descriptor syscalls are error-prone under varying terminal states.






### [2026-04-06 15:47:20] --tty flag previously forced TUI even when stdin was not a terminal














### [2026-04-06 15:48:51] Prior behavior of --tty flag forcing TUI is removed on non-terminals.










### [2026-04-06 16:02:42] Changelog version date is set to the future


### [2026-04-08 11:07:34] test


### [2026-04-08 11:08:23] Deploy uses Docker containers on Ubuntu VPS


### [2026-04-09 14:56:21] Error handling must record errors on spans via RecordError for proper telemetry


### [2026-04-09 14:56:21] Deploy different exporters (gRPC vs HTTP) based on endpoint configuration


### [2026-04-09 14:56:21] Graceful shutdown must call provider shutdown to flush pending spans


### [2026-04-09 14:56:21] AI coding agents reset to a blank slate on every new session without external memory.


### [2026-04-09 14:56:21] Direct LLM output requires human review to prevent noisy knowledge entries.


### [2026-04-09 14:56:21] SHA256 fingerprint uses only first 8 bytes which may cause collisions


### [2026-04-09 14:56:21] Entry normalization requires consistent case/format handling or different entries may be treated as duplicates


### [2026-04-09 14:56:21] Cross-topic duplicates keep entry based on topic alphabet order, not content quality


### [2026-04-09 14:56:21] File permissions set to 0600 may restrict access programmatically


### [2026-04-09 14:56:21] Empty files handled but could cause edge cases with file existence


### [2026-04-09 14:56:21] Duplicate entries across topic files can cause inconsistent knowledge retrieval.


### [2026-04-09 14:56:21] Ignoring warnings about duplicates detected leads to redundant memory consumption.




































### [2026-04-14 01:32:14] AdaptSkill in knowledge/adapt.go uses os.ReadFile directly to extract entry messages — this should use hub.Get() for consistency














### [2026-04-14 06:46:30] Avoid manual file descriptor syscalls for terminal detection as they are error-prone.


### [2026-04-14 06:46:30] Use variables, not constants, to support overrides via ldflags or build tags.


### [2026-04-14 06:46:30] Isolate test config writes to prevent corruption of production settings.


### [2026-04-14 06:46:30] Path-based exclusions are necessary to prevent test fixture false positives in security scanners.
























### [2026-04-14 06:46:30] Empty SKILL.md triggers early return skipping adaptation writing


### [2026-04-14 06:46:30] Thread-safe behavior tracking with mutex may bottleneck concurrent operations


### [2026-04-14 06:46:30] Weak entry threshold requires manual prune command triggering


### [2026-04-14 06:46:30] 0600 file permissions for behavior data may restrict read access for analysis












### [2026-04-14 19:13:33] Always confirm overwrites to modified skill files before updating


### [2026-04-14 19:13:33] Format strings for help banners must account for truncation limits
















### [2026-04-14 22:53:12] Duplicate entries require safe overwrite confirmation before updating


### [2026-04-15 05:54:00] String concatenation for file paths causes errors on non-Windows systems; use filepath.Join instead.


### [2026-04-15 05:54:00] Double-marshaling JSON sends corrupted base64 data to LLMs; avoid nested encoding.


### [2026-04-15 05:54:00] Direct TTY ioctl calls (TCGETS) fail on non-TTY inputs; detect terminal type before raw mode.


### [2026-04-15 05:54:00] Test config writes corrupt production settings; isolate with XDG_CONFIG_HOME.


### [2026-04-15 05:54:00] Non-blocking reads fail when stdin has no pending input; handle EOF conditions properly.












### [2026-04-15 13:29:10] CLI aliases were undocumented - 11 backward-compat aliases now documented in full help


### [2026-04-15 13:29:10] Future timestamps showing negative time applied by idx criteria, handle EOF properly when stdin empty


### [2026-04-15 13:29:10] Auto-restart daemon after update with error handling that may log warnings on failure


### [2026-04-15 13:29:10] Default Go exit of 0 even with errors causes silent failure


### [2026-04-15 13:29:10] Go strings package from strings import not strings


### [2026-04-15 13:29:10] Silent deprecation of aliases risks breaking user scripts relying on old commands.


### [2026-04-15 13:33:30] 
















### [2026-04-15 14:15:34] A gotcha entry




### [2026-04-15 16:05:39] Project uses argon2, NOT bcrypt


### [2026-04-15 16:24:55] CLI aliases deprecated breaking user scripts


### [2026-04-15 16:24:55] Go default exit code 0 causes silent failures


### [2026-04-15 16:24:55] Cross-platform TTY detection requires standard library packages


### [2026-04-15 16:24:55] Version inconsistency betwen CHANGELOG and build flags


### [2026-04-15 16:24:55] CLI aliases undocumented - backward compat alias deprecation risks breaking user scripts


### [2026-04-15 16:24:55] EOF handling during TUI review requires proper stdin check for graceful exit


### [2026-04-15 16:24:55] Future timestamps showing negative time applied by idx criteria may cause display issues.


### [2026-04-15 16:24:55] Go silent exit default of 0 requires explicit error handling even when logic fails


### [2026-04-15 16:24:55] Backward-compatible CLI aliases require proper documentation to avoid script breakage


### [2026-04-15 16:24:55] Daemon may log warnings on restart failures without stopping functionality


### [2026-04-15 16:24:55] Using external packages like strings import incorrectly can cause compilation errors


### [2026-04-15 16:24:55] Stale locks may persist on Windows crash due to best-effort file deletion


### [2026-04-15 16:24:55] PID lock files may become stale on Windows on daemon process crashes without auto-cleanup.


### [2026-04-15 16:24:55] Future timestamps may show negative time applied by idx criteria in review logic.


### [2026-04-15 16:24:55] Future timestamps appear as negative time when EOF handling is incomplete


### [2026-04-15 16:24:55] CLI aliases were undocumented and risk breaking user scripts


### [2026-04-15 16:24:55] Testing with same-topic and cross-topic duplicates covers different deduplication scenarios


### [2026-04-15 16:24:55] PID lock file on Windows may be stale on crash (best-effort handling)


### [2026-04-15 16:24:55] Go exit codes default to 0 causing silent failures if not handled explicitly


### [2026-04-15 16:24:55] Silent deprecation of CLI aliases risks breaking existing user scripts without migration


### [2026-04-15 16:24:55] Timestamps may appear negative due to index criteria applied to future dates


### [2026-04-15 16:24:55] Default handling of non-terminal stdin in TUI mode may cause unexpected graceful exits


### [2026-04-19 06:10:03] Test gotcha for v2 testing


### [2026-04-19 06:10:17] --global test


### [2026-04-19 09:30:06] --global QA test: global flag after topic


### [2026-04-19 09:50:43] test global flag position 1


### [2026-04-19 09:50:43] test global flag position 2


### [2026-04-19 09:50:43] test global flag position 3


### [2026-04-19 13:06:18] API key masking verified in config get


### [2026-04-19 13:06:18] v2.0.2 test: checksum verification works on update


### [2026-04-19 13:14:07] Go silent exit code 0 causes failures without explicit error handling


### [2026-04-19 13:14:07] Stale PID lock files persist on Windows crash without auto-cleanup


### [2026-04-19 13:14:07] TTL values not documented for retrievals and archives


### [2026-04-19 13:14:07] High thresholds on trigram-Jaccard dedup may miss similar entries


### [2026-04-19 13:14:07] Future timestamps showing negative time in review logic


### [2026-04-19 13:14:07] Unchecked stdin in TUI review causes fatal silent closure


### [2026-04-19 13:14:07] Version string defined as var instead of const to allow ldflags override


### [2026-04-19 13:14:07] API keys in URL query parameters get logged in server access logs


### [2026-04-19 13:14:07] Stale PID lock files may persist on Windows daemon crash


### [2026-04-19 13:14:07] Diff size caps prevent leaking internal hostnames


### [2026-04-19 13:14:07] Custom providers must be named explicitly instead of 'custom'

