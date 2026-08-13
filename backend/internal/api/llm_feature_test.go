package api

import (
	"strings"
	"testing"
)

// decodeStrictJSONObject is shared by every synchronous LLM feature, so its
// contract is pinned against each feature's own output type rather than against
// one of them: a reply that is not a JSON object must never be reported as a
// successful decode.
//
// `null` is the case that matters. encoding/json decodes it into any struct as
// the zero value, with no error and a clean trailing EOF, so without an
// explicit object check a model that answered "null" reads as a model that
// answered correctly and returned nothing.
func TestDecodeStrictJSONObjectRejectsNonObjects(t *testing.T) {
	for _, raw := range []string{
		"null",
		"  null  ",
		"```json\nnull\n```",
		"[]",
		`"a bare string"`,
		"42",
		"true",
	} {
		t.Run(strings.TrimSpace(raw), func(t *testing.T) {
			var blueprint generatedPackBlueprint
			if err := decodeStrictJSONObject(raw, &blueprint); err == nil {
				t.Errorf("decodeStrictJSONObject(%q) into a pack blueprint returned no error", raw)
			}
			var insights runRankingInsightsResponse
			if err := decodeStrictJSONObject(raw, &insights); err == nil {
				t.Errorf("decodeStrictJSONObject(%q) into ranking insights returned no error", raw)
			}
		})
	}
}

// An empty object is a JSON object and must still decode. Rejecting it here
// would move a content decision into the transport layer: each feature's own
// validation already rejects a reply with no validators or no winner, and with
// a message that names what was missing.
func TestDecodeStrictJSONObjectAcceptsEmptyObject(t *testing.T) {
	var blueprint generatedPackBlueprint
	if err := decodeStrictJSONObject("{}", &blueprint); err != nil {
		t.Fatalf("decodeStrictJSONObject(\"{}\"): %v", err)
	}
	if len(blueprint.Validators) != 0 {
		t.Fatalf("blueprint = %+v, want the zero value", blueprint)
	}

	// And it still fails the feature's own content validation.
	if _, err := generatedPackDraftBundle("{}", generateTestModel, generateTestProviderAccountID); err == nil {
		t.Fatal("an empty blueprint produced a pack, want a validation error")
	}
}
