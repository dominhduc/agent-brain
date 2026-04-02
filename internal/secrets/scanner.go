package secrets

import (
	"regexp"
	"strings"
)

type Finding struct {
	Type     string
	Line     string
	FileName string
}

var patterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"AWS Access Key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"AWS Secret Key", regexp.MustCompile(`(?i)(aws_secret_access_key|aws_secret_key)\s*[:=]\s*[A-Za-z0-9/+=]{40}`)},
	{"GitHub Token", regexp.MustCompile(`gh[porsu]_[A-Za-z0-9_]{36,}`)},
	{"GitLab Token", regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`)},
	{"Private Key", regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)},
	{"Generic Secret", regexp.MustCompile(`(?i)(api[_-]?key|secret[_-]?key|access[_-]?token|auth[_-]?token|password|passwd|credential)\s*[:=]\s*["']?[A-Za-z0-9\-_.]{20,}["']?`)},
	{"Slack Token", regexp.MustCompile(`xox[baprs]-[0-9]{10,}-[0-9]{10,}-[a-zA-Z0-9]{24,}`)},
	{"Stripe Key", regexp.MustCompile(`(?i)(sk|pk)_(test|live)_[A-Za-z0-9]{24,}`)},
	{"Database URL", regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[^\s'"<>]+:[^\s'"<>]+@[^\s'"<>]+`)},
	{"JWT", regexp.MustCompile(`eyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+`)},
	{".env Variable", regexp.MustCompile(`(?m)^[A-Z][A-Z0-9_]*=["']?[A-Za-z0-9\-_.+/=]{16,}["']?\s*$`)},
}

func Scan(content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")

	for _, p := range patterns {
		matches := p.pattern.FindAllStringIndex(content, -1)
		for _, match := range matches {
			_ = content[match[0]:match[1]]
			lineNum := countLines(content[:match[0]])
			line := strings.TrimSpace(lines[lineNum])
			if len(line) > 200 {
				line = line[:200] + "..."
			}
			findings = append(findings, Finding{
				Type:     p.name,
				Line:     line,
				FileName: "",
			})
		}
	}

	return findings
}

func ScanDiff(diff string) []Finding {
	return Scan(diff)
}

func HasSecrets(content string) bool {
	return len(Scan(content)) > 0
}

func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}
