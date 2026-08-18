# Prometheus

Prometheus is an open-source metrics collection and alerting system that solves the problem of knowing what your infrastructure and services are actually doing right now — and being paged the moment something goes wrong — without every team hand-rolling its own polling, storage, and alerting pipeline.

## Primary use cases and when to adopt it

- **Service and infrastructure metrics.** Prometheus pulls (scrapes) time-series metrics — request rates, error counts, latency histograms, CPU/memory, queue depth — from every service and node on a schedule, giving you a single, queryable source of truth for "what is the system doing."
- **Kubernetes-native monitoring.** Prometheus is the de facto metrics backbone of the Kubernetes ecosystem; kube-state-metrics, cAdvisor, and most Helm charts ship Prometheus exporters out of the box, and the Prometheus Operator lets you declare scrape targets and alert rules as Kubernetes CRDs alongside your workloads.
- **Alerting on symptoms, not just failures.** Via Alertmanager, teams define rules against PromQL expressions (e.g., error rate over 5% for 5 minutes) so on-call engineers get paged on the leading indicators of an outage rather than only after something is fully down.
- **Powering dashboards.** Prometheus is rarely used alone for visualization — it's almost always paired with Grafana, which queries Prometheus as a data source to render the graphs and SLO dashboards teams actually look at.

A team typically adopts Prometheus once they have more than a handful of services and ad-hoc `top`/log-grepping or vendor dashboards stop being enough to answer "is this healthy" quickly, or as soon as they run Kubernetes, where it's the default expectation. It's a natural pairing with OpenTelemetry: OTel handles vendor-neutral instrumentation and export, while Prometheus (or an OTLP-compatible backend) is one of the places that data lands and gets queried.

## Basic usage examples

**1. Run Prometheus locally with a minimal scrape config:**
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'my-service'
    static_configs:
      - targets: ['localhost:8080']
```
```bash
docker run -p 9090:9090 -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```
This scrapes `localhost:8080/metrics` every 15 seconds and serves a query UI at `http://localhost:9090`.

**2. Expose metrics from your app (Python example):**
```python
from prometheus_client import start_http_server, Counter

REQUESTS = Counter('http_requests_total', 'Total HTTP requests', ['endpoint'])

@app.route('/orders')
def orders():
    REQUESTS.labels(endpoint='/orders').inc()
    ...

start_http_server(8080)  # exposes /metrics for Prometheus to scrape
```

**3. Query with PromQL:**
```promql
# Per-second request rate over the last 5 minutes, by endpoint
rate(http_requests_total[5m])

# 95th percentile latency from a histogram metric
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

## Common pitfalls

- **Cardinality explosions.** Adding a label with unbounded values (user ID, raw URL path, request ID) multiplies the number of time series Prometheus has to store and can bring a server to its knees with memory pressure — always keep label values to a bounded, low-cardinality set.
- **Pull model means Prometheus must reach your targets.** Short-lived jobs (batch, serverless, CI runs) don't live long enough to be scraped; use the Pushgateway for those instead of expecting normal scraping to catch them.
- **Local storage doesn't scale to long retention by default.** Prometheus's on-disk TSDB is optimized for recent data (typically 15-30 days); for long-term retention or multi-cluster/global querying, teams pair it with remote-write to Thanos, Cortex, or Mimir rather than growing a single instance's local disk indefinitely.
- **Counters reset on restart.** `Counter` metrics reset to zero when a process restarts, which is why you almost always wrap them in `rate()` or `increase()` in PromQL rather than reading raw counter values — Prometheus handles the reset detection for you, but only if you query it that way.
- **Alerting on raw thresholds instead of durations.** A rule that fires the instant a value crosses a threshold produces noisy, flappy pages; use a `for:` duration on alert rules so a metric has to stay past the threshold for a sustained window before anyone gets paged.
