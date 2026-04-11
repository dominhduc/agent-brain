# Troubleshooting

Common errors, diagnostics, and fixes for the `brain` CLI.

## Initialization Issues

### "Knowledge hub already exists in this project"

**Cause:** `.brain/` directory already exists from a previous `brain init`.

**Fix:**
```bash
# To reinitialize cleanly:
rm -rf .brain/
brain init

# Or keep existing data and just reinstall skills:
brain skill update
```

### "Cannot determine current directory"

**Cause:** Running in a deleted or inaccessible directory.

**Fix:** Navigate to a valid directory first.

### "Not a git repository"

**Cause:** `brain init` requires a git repository.

**Fix:**
```bash
git init
brain init
```

### "No commits found"

**Cause:** The repository has no commits yet. The daemon needs at least one commit to function.

**Fix:** Make an initial commit, then run `brain init` again.

## Daemon Issues

### "Daemon not running"

**Diagnosis:**
```bash
brain daemon status
brain doctor
```

**Fix:**
```bash
brain daemon start
# Or restart:
brain daemon restart
```

### "Could not register daemon"

**Cause:** systemd/launchd registration failed (permissions or service conflict).

**Fix:**
```bash
# Check if service is already registered:
brain daemon status

# Manually start without service registration:
brain daemon run

# On Linux, check systemd:
systemctl --user status brain-<project-name>

# On macOS, check launchd:
launchctl list | grep brain
```

### "Queue processing failed"

**Diagnosis:**
```bash
brain daemon failed
brain status --json | jq '.queue'
```

**Fix:**
```bash
# Retry failed items:
brain daemon retry

# If queue is stuck, force a processing cycle:
brain daemon run

# Check API key:
brain config get llm.api-key
```

## API & LLM Issues

### "OpenRouter API key not configured"

**Fix:**
```bash
brain config set api-key <your-openrouter-key>
# Or set environment variable:
export OPENROUTER_API_KEY="your-key"
```

### "LLM analysis failed"

**Cause:** API rate limits, invalid key, or network issues.

**Diagnosis:**
```bash
brain doctor
# Check the "LLM API" section
```

**Fix:**
- Verify API key: `brain config get llm.api-key`
- Check rate limits on OpenRouter dashboard
- Ensure network connectivity

### "Secrets detected, commit skipped"

**Cause:** The pre-push hook detected potential secrets in the diff.

**Fix:**
- Remove the secret from your changes
- Add the file to `.gitignore`
- Use environment variables or a secrets manager instead
- If it's a false positive, add a `.brainsecrets` file to whitelist patterns

## Skill Issues

### "Skill files not found"

**Cause:** Skills were never installed or were deleted.

**Fix:**
```bash
brain skill update
# Or reinstall:
brain skill install
```

### "Skill not triggering"

**Cause:** The skill description may not match the user's query keywords.

**Fix:**
- Invoke directly: `/agent-brain`
- Check skill is in the right location: `brain skill list`
- Verify the skill file is valid YAML + markdown

### "Allowed-tools not working"

**Cause:** The agent may not support the `allowed-tools` frontmatter field.

**Fix:** This is a limitation of the agent, not the skill. The skill will still work but may require per-command approval.

## Knowledge Hub Issues

### "No entries found"

**Cause:** The daemon hasn't processed any commits yet, or all entries are archived.

**Diagnosis:**
```bash
brain status
brain daemon status
```

**Fix:**
- Make a commit to trigger the queue
- Run `brain daemon run` to force processing
- Check `brain review` for pending entries

### "Duplicate entries"

**Fix:**
```bash
brain dedup --dry-run   # Preview
brain dedup             # Execute
```

### "Too many stale entries"

**Fix:**
```bash
# Create prune patterns:
echo "v0.20.0" >> .brainprune
echo "old-api-endpoint" >> .brainprune

brain prune --dry-run   # Preview
brain prune             # Execute
```

## Git Issues

### "Post-commit hook failed"

**Diagnosis:**
```bash
cat .git/hooks/post-commit
# Check the hook content
```

**Fix:**
```bash
# Reinstall the hook:
brain init
# Or manually:
chmod +x .git/hooks/post-commit
```

### "Pre-push hook not analyzing"

**Cause:** The pre-push hook only analyzes "stable shifted code" — it may skip if the push includes unstable branches.

**Diagnosis:**
```bash
cat .git/hooks/pre-push
```

**Fix:** Ensure you're pushing to a tracked branch with proper upstream configuration.

## Performance Issues

### "brain is slow"

**Diagnosis:**
```bash
brain status
# Check entry count and hub size
```

**Fix:**
```bash
# Consolidate memory:
brain sleep

# Archive old entries:
brain prune

# Rebuild index (if search is slow):
brain index rebuild
```

### "Context window filling up"

**Cause:** Too many skill files or large knowledge entries.

**Fix:**
- Use `brain get all --summary` for compact output
- Prune stale entries: `brain prune`
- Archive old sessions: `brain sleep`

## Platform-Specific Issues

### Linux (systemd)

```bash
# Check service:
systemctl --user status brain-*

# View logs:
journalctl --user -u brain-* -f

# Restart service:
systemctl --user restart brain-*
```

### macOS (launchd)

```bash
# List services:
launchctl list | grep brain

# View logs:
log show --predicate 'process == "brain"' --last 1h

# Restart service:
brain daemon restart
```

### Windows

```bash
# Check if running:
tasklist | findstr brain

# Run daemon manually:
brain daemon run
```

## Getting Help

```bash
# Full health check:
brain doctor

# Show version and build info:
brain version

# Check configuration:
brain config list

# View recent activity:
brain status
```

If issues persist, check the logs in `.brain/.queue/` for failed queue items, or run `brain daemon failed` to see error details.
