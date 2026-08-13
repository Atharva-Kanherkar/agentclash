package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agentclash/agentclash/runtime/challengepack"
	"github.com/google/uuid"
)

// Shared ground for the producers of a challenge-pack draft: promoting an agent
// tryout (agent_tryout_pack_composition.go) and generating a pack from a
// plain-English description (challenge_pack_generate.go). Both assemble a
// challengepack.Bundle and both must land a draft the visual builder can
// actually compile, so the "does this compile?" guarantee lives here once
// rather than once per producer.

// packSlugIDLength is how much of a source id is folded into a derived pack
// slug so two drafts from the same source do not collide on publish.
const packSlugIDLength = 8

// packBundleCompileError reports why a bundle would fail the path the builder
// actually runs on compile: BundleToComposition -> ComposeBundle ->
// ValidateBundle. Validating the raw bundle instead would report false
// failures, because ComposeBundle fills in fields producers deliberately leave
// unset (judge_mode is inferred from what the spec declares). Returns nil when
// the bundle compiles.
func packBundleCompileError(bundle challengepack.Bundle) error {
	composition, err := challengepack.BundleToComposition(bundle)
	if err != nil {
		return fmt.Errorf("build composition: %w", err)
	}
	composed, err := challengepack.ComposeBundle(composition, challengepack.ResolvedPieces{})
	if err != nil {
		return fmt.Errorf("compose bundle: %w", err)
	}
	return challengepack.ValidateBundle(composed)
}

// packBundlePublishable is the boolean form of packBundleCompileError, for
// producers that pick between candidate bundles rather than reporting why.
func packBundlePublishable(bundle challengepack.Bundle) bool {
	return packBundleCompileError(bundle) == nil
}

// packBundleComposition renders the builder's working document for a bundle:
// the composition JSON stored verbatim in challenge_pack_drafts.composition.
func packBundleComposition(bundle challengepack.Bundle) (json.RawMessage, error) {
	composition, err := challengepack.BundleToComposition(bundle)
	if err != nil {
		return nil, fmt.Errorf("build composition from bundle: %w", err)
	}
	encoded, err := json.Marshal(composition)
	if err != nil {
		return nil, fmt.Errorf("marshal composition: %w", err)
	}
	return encoded, nil
}

// packSlugWithSuffix keeps derived packs from colliding: two drafts built from
// the same template or the same description would otherwise both try to
// publish version 1 of one pack slug.
func packSlugWithSuffix(prefix, key string, id uuid.UUID) string {
	suffix := strings.ReplaceAll(id.String(), "-", "")
	if len(suffix) > packSlugIDLength {
		suffix = suffix[:packSlugIDLength]
	}
	if suffix == "" {
		return prefix + "-" + key
	}
	return prefix + "-" + key + "-" + suffix
}

// packSlugify reduces a value to lowercase alphanumerics plus single dashes so
// it is safe as a pack slug, challenge key, or validator key.
func packSlugify(value, fallback string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case r == '_':
			b.WriteRune('_')
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return fallback
	}
	return slug
}

// optionalTrimmedString returns nil for a blank value so optional pack metadata
// stays absent rather than empty.
func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
