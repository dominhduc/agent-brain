package profile

import "fmt"

type Profile struct {
	Name           string            `yaml:"name"`
	AutoAccept     bool              `yaml:"auto_accept"`
	AutoDedup      bool              `yaml:"auto_dedup"`
	TopicOverrides map[string]string `yaml:"topic_overrides,omitempty"`
}

func DefaultProfile() Profile {
	return Profile{Name: "guard", AutoAccept: false, AutoDedup: false, TopicOverrides: make(map[string]string)}
}

func FromName(name string) (Profile, error) {
	profiles := map[string]Profile{
		"guard":  {Name: "guard", AutoAccept: false, AutoDedup: false, TopicOverrides: make(map[string]string)},
		"assist": {Name: "assist", AutoAccept: false, AutoDedup: true, TopicOverrides: make(map[string]string)},
		"agent":  {Name: "agent", AutoAccept: true, AutoDedup: true, TopicOverrides: make(map[string]string)},
	}
	p, ok := profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown profile %q. Valid profiles: guard, assist, agent", name)
	}
	return p, nil
}

func ValidNames() []string { return []string{"guard", "assist", "agent"} }

func (p Profile) Validate() error {
	valid := map[string]bool{"guard": true, "assist": true, "agent": true}
	if !valid[p.Name] {
		return fmt.Errorf("invalid profile name %q", p.Name)
	}
	return nil
}

func (p Profile) Description() string {
	switch p.Name {
	case "guard":
		return "Every entry needs manual approval"
	case "assist":
		return "Auto-deduplicate similar entries, manual approval for new ones"
	case "agent":
		return "Auto-accept all entries, periodic summary only"
	default:
		return "Unknown profile"
	}
}
