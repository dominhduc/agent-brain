package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const PrePushHookContent = `#!/bin/bash
# Pre-push hook for agent-brain
# Installed by: brain init
# Triggers analysis when code is pushed to remote (stable code only)

BRAIN_DIR="$(git rev-parse --show-toplevel)/.brain"
QUEUE_DIR="$BRAIN_DIR/.queue"
PROJECT_DIR="$(git rev-parse --show-toplevel)"

# Sync knowledge to docs/brain/ before push (non-blocking)
if command -v brain >/dev/null 2>&1; then
    brain sync 2>/dev/null || true
fi

if [ ! -d "$QUEUE_DIR" ]; then
    exit 0
fi

while read local_ref local_sha remote_ref remote_sha; do
    if [ "$remote_sha" = "0000000000000000000000000000000000000000" ]; then
        # New branch — diff against empty tree
        DIFF_STAT=$(git diff --stat 4b825dc642cb6eb9a060e54bf899d69f8272690f..$local_sha 2>/dev/null || echo "No diff")
        FILES=$(git diff --name-status 4b825dc642cb6eb9a060e54bf899d69f8272690f..$local_sha 2>/dev/null || echo "No diff")
    else
        DIFF_STAT=$(git diff --stat $remote_sha..$local_sha 2>/dev/null || echo "No diff")
        FILES=$(git diff --name-status $remote_sha..$local_sha 2>/dev/null || echo "No diff")
    fi

    REPO=$(git rev-parse --show-toplevel)
    TIMESTAMP=$(date +%Y%m%dT%H%M%S)

    escape_json() {
        printf '%s' "$1" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))' 2>/dev/null || printf '"%s"' "$(echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | tr '\n' ' ')"
    }

    DIFF_ESCAPED=$(escape_json "$DIFF_STAT")
    FILES_ESCAPED=$(escape_json "$FILES")

    cat > "$QUEUE_DIR/commit-${TIMESTAMP}.json" << EOF
{
  "timestamp": "${TIMESTAMP}",
  "repo": "${REPO}",
  "diff_stat": ${DIFF_ESCAPED},
  "files": ${FILES_ESCAPED},
  "attempts": 0,
  "status": "pending"
}
EOF
done
`

func InstallPrePushHook(cwd string) error {
	hooksDir := filepath.Join(cwd, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	hookPath := filepath.Join(hooksDir, "pre-push")

	if _, err := os.Stat(hookPath); err == nil {
		existing, err := os.ReadFile(hookPath)
		if err != nil {
			return fmt.Errorf("cannot read existing hook: %w", err)
		}
		if strings.Contains(string(existing), "agent-brain") {
			return nil
		}
		backupPath := hookPath + ".bak"
		if err := os.Rename(hookPath, backupPath); err != nil {
			return fmt.Errorf("cannot back up existing hook: %w", err)
		}
		fmt.Printf("Existing pre-push hook backed up to %s\n", backupPath)
	}

	return os.WriteFile(hookPath, []byte(PrePushHookContent), 0700)
}
