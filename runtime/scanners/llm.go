package scanners

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseLLMVerdict validates and parses structured LLM scanner output.
// Malformed payloads are rejected (never stored raw).
func ParseLLMVerdict(raw []byte, expectedSchema int) (LLMVerdict, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return LLMVerdict{}, fmt.Errorf("empty llm scanner verdict")
	}
	// Strip optional markdown fences.
	s := string(raw)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		raw = []byte(strings.TrimSpace(s))
	}
	var v LLMVerdict
	if err := json.Unmarshal(raw, &v); err != nil {
		return LLMVerdict{}, fmt.Errorf("llm scanner verdict is not JSON: %w", err)
	}
	if expectedSchema <= 0 {
		expectedSchema = 1
	}
	if v.SchemaVersion != expectedSchema {
		return LLMVerdict{}, fmt.Errorf("llm scanner schema_version=%d want %d", v.SchemaVersion, expectedSchema)
	}
	if v.Hit {
		if strings.TrimSpace(v.Evidence) == "" {
			return LLMVerdict{}, fmt.Errorf("llm scanner hit requires evidence")
		}
		if v.Confidence < 0 || v.Confidence > 1 {
			return LLMVerdict{}, fmt.Errorf("llm scanner confidence out of range")
		}
		switch v.Severity {
		case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		default:
			return LLMVerdict{}, fmt.Errorf("llm scanner severity %q invalid", v.Severity)
		}
	}
	return v, nil
}

func FindingFromLLM(def Definition, v LLMVerdict) *Finding {
	if !v.Hit {
		return nil
	}
	sev := v.Severity
	if sev == "" {
		sev = def.Severity
	}
	cat := v.Category
	if cat == "" {
		cat = def.Category
	}
	return &Finding{
		Scanner:    def.Name,
		Version:    def.Version,
		Severity:   sev,
		Category:   cat,
		Evidence:   v.Evidence,
		Confidence: v.Confidence,
	}
}
