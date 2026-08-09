# Fleet observability (Fleet 14)

Prometheus scrape endpoint and alert/dashboard pack for the execution plane.

## Enable metrics

```bash
export METRICS_ENABLED=true
export METRICS_ADDR=:9464          # scrape http://<pod>:9464/metrics
export FLEET_STALL_THRESHOLD=30m   # stalled eval-set detector
export FLEET_STALL_INTERVAL=5m
export SSE_MAX_CONNECTIONS=0       # 0 = unlimited; >0 returns 503 when full
```

Default is **off** (`METRICS_ENABLED` unset/false) — no scrape listener, no process overhead beyond noop Fleet helpers.

## Artifacts

| File | Purpose |
|------|---------|
| `prometheus-alerts.yaml` | Alert rules (stalled sets, sandbox p95, provider cooldown, event queue, worker slots) |
| `grafana-fleet-dashboard.json` | Fleet overview dashboard |

Validate alerts when `promtool` is available:

```bash
promtool check rules deploy/observability/prometheus-alerts.yaml
```

The Fleet 12 Helm chart should expose a ServiceMonitor toggle that scrapes `METRICS_ADDR` on api-server and worker pods.

## Manual checks (kind / load)

1. Start worker+api with `METRICS_ENABLED=true`, curl `/metrics`, confirm `fleet_*` and `temporal_*` families.
2. Stall test: leave an eval set `running` with no child transitions past `FLEET_STALL_THRESHOLD`; expect `fleet_set_stalled` and `fleet.set.stalled` slog.
3. Temporal slot saturation: lower `WORKER_MAX_CONCURRENT_ACTIVITIES`, flood runs, watch Temporal worker slot gauges under load.
