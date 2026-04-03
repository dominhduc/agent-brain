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

