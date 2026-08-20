package posthog

import "testing"

func TestAnalyticsRequired(t *testing.T) {
	for _, value := range []string{"1", "true", "YES", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ANALYTICS_REQUIRED", value)
			if !AnalyticsRequired() {
				t.Fatalf("AnalyticsRequired() = false for %q", value)
			}
		})
	}
	t.Setenv("ANALYTICS_REQUIRED", "false")
	if AnalyticsRequired() {
		t.Fatal("AnalyticsRequired() = true for false")
	}
}
