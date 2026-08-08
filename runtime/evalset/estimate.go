package evalset

import (
	"strings"

	"github.com/agentclash/agentclash/runtime/provider"
)

// Cost estimate heuristics (Fleet 13). These are intentionally coarse: expand
// does not resolve pack case catalogs. Documented tolerance for estimate-vs-actual
// acceptance is ±50% when packs average ~DefaultCasesPerPack cases and models
// match StaticModelPrice entries.
const (
	DefaultCasesPerPack     = 10
	DefaultInputTokensCase  = 2000
	DefaultOutputTokensCase = 500
	// Fallback when no model price is known (approx mid-tier chat).
	DefaultInPerMillion  = 3.0
	DefaultOutPerMillion = 15.0
)

// CostEstimate is a pre-flight USD roll-up returned by expand / create.
type CostEstimate struct {
	EstimatedUSD       float64 `json:"estimated_usd"`
	Combinations       int     `json:"combinations"`
	AssumedCasesPerPack int    `json:"assumed_cases_per_pack"`
	PerCaseUSD         float64 `json:"per_case_usd"`
	Heuristic          string  `json:"heuristic"`
	TolerancePct       int     `json:"tolerance_pct"`
	BudgetUSD          *float64 `json:"budget_usd,omitempty"`
	ExceedsBudget      bool    `json:"exceeds_budget"`
}

// EstimateCost computes a heuristic USD estimate for an expansion report.
// modelHints are optional provider/model pairs ("anthropic/claude-haiku-4"); the
// cheapest known price among hints is used, else defaults.
func EstimateCost(report ExpansionReport, budgetUSD *float64, modelHints []string) CostEstimate {
	in, out := DefaultInPerMillion, DefaultOutPerMillion
	for _, hint := range modelHints {
		hint = strings.TrimSpace(hint)
		if hint == "" {
			continue
		}
		providerKey, modelID := splitProviderModel(hint)
		if pin, pout, ok := provider.StaticModelPrice(providerKey, modelID); ok {
			// Prefer the cheapest hint for a conservative "can I afford this?" warn.
			caseCost := tokenCost(pin, pout)
			if caseCost < tokenCost(in, out) {
				in, out = pin, pout
			}
		}
	}
	perCase := tokenCost(in, out)
	cases := report.Count * DefaultCasesPerPack
	if cases < 0 {
		cases = 0
	}
	est := float64(cases) * perCase
	outEst := CostEstimate{
		EstimatedUSD:        round2(est),
		Combinations:        report.Count,
		AssumedCasesPerPack: DefaultCasesPerPack,
		PerCaseUSD:          round4(perCase),
		Heuristic:           "combinations × assumed_cases_per_pack × (2k_in+500_out) × static_model_price",
		TolerancePct:        50,
		BudgetUSD:           budgetUSD,
	}
	if budgetUSD != nil && *budgetUSD >= 0 && outEst.EstimatedUSD > *budgetUSD {
		outEst.ExceedsBudget = true
	}
	return outEst
}

func tokenCost(inPerM, outPerM float64) float64 {
	return (float64(DefaultInputTokensCase)/1e6)*inPerM + (float64(DefaultOutputTokensCase)/1e6)*outPerM
}

func splitProviderModel(hint string) (providerKey, modelID string) {
	parts := strings.SplitN(hint, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "anthropic", hint
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
func round4(v float64) float64 { return float64(int(v*10000+0.5)) / 10000 }
