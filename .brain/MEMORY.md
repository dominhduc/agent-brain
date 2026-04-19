# Project Memory Index

## Project
agent-brain — Go CLI tool that builds a knowledge base for coding agents. Daemon watches git commits, analyzes diffs via LLM, and writes findings to .brain/ topic files.

## Stack
Go 1.24, stdlib only (no external dependencies), net/http/httptest, t.TempDir(), t.Setenv()

## Commands
- `go build ./...` — build all packages
- `go test ./... -count=1 -race` — run all tests with race detection
- `GOOS=<os> GOARCH=<arch> go build -buildvcs=false -o /dev/null ./cmd/brain` — cross-platform build check
- `make build` — build binary to bin/brain
- `brain get <topic>` — get knowledge (use --summary for compact view, --json for JSON output)

## Key Patterns
- TDD for internal packages (write test first in internal/<pkg>/<pkg>_test.go)
- Cross-platform via //go:build tags (lock_posix.go, signal_posix.go, service_posix.go)
- Error messages include "What to do:" guidance
- Package extraction: create internal/<pkg>/, write tests, update main.go, delete old code
- Commit after every task, run tests before and after
- SUPER principles for agent-friendly code (S: Side Effects at Edge, U: Uncoupled Logic, P: Pure & Total Functions, E: Explicit Data Flow, R: Replaceable by Value)
- SPIRALS process loop for agent collaboration (Sense, Plan, Inquire, Refine, Act, Learn, Scan)
- Code must be machine-reasonable: no hidden state, no globals, no entangled side effects

## Active Gotchas
- Go const cannot be overridden by ldflags — use var instead
- sync.Once is NOT thread-safe with ResetCache — add sync.Mutex
- brain update needs GITHUB_TOKEN for private repos
- SetValue must validate numeric/duration fields with strconv/time
- filepath.Join() always, never string concat with /
- os.UserHomeDir() errors must be handled, never discard with ,_
- syscall.Flock doesn't exist on Windows — use build tags
- Hidden dependencies (globals, singletons) trap agents — every dependency must be explicit in function parameters

## Topic Files
- `gotchas.md` — Error patterns and fixes
- `patterns.md` — Discovered conventions
- `architecture.md` — Module structure and relationships
- `decisions.md` — Architecture decisions and rationale

## Knowledge Base Stats (after cleanup)
- 392 total entries across 5 topics (63 stale entries removed)
- 0 pending daemon entries, 0 duplicates
- `.brainprune` file created for ongoing stale entry management

## Last Updated
2026-04-15
