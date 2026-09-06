package vibe

import (
	"math/rand"
	"sort"
	"testing"
)

func TestTrialEconomicsAcrossPilotModels(t *testing.T) {
	// Synthetic journeys, not observed production percentiles. Every role's
	// input includes its complete assembled context. No cache discount assumed.
	prices := []struct {
		name    string
		in, out float64
	}{{"4o-mini", .15, .60}, {"4.1-mini", .40, 1.60}, {"4.1", 2, 8}}
	for _, assistant := range prices {
		for _, target := range prices {
			rng := rand.New(rand.NewSource(6072026))
			full := []float64{}
			first := []float64{}
			for journey := 0; journey < 9000; journey++ {
				helpers := []int{6, 9, 15}[journey%3]
				before := []int{3, 5, 8}[journey%3]
				cost := 0.0
				for call := 0; call < helpers; call++ {
					in := 2400 + rng.Float64()*7000
					out := 180 + rng.Float64()*1320
					cost += (in*assistant.in + out*assistant.out) / 1e6
					if call == before-1 {
						first = append(first, cost)
					}
				}
				for call := 0; call < 6; call++ {
					in := 1800 + rng.Float64()*7200
					out := 150 + rng.Float64()*1350
					targetCost := (in*target.in + out*target.out) / 1e6
					judgeCost := ((2500+rng.Float64()*8500)*.4 + (100+rng.Float64()*500)*1.6) / 1e6
					cost += targetCost + judgeCost
					if call < 3 {
						first[len(first)-1] += targetCost + judgeCost
					}
				}
				full = append(full, cost)
			}
			sort.Float64s(full)
			sort.Float64s(first)
			t.Logf("assistant=%s target=%s first P50=$%.4f P90=$%.4f complete P50=$%.4f P90=$%.4f", assistant.name, target.name, first[4500], first[8100], full[4500], full[8100])
			if full[8100] >= 1 {
				t.Fatal("P90 cannot reach check and retest within trial")
			}
		}
	}
	// Worst admitted protected flow: two authoring calls, three target + judge
	// cases, one identical retest. Exploratory messages cannot consume this pool.
	high := ModelProfile{InputNanoPerToken: 2000, OutputNanoPerToken: 8000}
	judge := ModelProfile{InputNanoPerToken: 400, OutputNanoPerToken: 1600}
	h, _ := high.BoundCost(16384, 2048)
	j, _ := judge.BoundCost(16384, 2048)
	if 8*h+6*j > TrialBudget-TrialExploreBudget {
		t.Fatal("protected trial allocation cannot fund the intended scorecard journey")
	}
}
