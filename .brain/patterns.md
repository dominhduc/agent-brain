# Patterns
<!-- Entries added by brain add pattern or daemon analysis -->

### [2026-04-03 08:42:34] Error messages follow 'What to do:' guidance format — always tell user how to resolve the error, not just what went wrong


### [2026-04-03 08:42:34] TDD for internal packages — write test file first in internal/<pkg>/<pkg>_test.go, verify it fails, then implement


### [2026-04-03 08:42:34] Cross-platform files use //go:build tags — lock_posix.go (!windows), lock_windows.go (windows), signal_posix.go, signal_windows.go


### [2026-04-03 08:42:34] Package extraction pattern: create internal/<pkg>/<pkg>.go + _test.go, add import to main.go, replace inline code with package calls, delete old code from main.go


### [2026-04-03 08:42:34] Service registration uses runtime.GOOS switch — platform-specific implementations in service_posix.go and service_windows.go

