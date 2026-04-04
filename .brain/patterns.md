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

