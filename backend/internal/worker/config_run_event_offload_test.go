package worker

import (
	"testing"
)

func TestLoadConfigFromEnvRunEventInlineMaxBytesZeroDisables(t *testing.T) {
	unsetEnv(t, "AGENTCLASH_SECRETS_MASTER_KEY")
	t.Setenv("APP_ENV", "development")
	t.Setenv("RUN_EVENT_INLINE_MAX_BYTES", "0")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RunEventInlineMaxBytes != 0 {
		t.Fatalf("got %d", cfg.RunEventInlineMaxBytes)
	}
}
