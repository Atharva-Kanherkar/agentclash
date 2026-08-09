// Package evalset parses and expands agentclash.evalset.yaml manifests
// (Fleet eval-set primitive).
package evalset

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SchemaV1         = "evalset/v1"
	DefaultMaxCombos = 2000
	// MaxAllowedCombos is the hard server-side ceiling for max_combinations.
	// Clients may request a lower cap, but never raise this limit.
	MaxAllowedCombos = 2000
	DefaultRepeats   = 1
)

// Manifest is the declarative eval-set face (YAML or JSON).
type Manifest struct {
	Schema     string       `yaml:"schema" json:"schema"`
	Name       string       `yaml:"name" json:"name"`
	Packs      []string     `yaml:"packs" json:"packs"`
	Agents     []AgentEntry `yaml:"agents" json:"agents"`
	Models     []string     `yaml:"models" json:"models"`
	Repeats    int          `yaml:"repeats" json:"repeats"`
	Seeds      *SeedConfig  `yaml:"seeds" json:"seeds,omitempty"`
	Limits     Limits       `yaml:"limits" json:"limits"`
	CaseFanout bool         `yaml:"case_fanout" json:"case_fanout"`
}

// AgentEntry is one lineup axis entry.
type AgentEntry struct {
	Deployment string `yaml:"deployment" json:"deployment"`
	Label      string `yaml:"label,omitempty" json:"label,omitempty"`
}

// SeedConfig controls per-combination seeds.
type SeedConfig struct {
	Strategy string  `yaml:"strategy" json:"strategy"`
	Seeds    []int64 `yaml:"seeds,omitempty" json:"seeds,omitempty"`
}

// Limits are set-level caps (budget enforced later by Fleet 13).
type Limits struct {
	MaxConcurrentRuns int     `yaml:"max_concurrent_runs" json:"max_concurrent_runs"`
	BudgetUSD         float64 `yaml:"budget_usd" json:"budget_usd"`
}

// Combination is one expanded cell in the cartesian product.
type Combination struct {
	MatrixKey  string `json:"matrix_key"`
	PackRef    string `json:"pack_ref"`
	AgentRef   string `json:"agent_ref"`
	ModelRef   string `json:"model_ref,omitempty"`
	Repeat     int    `json:"repeat"`
	Seed       *int64 `json:"seed,omitempty"`
	CaseFanout bool   `json:"case_fanout"`
}

// ExpansionReport is the dry-run result.
type ExpansionReport struct {
	Name          string        `json:"name"`
	Combinations  []Combination `json:"combinations"`
	Count         int           `json:"count"`
	PackCount     int           `json:"pack_count"`
	AgentCount    int           `json:"agent_count"`
	ModelCount    int           `json:"model_count"`
	Repeats       int           `json:"repeats"`
	MaxConcurrent int           `json:"max_concurrent_runs,omitempty"`
	BudgetUSD     float64       `json:"budget_usd,omitempty"`
	CaseFanout    bool          `json:"case_fanout"`
}

// ParseManifest parses YAML or JSON bytes into a Manifest.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse evalset manifest: %w", err)
	}
	return m, nil
}

// Validate checks structural rules (does not resolve pack/deployment IDs).
func (m Manifest) Validate(maxCombos int) error {
	if maxCombos <= 0 {
		maxCombos = DefaultMaxCombos
	}
	if maxCombos > MaxAllowedCombos {
		return fmt.Errorf("max_combinations %d exceeds server limit %d", maxCombos, MaxAllowedCombos)
	}
	if strings.TrimSpace(m.Schema) == "" {
		return fmt.Errorf("schema is required (want %s)", SchemaV1)
	}
	if m.Schema != SchemaV1 {
		return fmt.Errorf("unsupported schema %q (want %s)", m.Schema, SchemaV1)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(m.Packs) == 0 {
		return fmt.Errorf("packs must contain at least one pack ref")
	}
	for i, p := range m.Packs {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("packs[%d] is empty", i)
		}
	}
	if len(m.Agents) == 0 {
		return fmt.Errorf("agents must contain at least one deployment entry")
	}
	for i, a := range m.Agents {
		if strings.TrimSpace(a.Deployment) == "" {
			return fmt.Errorf("agents[%d].deployment is required", i)
		}
	}
	repeats := m.Repeats
	if repeats == 0 {
		repeats = DefaultRepeats
	}
	if repeats < 1 {
		return fmt.Errorf("repeats must be >= 1")
	}
	modelCount := len(m.Models)
	if modelCount == 0 {
		modelCount = 1
	}
	total := len(m.Packs) * len(m.Agents) * modelCount * repeats
	if total > maxCombos {
		return fmt.Errorf("combination count %d exceeds max %d (packs=%d agents=%d models=%d repeats=%d)",
			total, maxCombos, len(m.Packs), len(m.Agents), modelCount, repeats)
	}
	if m.Seeds != nil {
		switch strings.ToLower(strings.TrimSpace(m.Seeds.Strategy)) {
		case "", "auto":
			if len(m.Seeds.Seeds) > 0 {
				return fmt.Errorf("seeds.seeds must be empty when strategy is auto")
			}
		case "explicit":
			if len(m.Seeds.Seeds) == 0 {
				return fmt.Errorf("seeds.seeds required when strategy is explicit")
			}
			if len(m.Seeds.Seeds) != repeats {
				return fmt.Errorf("seeds.seeds length %d must match repeats %d", len(m.Seeds.Seeds), repeats)
			}
			seen := map[int64]struct{}{}
			for i, s := range m.Seeds.Seeds {
				if s <= 0 {
					return fmt.Errorf("seeds.seeds[%d] must be a positive integer", i)
				}
				if _, ok := seen[s]; ok {
					return fmt.Errorf("duplicate seed %d", s)
				}
				seen[s] = struct{}{}
			}
		default:
			return fmt.Errorf("unsupported seeds.strategy %q (want auto or explicit)", m.Seeds.Strategy)
		}
	}
	if m.Limits.MaxConcurrentRuns < 0 {
		return fmt.Errorf("limits.max_concurrent_runs must be >= 0")
	}
	if m.Limits.BudgetUSD < 0 {
		return fmt.Errorf("limits.budget_usd must be >= 0")
	}
	return nil
}

// Expand returns the full cartesian product in stable order.
func (m Manifest) Expand(maxCombos int) (ExpansionReport, error) {
	if err := m.Validate(maxCombos); err != nil {
		return ExpansionReport{}, err
	}
	repeats := m.Repeats
	if repeats == 0 {
		repeats = DefaultRepeats
	}
	models := m.Models
	if len(models) == 0 {
		models = []string{""}
	}

	combos := make([]Combination, 0, len(m.Packs)*len(m.Agents)*len(models)*repeats)
	seenKeys := make(map[string]struct{}, len(m.Packs)*len(m.Agents)*len(models)*repeats)
	for _, pack := range m.Packs {
		packRef := strings.TrimSpace(pack)
		for _, agent := range m.Agents {
			agentRef := agentRef(agent)
			for _, model := range models {
				modelRef := strings.TrimSpace(model)
				for rep := 1; rep <= repeats; rep++ {
					key := matrixKey(packRef, agentRef, modelRef, rep)
					if _, dup := seenKeys[key]; dup {
						return ExpansionReport{}, fmt.Errorf("duplicate matrix_key %q (check agent labels and refs)", key)
					}
					seenKeys[key] = struct{}{}
					c := Combination{
						MatrixKey:  key,
						PackRef:    packRef,
						AgentRef:   agentRef,
						ModelRef:   modelRef,
						Repeat:     rep,
						CaseFanout: m.CaseFanout,
					}
					if seed := seedFor(m.Seeds, rep); seed != nil {
						c.Seed = seed
					}
					combos = append(combos, c)
				}
			}
		}
	}

	modelCount := len(m.Models)
	return ExpansionReport{
		Name:          strings.TrimSpace(m.Name),
		Combinations:  combos,
		Count:         len(combos),
		PackCount:     len(m.Packs),
		AgentCount:    len(m.Agents),
		ModelCount:    modelCount,
		Repeats:       repeats,
		MaxConcurrent: m.Limits.MaxConcurrentRuns,
		BudgetUSD:     m.Limits.BudgetUSD,
		CaseFanout:    m.CaseFanout,
	}, nil
}

func agentRef(a AgentEntry) string {
	if label := strings.TrimSpace(a.Label); label != "" {
		return sanitizeRef(label)
	}
	return sanitizeRef(strings.TrimSpace(a.Deployment))
}

func matrixKey(pack, agent, model string, rep int) string {
	parts := []string{sanitizeRef(pack), sanitizeRef(agent)}
	if model != "" {
		parts = append(parts, sanitizeRef(model))
	}
	parts = append(parts, fmt.Sprintf("%d", rep))
	return strings.Join(parts, "/")
}

func sanitizeRef(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func seedFor(cfg *SeedConfig, rep int) *int64 {
	if cfg == nil {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Strategy)) {
	case "explicit":
		if rep-1 < 0 || rep-1 >= len(cfg.Seeds) {
			return nil
		}
		s := cfg.Seeds[rep-1]
		return &s
	default:
		// auto: leave unset; session layer assigns later
		return nil
	}
}
