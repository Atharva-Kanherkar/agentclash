package scanners

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const SchemaV1 = "scanner/v1"

type Kind string

const (
	KindPattern Kind = "pattern"
	KindLLM     Kind = "llm"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Definition is a named, versioned scanner.
type Definition struct {
	Schema      string   `yaml:"schema" json:"schema"`
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Kind        Kind     `yaml:"kind" json:"kind"`
	Category    string   `yaml:"category" json:"category"`
	Severity    Severity `yaml:"severity" json:"severity"`
	Description string   `yaml:"description" json:"description"`
	Pattern     *PatternSpec `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	LLM         *LLMSpec     `yaml:"llm,omitempty" json:"llm,omitempty"`
}

type PatternSpec struct {
	// AnyMatch: if any regex matches transcript text, emit a finding.
	AnyMatch []string `yaml:"any_match" json:"any_match"`
}

type LLMSpec struct {
	Prompt   string `yaml:"prompt" json:"prompt"`
	// SchemaVersion pins the expected JSON verdict shape.
	SchemaVersion int `yaml:"schema_version" json:"schema_version"`
}

// Finding is a scanner hit (not yet persisted).
type Finding struct {
	Scanner     string   `json:"scanner"`
	Version     string   `json:"version"`
	Severity    Severity `json:"severity"`
	Category    string   `json:"category"`
	Evidence    string   `json:"evidence"`
	Confidence  float64  `json:"confidence"`
	MatchedRule string   `json:"matched_rule,omitempty"`
}

// LLMVerdict is the structured output required from LLM scanners.
type LLMVerdict struct {
	SchemaVersion int      `json:"schema_version"`
	Hit           bool     `json:"hit"`
	Severity      Severity `json:"severity"`
	Category      string   `json:"category"`
	Evidence      string   `json:"evidence"`
	Confidence    float64  `json:"confidence"`
}

func ParseDefinition(data []byte) (Definition, error) {
	var d Definition
	if err := yaml.Unmarshal(data, &d); err != nil {
		return Definition{}, fmt.Errorf("parse scanner: %w", err)
	}
	if err := d.Validate(); err != nil {
		return Definition{}, err
	}
	return d, nil
}

func (d Definition) Validate() error {
	if d.Schema != SchemaV1 {
		return fmt.Errorf("unsupported scanner schema %q", d.Schema)
	}
	if strings.TrimSpace(d.Name) == "" || strings.TrimSpace(d.Version) == "" {
		return fmt.Errorf("scanner name and version are required")
	}
	switch d.Kind {
	case KindPattern:
		if d.Pattern == nil || len(d.Pattern.AnyMatch) == 0 {
			return fmt.Errorf("pattern scanner requires pattern.any_match")
		}
		for _, p := range d.Pattern.AnyMatch {
			if _, err := regexp.Compile(p); err != nil {
				return fmt.Errorf("invalid pattern %q: %w", p, err)
			}
		}
	case KindLLM:
		if d.LLM == nil || strings.TrimSpace(d.LLM.Prompt) == "" {
			return fmt.Errorf("llm scanner requires llm.prompt")
		}
		if d.LLM.SchemaVersion <= 0 {
			d.LLM.SchemaVersion = 1
		}
	default:
		return fmt.Errorf("unknown scanner kind %q", d.Kind)
	}
	return nil
}

// LoadCatalogDir loads all *.yaml scanners from a directory.
func LoadCatalogDir(dir string) ([]Definition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Definition, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		def, err := ParseDefinition(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, def)
	}
	return out, nil
}
