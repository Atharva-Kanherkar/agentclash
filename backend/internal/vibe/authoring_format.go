package vibe

import "encoding/json"

const (
	MaxProposedRequirements = 3
	MaxProposedAssumptions  = 2
)

// Schema support is operator-verified per endpoint, not inferred from a model
// name or a successful JSON-object response. A schema rejection does not cause
// a hidden downgrade/provider retry. Decode and validate remain authoritative.
func authoringFormat(p ModelProfile) json.RawMessage {
	if !p.StructuredOutputs {
		return jsonFormat
	}
	return strictAuthoringFormat
}

var strictAuthoringFormat = raw(map[string]any{
	"type": "json_schema",
	"json_schema": map[string]any{
		"name": "vibe_reply", "strict": true,
		"schema": map[string]any{
			"type": "object", "additionalProperties": false,
			"required": []string{"reply", "proposed_requirements", "assumptions", "draft"},
			"properties": map[string]any{
				"reply":                 map[string]any{"type": "string"},
				"proposed_requirements": stringListSchema(MaxProposedRequirements),
				"assumptions":           stringListSchema(MaxProposedAssumptions),
				"draft": map[string]any{"anyOf": []any{
					map[string]any{"type": "null"},
					map[string]any{
						"type": "object", "additionalProperties": false,
						"required": []string{"title", "agent_prompt", "examples", "success_criteria"},
						"properties": map[string]any{
							"title":            map[string]any{"type": "string"},
							"agent_prompt":     map[string]any{"type": "string"},
							"examples":         stringListSchema(3),
							"success_criteria": map[string]any{"type": "string"},
						},
					},
				}},
			},
		},
	},
})

func stringListSchema(max int) map[string]any {
	return map[string]any{"type": "array", "maxItems": max, "items": map[string]any{"type": "string"}}
}
