package observability

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMetricsDisabledNoServer(t *testing.T) {
	rt, err := Start(context.Background(), Config{Enabled: false}, slog.Default(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(context.Background())
	if rt.server != nil {
		t.Fatal("expected no scrape server when disabled")
	}
}

func TestMetricsEnabledExposesFleetFamilies(t *testing.T) {
	rt, err := Start(context.Background(), Config{
		Enabled: true,
		Addr:    "127.0.0.1:0",
	}, slog.Default(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close(context.Background())

	rt.Fleet().RecordSetStalled(context.Background())
	rt.Fleet().RecordProviderRequest(context.Background(), "openai")

	deadline := time.Now().Add(2 * time.Second)
	var body string
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + rt.ScrapeAddr() + "/metrics")
		if err != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		body = string(raw)
		break
	}
	if body == "" {
		t.Fatal("failed to scrape /metrics")
	}
	for _, name := range []string{"fleet_set_stalled", "fleet_provider_requests"} {
		if !strings.Contains(body, name) {
			t.Fatalf("metrics missing %q:\n%s", name, body)
		}
	}
}
