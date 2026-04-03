package secrets

import (
	"strings"
	"testing"
)

func TestHasSecrets_AWSAccessKey(t *testing.T) {
	content := "aws_access_key = AKIAI" + "OSFODNN7EXAMPLE"
	if !HasSecrets(content) {
		t.Error("expected AWS access key to be detected")
	}
}

func TestHasSecrets_GitHubToken(t *testing.T) {
	content := "token = ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn"
	if !HasSecrets(content) {
		t.Error("expected GitHub token to be detected")
	}
}

func TestHasSecrets_PrivateKey(t *testing.T) {
	content := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA..."
	if !HasSecrets(content) {
		t.Error("expected private key to be detected")
	}
}

func TestHasSecrets_CleanContent(t *testing.T) {
	content := "this is just a normal line of code\nvar x = 42\nconsole.log('hello')"
	if HasSecrets(content) {
		t.Error("expected clean content to not trigger detection")
	}
}

func TestHasSecrets_EmptyString(t *testing.T) {
	if HasSecrets("") {
		t.Error("expected empty string to not trigger detection")
	}
}

func TestHasSecrets_ShortPassword(t *testing.T) {
	content := "password = short"
	if HasSecrets(content) {
		t.Error("expected short password value (< 20 chars) to not trigger detection")
	}
}

func TestScan_ReturnsFindingType(t *testing.T) {
	content := "AWS_ACCESS_KEY=AKIAI" + "OSFODNN7EXAMPLE"
	findings := Scan(content)
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	if findings[0].Type != "AWS Access Key" {
		t.Errorf("expected finding type 'AWS Access Key', got '%s'", findings[0].Type)
	}
}

func TestScan_ReturnsLineNumber(t *testing.T) {
	lines := []string{
		"first line",
		"second line",
		"aws_key = AKIAI" + "OSFODNN7EXAMPLE",
	}
	content := strings.Join(lines, "\n")
	findings := Scan(content)
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	if !strings.Contains(findings[0].Line, "AKIA") {
		t.Errorf("expected finding line to contain the key, got '%s'", findings[0].Line)
	}
}

func TestScanAllPatterns(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"AWS Access Key",
			"key = AKIAI" + "OSFODNN7EXAMPLE",
			"AWS Access Key",
		},
		{
			"AWS Secret Key",
			"aws_secret_access_key = ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij1234",
			"AWS Secret Key",
		},
		{
			"GitHub Token ghp",
			"token = ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn",
			"GitHub Token",
		},
		{
			"GitLab Token",
			"token = glpat-" + "ABCDEFGHIJKLMNOPQRST",
			"GitLab Token",
		},
		{
			"RSA Private Key",
			"-----BEGIN RSA PRIVATE KEY-----\ncontent",
			"Private Key",
		},
		{
			"EC Private Key",
			"-----BEGIN EC PRIVATE KEY-----\ncontent",
			"Private Key",
		},
		{
			"OpenSSH Private Key",
			"-----BEGIN OPENSSH PRIVATE KEY-----\ncontent",
			"Private Key",
		},
		{
			"Generic API Key",
			"api_key = abcdefghijklmnopqrstuvwxyz12345",
			"Generic Secret",
		},
		{
			"Generic Secret Key",
			"secret_key = abcdefghijklmnopqrstuvwxyz12345",
			"Generic Secret",
		},
		{
			"Generic Access Token",
			"access_token = abcdefghijklmnopqrstuvwxyz12345",
			"Generic Secret",
		},
		{
			"Generic Auth Token",
			"auth_token = abcdefghijklmnopqrstuvwxyz12345",
			"Generic Secret",
		},
		{
			"Generic Password",
			"password = abcdefghijklmnopqrstuvwxyz12345",
			"Generic Secret",
		},
		{
			"Slack Token",
			"token = xoxb-" + "1234567890-0987654321-abcdefghijklmnopqrstuvwxyz",
			"Slack Token",
		},
		{
			"Stripe Secret Key",
			"key = sk_live_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			"Stripe Key",
		},
		{
			"Stripe Public Key",
			"key = pk_test_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			"Stripe Key",
		},
		{
			"Database URL postgres",
			"DATABASE_URL=postgres://user:password@localhost:5432/mydb",
			"Database URL",
		},
		{
			"Database URL mysql",
			"DB=mysql://admin:secret123@db.example.com:3306/prod",
			"Database URL",
		},
		{
			"Database URL redis",
			"url = redis://default:redispassword@127.0.0.1:6379",
			"Database URL",
		},
		{
			"JWT",
			"token = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			"JWT",
		},
		{
			".env Variable",
			"DATABASE_PASSWORD=abcdefghijklmnop",
			".env Variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := Scan(tt.content)
			found := false
			for _, f := range findings {
				if f.Type == tt.want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected to detect '%s' in content, got %d findings", tt.want, len(findings))
			}
		})
	}
}

func TestScan_FalsePositives(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"short variable value", "MY_VAR=short"},
		{"example password in docs", "password = example"},
		{"placeholder text", "api_key = your-key-here"},
		{"commented out", "# api_key = something"},
		{"lowercase env var", "my_api_key=abcdefghijklmnop"},
		{"normal code", "func main() { fmt.Println(\"hello\") }"},
		{"URL without credentials", "https://example.com/api"},
		{"empty password", "password = "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if HasSecrets(tt.content) {
				findings := Scan(tt.content)
				types := make([]string, len(findings))
				for i, f := range findings {
					types[i] = f.Type
				}
				t.Errorf("false positive: detected secrets %v in clean content: %s", types, tt.content)
			}
		})
	}
}

func TestScanDiff(t *testing.T) {
	diff := "diff --git a/config.yaml b/config.yaml\n+api_key: sk_live_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	findings := ScanDiff(diff)
	if len(findings) == 0 {
		t.Error("expected ScanDiff to detect Stripe key in diff")
	}
}
