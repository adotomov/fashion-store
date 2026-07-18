# Observability

The API is instrumented for Google Cloud's operations suite only — no paid
aggregator. Three signals:

| Signal | Backend | Enabled by |
|---|---|---|
| Structured logs + trace correlation | Cloud Logging | always on |
| Distributed traces | Cloud Trace | `OTEL_TRACES_ENABLED=true` |
| Custom business metrics | Cloud Monitoring | `OTEL_METRICS_ENABLED=true` |

Cloud Run auto-ingests stdout to Cloud Logging, so logging needs no exporter.
Traces and metrics are pushed via OpenTelemetry and are **off by default** so
local/devbox runs need no GCP credentials.

## Configuration

Set via env (Cloud Run env is managed in `infra/terraform/envs/<env>/cloud_run.tf`):

| Env var | Default | Meaning |
|---|---|---|
| `LOG_LEVEL` | `info` | `debug`\|`info`\|`warn`\|`error` |
| `LOG_FORMAT` | `json` | `json` (GCP schema) or `text` (local, human-readable) |
| `GCP_PROJECT_ID` | `STORAGE_PROJECT_ID` | project for trace correlation + metric/span export |
| `OTEL_TRACES_ENABLED` | `false` | export spans to Cloud Trace |
| `OTEL_METRICS_ENABLED` | `false` | export metrics to Cloud Monitoring |
| `OTEL_TRACE_SAMPLE_RATIO` | `0.1` | parent-based root-span sampling ratio |
| `OTEL_METRIC_EXPORT_INTERVAL` | `60s` | how often metrics are pushed |

In Terraform, the OTel exporters are gated by `var.observability_enabled`
(dev defaults to `true`, prod to `false`). `var.otel_trace_sample_ratio` and
`var.alert_email` tune sampling and alert notifications.

## Logging

`internal/platform/logger` installs a slog handler that emits Cloud Logging's
schema: slog `level`→`severity`, `msg`→`message`, `time`→`timestamp`, source →
`logging.googleapis.com/sourceLocation`. Every record is annotated from
request-scoped context attributes seeded by middleware:

- `request_id` — from `X-Request-Id` or generated (echoed in the response).
- `logging.googleapis.com/trace` / `spanId` — from the active OTel span, else
  Cloud Run's `X-Cloud-Trace-Context` header. This is what groups a request's
  log lines under its trace in Logs Explorer.
- `method`, `path`.

Service code logs with the `*Context` slog methods (`log.ErrorContext(ctx, …)`)
so those attributes are attached automatically — no manual threading. Standard
attribute keys: `order_id`, `order_number`, `user_id`, `reservation_id`,
`code_id`, provider ids. **Never log secrets or raw PII** (card data is
tokenized by Revolut; avoid logging raw email/phone).

### Querying logs (Logs Explorer)

```
resource.type="cloud_run_revision"
resource.labels.service_name="api"
severity>=ERROR
```

Follow one request: click a log line → **Show entries for this trace**, or
filter `trace="projects/<PROJECT_ID>/traces/<TRACE_ID>"`.

## Tracing

`internal/platform/telemetry` wires an OTel `TracerProvider` with the Cloud
Trace exporter, GCP resource detection (Cloud Run service/revision), and a
parent-based ratio sampler. Instrumentation:

- **Inbound HTTP** — `otelhttp` wraps the router (server span per request).
- **Database** — `otelpgx` traces every pgx query.
- **Outbound HTTP** — Revolut and Speedy clients use an `otelhttp` transport.

A sampled checkout request shows a waterfall: `HTTP POST` → pgx spans → the
Revolut client span. Open **Trace Explorer** and filter by service `api`.

## Metrics

`internal/platform/metrics` defines a small, low-cardinality set of business
counters (all attributes are bounded enums). HTTP RED metrics (request count /
latency / errors) are intentionally left to Cloud Run's **free built-in**
metrics, so nothing high-cardinality is pushed.

| Metric | Attributes |
|---|---|
| `orders_placed_total` | `payment_method` |
| `payments_initiated_total` | — |
| `payments_succeeded_total` | — |
| `payments_failed_total` | `reason` |
| `refunds_total` | `status` |
| `checkout_reservation_conflicts_total` | — |
| `checkout_sweeper_reclaims_total` | `kind` |
| `webhook_events_total` | `type`, `result` |
| `fulfillment_poll_errors_total` | — |

In **Metrics Explorer** they appear under `custom.googleapis.com/…` (or
`workload.googleapis.com/…`) once metrics are enabled and the first batch has
exported (up to ~1–2 min).

> **Cardinality rule:** never add a per-user, per-order-id or otherwise
> unbounded attribute — it multiplies billable time series. Keep every label a
> small, fixed enum.

## Alerts (`monitoring.tf`)

Terraform provisions, per env:

- **Uptime check** on `https://<api_subdomain>.<domain_root>/healthz` + an alert
  when it fails.
- **API 5xx** alert (Cloud Run `request_count`, `response_code_class=5xx`).
- **Error-log spike** alert (log-based metric counting `severity>=ERROR`).

Set `alert_email` (tfvars) to attach an email notification channel; without it
the policies are created but view-only.

### Runbook

- **Uptime failing** — check `readyz` (it pings the DB): DB down or the service
  is not serving. Inspect recent revisions / Cloud SQL.
- **5xx spike** — filter logs to `severity>=ERROR` for the alerting window;
  `panic recovered` indicates a code bug, otherwise follow the failing trace.
- **Error-log spike** — often `revolut payment amount mismatch`, `webhook: …
  failed`, or invoice/cart cleanup failures. Use the `order_id` / `order_number`
  on the log line to trace the specific order.

### Health endpoints

- `GET /healthz` — liveness (200 while the process serves).
- `GET /readyz` — readiness; pings the DB and returns **503** when it is
  unreachable, so the load balancer and uptime check drain a broken instance.

## Follow-ups (add once metrics are flowing)

Custom-metric alert types don't exist until the first export, so they're not in
`monitoring.tf` yet. After enabling metrics in an env, consider adding alert
policies on `payments_failed_total` (failure-rate spike) and
`fulfillment_poll_errors_total`. Validate the first-week Cloud Monitoring bill
against the estimate in the plan before enabling metrics in prod.
