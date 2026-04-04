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

