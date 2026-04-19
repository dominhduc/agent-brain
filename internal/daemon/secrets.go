package daemon

import (
	"github.com/dominhduc/agent-brain/internal/secrets"
)

type SecretFinding = secrets.Finding

func ScanSecrets(content string) []SecretFinding {
	return secrets.Scan(content)
}

func ScanDiffSecrets(diff string) []SecretFinding {
	return secrets.Scan(diff)
}

func HasSecrets(content string) bool {
	return secrets.HasSecrets(content)
}
