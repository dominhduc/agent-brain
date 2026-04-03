# Patterns
<!-- Entries added by brain add pattern or daemon analysis -->

### [2026-04-03 08:42:34] Error messages follow 'What to do:' guidance format — always tell user how to resolve the error, not just what went wrong


### [2026-04-03 08:42:34] TDD for internal packages — write test file first in internal/<pkg>/<pkg>_test.go, verify it fails, then implement


### [2026-04-03 08:42:34] Cross-platform files use //go:build tags — lock_posix.go (!windows), lock_windows.go (windows), signal_posix.go, signal_windows.go


### [2026-04-03 08:42:34] Package extraction pattern: create internal/<pkg>/<pkg>.go + _test.go, add import to main.go, replace inline code with package calls, delete old code from main.go


### [2026-04-03 08:42:34] Service registration uses runtime.GOOS switch — platform-specific implementations in service_posix.go and service_windows.go


### [2026-04-03 09:06:05] Public repo readiness checklist: (1) Remove dev-only files plans from tracking,2) gitignore dev notes, (3) verify no real secrets/paths/email, (4) verify no hardcoded IP/internal URLs, (5) run secret scanner on all tracked files


### [2026-04-04 06:43:44] After releasing new CLI features: 1) Bump version in CHANGELOG.md 2) Update printUsage() in cmd.go 3) Don't forget either - both need updating together

