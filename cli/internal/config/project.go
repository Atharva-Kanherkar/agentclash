package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfigFile is the filename for project-level configuration.
const ProjectConfigFile = ".agentclash.yaml"

// ProjectConfig holds project-level settings found in .agentclash.yaml.
//
// Deployments is the (deployment name → deployment id) map written by
// `agentclash link` / `agentclash init --provision`. `agentclash eval
// --models foo,bar` resolves each comma-separated name against this map so
// humans never have to paste deployment UUIDs.
type ProjectConfig struct {
	WorkspaceID   string            `yaml:"workspace_id,omitempty"`
	WorkspaceName string            `yaml:"workspace_name,omitempty"`
	OrgID         string            `yaml:"org_id,omitempty"`
	Deployments   map[string]string `yaml:"deployments,omitempty"`
}

// FindProjectConfig searches upward from the current directory for .agentclash.yaml.
// Returns nil if no project config is found.
func FindProjectConfig() *ProjectConfig {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}

	for {
		path := filepath.Join(dir, ProjectConfigFile)
		data, err := os.ReadFile(path)
		if err == nil {
			var cfg ProjectConfig
			if yaml.Unmarshal(data, &cfg) == nil {
				return &cfg
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

// WriteProjectConfig writes a .agentclash.yaml in the given directory.
func WriteProjectConfig(dir string, cfg ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ProjectConfigFile), data, 0644)
}
