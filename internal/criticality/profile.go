package criticality

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProfileConfig struct {
	Version  string         `yaml:"version"`
	Profiles []ProfileMatch `yaml:"profiles"`
}

type ProfileMatch struct {
	Match       string `yaml:"match"`
	Criticality string `yaml:"criticality"`
}

func GetMultiplier(level string) float64 {
	switch strings.ToLower(level) {
	case "critical":
		return 1.00
	case "high":
		return 0.80
	case "medium":
		return 0.60
	case "low":
		return 0.30
	case "minimal":
		return 0.10
	default:
		return 0.60
	}
}

func LoadProfile() (*ProfileConfig, error) {
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ".kubectl-plan", "criticality.yaml")
		if _, err := os.Stat(path); err == nil {
			return loadFile(path)
		}
	}

	path := "./kubectl-plan-criticality.yaml"
	if _, err := os.Stat(path); err == nil {
		return loadFile(path)
	}

	return nil, nil
}

func loadFile(path string) (*ProfileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &ProfileConfig{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func EvaluateNamespace(ns string, cfg *ProfileConfig) (string, float64) {
	if cfg != nil {
		for _, p := range cfg.Profiles {
			if matchPattern(ns, p.Match) {
				return p.Criticality, GetMultiplier(p.Criticality)
			}
		}
	}

	if strings.Contains(ns, "prod") {
		return "medium", 0.60
	}
	if strings.Contains(ns, "stage") || strings.Contains(ns, "test") {
		return "low", 0.30
	}
	if strings.Contains(ns, "dev") {
		return "minimal", 0.10
	}
	return "medium", 0.60
}

func matchPattern(val, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(val, prefix)
	}
	return val == pattern
}
