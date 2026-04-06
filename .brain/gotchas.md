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


### [2026-04-05 07:57:51] Debug output may be enabled in production accidentally.


### [2026-04-05 07:57:51] Debug output may be enabled in production by accident


### [2026-04-05 08:34:34] PostJSON double-marshaling sends base64-encoded garbage to LLM APIs instead of raw JSON.


### [2026-04-05 08:34:34] Direct TTY operations cause TCGETS errors when not in interactive terminal.


### [2026-04-05 08:34:34] Testing config writes can corrupt live settings without proper isolation.


### [2026-04-05 08:34:34] String-based path concatenation breaks on cross-platform environments.


### [2026-04-05 08:34:34] Double-marshaling PostJSON returned base64 garbage to LLM APIs.


### [2026-04-05 08:34:34] Non-TTY inputs initially caused TCGETS errors in review commands.


### [2026-04-05 08:34:34] Corrupted config files required specific logic to return defaults gracefully.


### [2026-04-05 10:33:34] Test entry 1


### [2026-04-05 10:39:47] PostJSON double-marshaling sends base64-encoded garbage to LLM APIs instead of raw JSON


### [2026-04-05 10:39:47] String-based path concatenation breaks on cross-platform environments


### [2026-04-05 10:39:47] Direct TTY operations cause TCGETS errors when not in interactive terminal


### [2026-04-05 10:39:47] Testing config writes can corrupt live settings without proper isolation


### [2026-04-05 10:39:47] Direct TTY operations without TCGETS detection cause TCGETS errors in non-interactive mode


### [2026-04-05 10:39:47] Non-TTY inputs cause TCGETS errors in review commands


### [2026-04-05 10:39:47] Corrupted config files require graceful defaults instead of crashes


### [2026-04-06 08:36:31] Hidden dependencies trap agents: functions that use GLOBAL_CONFIG, Database.get_instance(), or singletons look pure by signature but fail in production. Agents can't see implicit state. Every function dependency must be explicit in parameters. This is the #1 reason agent projects degrade in production — each iteration introduces subtle state corruption from invisible blast radius


### [2026-04-06 11:46:28] Double-marshaling PostJSON sends base64 garbage instead of raw JSON to LLM APIs.


### [2026-04-06 11:46:28] Direct TTY operations cause TCGETS errors when not in interactive terminal mode.


### [2026-04-06 11:46:28] String-based path concatenation breaks on cross-platform environments without filepath utilities.


### [2026-04-06 11:46:28] Testing config writes can corrupt live settings without proper isolation.


### [2026-04-06 11:46:28] Test fixture secrets cause false positives without proper exclusions


### [2026-04-06 11:46:28] Hidden dependencies trap agents — every function dependency must be explicit in parameters


### [2026-04-06 11:46:28] Duplicate entries across iterations degrade production code over time


### [2026-04-06 11:46:28] Go constants cannot be overridden by ldflags — use var instead


### [2026-04-06 11:46:28] Mutable state and hidden dependencies make agent output non-deterministic and impossible to debug


### [2026-04-06 11:46:28] Skipping normalization checks risks adding duplicate entries to the knowledge base


### [2026-04-06 11:46:28] Changelog entry date is set to future year 2026.


### [2026-04-06 12:07:46] brain review TCGETS ioctl can succeed on non-TTY stdin in some environments - check stdin is character device before raw mode


### [2026-04-06 13:43:38] Double-marshaling POST JSON sends base64 garbage instead of raw JSON to LLM APIs


### [2026-04-06 13:43:38] Direct TTY operations cause TCGETS errors when not in interactive terminal mode


### [2026-04-06 13:43:38] String-based path concatenation breaks on cross-platform environments without filepath utilities


### [2026-04-06 13:43:38] Hidden dependencies trap agents: functions using GLOBAL_CONFIG or singletons look pure but fail in production


### [2026-04-06 13:43:38] Testing config writes can corrupt live settings without proper isolation


### [2026-04-06 13:43:38] Go constants cannot be overridden by ldflags — use var instead


### [2026-04-06 13:43:38] EOF errors in TUI review needed graceful handling instead of error treatment


### [2026-04-06 13:43:38] stdin device checking required before TCGETS calls


### [2026-04-06 13:43:38] TCGETS ioctl may succeed on non-TTY stdin in some environments - must verify stdin is character device before entering raw mode


### [2026-04-06 13:43:38] SSH sessions can close stdin without explicit user interaction, triggering unintended errors.


### [2026-04-06 13:43:38] EOF detection logic changed to distinguish between initial and sub-sequence reads.


### [2026-04-06 15:41:55] Non-blocking reads for terminal detection fail when no input is pending.


### [2026-04-06 15:41:55] Manual file descriptor syscalls are error-prone under varying terminal states.

