package scoring

import (
	"encoding/json"
	"math"
	"testing"
)

func TestValidateBLEUScore(t *testing.T) {
	tests := []struct {
		name        string
		actual      string
		expected    string
		config      string
		wantVerdict string
		minScore    float64
		maxScore    float64
	}{
		{
			name:        "exact_match_scores_one",
			actual:      "the cat is on the mat",
			expected:    "the cat is on the mat",
			config:      `{}`,
			wantVerdict: "pass",
			minScore:    1,
			maxScore:    1,
		},
		{
			name:        "no_overlap_scores_near_zero",
			actual:      "alpha beta gamma",
			expected:    "delta epsilon zeta",
			config:      `{}`,
			wantVerdict: "fail",
			minScore:    0,
			maxScore:    0,
		},
		{
			name:        "brevity_penalty_reduces_score",
			actual:      "the cat",
			expected:    "the cat is on the mat",
			config:      `{"threshold":0.05,"smoothing":"method1"}`,
			wantVerdict: "pass",
			minScore:    0.05,
			maxScore:    0.5,
		},
		{
			name:        "multi_reference_uses_best_reference",
			actual:      "the cat sits on the mat",
			expected:    `["a dog runs outside","the cat sits on the mat"]`,
			config:      `{}`,
			wantVerdict: "pass",
			minScore:    1,
			maxScore:    1,
		},
		{
			name:        "threshold_can_fail_partial_match",
			actual:      "the cat is here",
			expected:    "the dog is there",
			config:      `{"threshold":0.7,"smoothing":"method1"}`,
			wantVerdict: "fail",
			minScore:    0,
			maxScore:    0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome := validateBLEUScore(tt.actual, tt.expected, json.RawMessage(tt.config))
			if outcome.verdict != tt.wantVerdict {
				t.Fatalf("verdict = %q, want %q (reason: %s)", outcome.verdict, tt.wantVerdict, outcome.reason)
			}
			if outcome.normalizedScore == nil {
				t.Fatal("normalizedScore is nil")
			}
			score := *outcome.normalizedScore
			if score < tt.minScore || score > tt.maxScore {
				t.Fatalf("score = %f, want [%f, %f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestValidateROUGEScore(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		config   string
		minScore float64
		maxScore float64
	}{
		{
			name:     "rouge_1_exact_match",
			actual:   "the cat sat on the mat",
			expected: "the cat sat on the mat",
			config:   `{"variant":"rouge-1"}`,
			minScore: 1,
			maxScore: 1,
		},
		{
			name:     "rouge_2_low_overlap",
			actual:   "alpha beta gamma",
			expected: "alpha zeta theta",
			config:   `{"variant":"rouge-2"}`,
			minScore: 0,
			maxScore: 0.4,
		},
		{
			name:     "rouge_l_partial_sequence_match",
			actual:   "the cat sat on the mat",
			expected: "the cat slept on the rug",
			config:   `{"variant":"rouge-l","threshold":0.4}`,
			minScore: 0.4,
			maxScore: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome := validateROUGEScore(tt.actual, tt.expected, json.RawMessage(tt.config))
			if outcome.normalizedScore == nil {
				t.Fatal("normalizedScore is nil")
			}
			score := *outcome.normalizedScore
			if score < tt.minScore || score > tt.maxScore {
				t.Fatalf("score = %f, want [%f, %f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestValidateChrFScore(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		config   string
		minScore float64
		maxScore float64
	}{
		{
			name:     "exact_match_scores_one",
			actual:   "kitten",
			expected: "kitten",
			config:   `{}`,
			minScore: 1,
			maxScore: 1,
		},
		{
			name:     "unicode_text_supported",
			actual:   "こんにちは世界",
			expected: "こんにちは世界",
			config:   `{}`,
			minScore: 1,
			maxScore: 1,
		},
		{
			name:     "low_overlap_scores_low",
			actual:   "abcdef",
			expected: "uvwxyz",
			config:   `{}`,
			minScore: 0,
			maxScore: 0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome := validateChrFScore(tt.actual, tt.expected, json.RawMessage(tt.config))
			if outcome.normalizedScore == nil {
				t.Fatal("normalizedScore is nil")
			}
			score := *outcome.normalizedScore
			if score < tt.minScore || score > tt.maxScore {
				t.Fatalf("score = %f, want [%f, %f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestParseBLEUScoreConfig(t *testing.T) {
	cfg, err := parseBLEUScoreConfig(nil)
	if err != nil {
		t.Fatalf("parseBLEUScoreConfig(nil) error = %v", err)
	}
	if cfg.MaxNGram == nil || *cfg.MaxNGram != 4 || cfg.Smoothing != bleuSmoothingNone {
		t.Fatalf("default cfg = %+v, want max_ngram=4 smoothing=none", cfg)
	}

	if _, err := parseBLEUScoreConfig(json.RawMessage(`{bad json`)); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestParseROUGEScoreConfig(t *testing.T) {
	cfg, err := parseROUGEScoreConfig(nil)
	if err != nil {
		t.Fatalf("parseROUGEScoreConfig(nil) error = %v", err)
	}
	if cfg.Variant != rougeVariantL {
		t.Fatalf("default variant = %q, want %q", cfg.Variant, rougeVariantL)
	}
}

func TestParseChrFScoreConfig(t *testing.T) {
	cfg, err := parseChrFScoreConfig(nil)
	if err != nil {
		t.Fatalf("parseChrFScoreConfig(nil) error = %v", err)
	}
	if cfg.CharOrder == nil || *cfg.CharOrder != 6 {
		t.Fatalf("default char_order = %+v, want 6", cfg.CharOrder)
	}
}

func TestParseGenerationReferencesRejectsInvalidJSONArray(t *testing.T) {
	_, err := parseGenerationReferences(`[1,2,3]`, false, false)
	if err == nil {
		t.Fatal("expected JSON array type error")
	}
}

func TestFScore(t *testing.T) {
	got := fScore(0.5, 0.5, 1)
	if math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("fScore = %f, want 0.5", got)
	}
}
