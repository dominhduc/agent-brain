package profile

import (
	"testing"
)

func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile()

	if p.Name != "guard" {
		t.Errorf("expected name 'guard', got %q", p.Name)
	}
	if p.AutoAccept {
		t.Error("expected AutoAccept false for guard")
	}
	if p.AutoDedup {
		t.Error("expected AutoDedup false for guard")
	}
	if p.TopicOverrides == nil {
		t.Error("expected non-nil TopicOverrides")
	}
}

func TestFromName_Valid(t *testing.T) {
	tests := []struct {
		name       string
		autoAccept bool
		autoDedup  bool
	}{
		{"guard", false, false},
		{"assist", false, true},
		{"agent", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := FromName(tt.name)
			if err != nil {
				t.Fatalf("FromName(%q) unexpected error: %v", tt.name, err)
			}
			if p.Name != tt.name {
				t.Errorf("expected name %q, got %q", tt.name, p.Name)
			}
			if p.AutoAccept != tt.autoAccept {
				t.Errorf("expected AutoAccept %v, got %v", tt.autoAccept, p.AutoAccept)
			}
			if p.AutoDedup != tt.autoDedup {
				t.Errorf("expected AutoDedup %v, got %v", tt.autoDedup, p.AutoDedup)
			}
			if p.TopicOverrides == nil {
				t.Error("expected non-nil TopicOverrides")
			}
		})
	}
}

func TestFromName_Invalid(t *testing.T) {
	_, err := FromName("unknown")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestValidNames(t *testing.T) {
	names := ValidNames()
	expected := []string{"guard", "assist", "agent"}

	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, n := range expected {
		if names[i] != n {
			t.Errorf("expected names[%d] = %q, got %q", i, n, names[i])
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"guard", false},
		{"assist", false},
		{"agent", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Profile{Name: tt.name}
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDescription(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"guard", "Every entry needs manual approval"},
		{"assist", "Auto-deduplicate similar entries, manual approval for new ones"},
		{"agent", "Auto-accept all entries, periodic summary only"},
		{"unknown", "Unknown profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Profile{Name: tt.name}
			if got := p.Description(); got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}
