package observability

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAlertRulesFilePresent(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// backend/internal/observability -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../.."))
	path := filepath.Join(root, "deploy/observability/prometheus-alerts.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, needle := range []string{
		"FleetEvalSetStalled",
		"FleetSandboxAcquireSlow",
		"FleetProviderCooldownSustained",
		"FleetEventQueueDepthGrowing",
		"FleetTemporalWorkerSlotExhaustion",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("alerts missing %q", needle)
		}
	}
}
