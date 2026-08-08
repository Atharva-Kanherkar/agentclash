package scanners

import (
	"regexp"
	"strings"
)

// RunPattern evaluates a pattern scanner against transcript text.
func RunPattern(def Definition, transcript string) ([]Finding, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	if def.Kind != KindPattern {
		return nil, nil
	}
	text := transcript
	findings := make([]Finding, 0)
	for _, pattern := range def.Pattern.AnyMatch {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, err
		}
		loc := re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		quote := text[loc[0]:loc[1]]
		// Expand evidence window slightly.
		start := loc[0] - 40
		if start < 0 {
			start = 0
		}
		end := loc[1] + 40
		if end > len(text) {
			end = len(text)
		}
		evidence := strings.TrimSpace(text[start:end])
		findings = append(findings, Finding{
			Scanner:     def.Name,
			Version:     def.Version,
			Severity:    def.Severity,
			Category:    def.Category,
			Evidence:    evidence,
			Confidence:  1.0,
			MatchedRule: quote,
		})
		break // one finding per scanner per case for v1
	}
	return findings, nil
}
