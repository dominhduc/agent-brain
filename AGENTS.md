# Project Instructions

## Knowledge Base
This project uses a `.brain/` knowledge hub managed by the `brain` CLI.

### At Session Start
Run `brain get all` to load accumulated project knowledge before starting work.

### During Work
- Run `brain search <topic>` before writing code against unfamiliar patterns
- Run `brain get gotchas` before debugging to avoid known pitfalls

### Self-Evolution
When I correct you, express frustration about a repeated mistake, or point out a pattern:
1. Add the learning: `brain add gotcha "..."` or `brain add pattern "..."`
2. Update MEMORY.md if the index needs refreshing
3. Treat every correction as permanent — don't repeat mistakes

### At Session End
Run `brain eval` to write a self-evaluation to the current session file.
Include: what you did, what worked, what failed, confidence scores, knowledge persisted.

### Confidence Reporting
Always report confidence on technical decisions:
- HIGH: documented best practice, matches codebase patterns
- MEDIUM: reasonable approach, alternatives exist
- LOW: best guess, recommend verification
When confidence is below HIGH, state what would increase it and the risks.

### Clarifying Questions
If requirements are ambiguous, ask BEFORE coding. Present 2-3 options with tradeoffs.

### Safety Rules
- NEVER delete files or run destructive commands without explicit approval
- NEVER read or expose `.env` files or secrets
- Flag risky changes (auth, payments, data mutations) and wait for my review

## Project Overview
[Auto-populated by daemon analysis or first agent session]

## Stack
[Auto-populated by daemon analysis or first agent session]

## Commands
[Auto-populated by daemon analysis or first agent session]
