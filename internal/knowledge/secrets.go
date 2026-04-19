package knowledge

import (
	"github.com/dominhduc/agent-brain/internal/secrets"
)

type SecretFinding = secrets.Finding

func ScanSecrets(content string) []SecretFinding {
	return secrets.Scan(content)
}

func HasSecrets(content string) bool {
	return secrets.HasSecrets(content)
}
