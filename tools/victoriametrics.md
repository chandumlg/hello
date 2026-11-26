# VictoriaMetrics

VictoriaMetrics is a fast, cost-efficient, Prometheus-compatible time-series database and monitoring stack that stores far more metrics on far less RAM and disk than vanilla Prometheus, while still speaking PromQL, remote-write, and the Prometheus HTTP API.

## What problem it solves

Prometheus is excellent at scraping and alerting, but it is a single-node, local-disk TSDB with short default retention. Once you have millions of active series, months of retention, or many clusters to aggregate, teams usually bolt on Thanos, Cortex, or Mimir — which bring object storage, many microservices, and real operational overhead. VictoriaMetrics takes a different approach: a small number of statically linked Go binaries with aggressive compression (often ~0.4–1 byte per sample) and low memory use, scaling from a single process to a sharded cluster without object storage.

## Primary use cases and when to adopt it

- **Long-term storage for Prometheus.** Point `remote_write` at VictoriaMetrics and keep months or years of data, queryable from Grafana as a normal Prometheus datasource.
- **Global view across clusters.** Every Kubernetes cluster ships metrics (via Prometheus or `vmagent`) into one central store, so you query fleet-wide without federation.
- **High-cardinality / high-ingest workloads.** Teams hitting Prometheus OOMs, or paying a lot for hosted metrics, often move here for the resource efficiency.
- **Lightweight agent at the edge.** `vmagent` is a drop-in scraper/forwarder with on-disk buffering, relabeling, and fan-out to multiple remote-write targets — useful even if your backend isn't VictoriaMetrics.
- **Simpler alternative to Thanos/Mimir.** If you don't need object-storage-backed, effectively unlimited retention and want fewer moving parts, single-node VictoriaMetrics handles surprisingly large workloads (commonly tens of millions of active series on one big box).

Components you will meet:

| Component | Role |
|---|---|
| `victoria-metrics` (single-node) | Ingest + store + query in one binary (port 8428) |
| `vminsert` / `vmstorage` / `vmselect` | Cluster version: stateless ingest, stateful sharded storage, stateless query |
| `vmagent` | Scraping, relabeling, buffering, remote-write fan-out |
| `vmalert` | Evaluates Prometheus-style recording and alerting rules, sends to Alertmanager |
| `vmui` | Built-in query UI, cardinality explorer, top-queries view (`/vmui`) |
| `vmctl` / `vmbackup` / `vmrestore` | Migration from Prometheus/Thanos/InfluxDB, snapshot-based backup and restore |

## Getting started

### 1. Run single-node VictoriaMetrics and push data into it

```bash
docker run -d --name vm -p 8428:8428 \
  -v vm-data:/victoria-metrics-data \
  victoriametrics/victoria-metrics:latest \
  -storageDataPath=/victoria-metrics-data \
  -retentionPeriod=6        # months; default is only 1 month

# Ingest a sample using the Prometheus text exposition format
curl -d 'http_requests_total{service="api",code="200"} 42' \
  http://localhost:8428/api/v1/import/prometheus

# Query it back through the Prometheus-compatible API
curl 'http://localhost:8428/api/v1/query?query=http_requests_total'

# Or open the UI
open http://localhost:8428/vmui
```

To use it as long-term storage for an existing Prometheus, add to `prometheus.yml`:

```yaml
remote_write:
  - url: http://victoriametrics:8428/api/v1/write
```

In Grafana, add a **Prometheus** datasource with URL `http://victoriametrics:8428` — dashboards work unchanged.

### 2. Replace Prometheus scraping with `vmagent`

`vmagent` reads standard Prometheus `scrape_configs`, so you can reuse your existing config:

```bash
vmagent \
  -promscrape.config=/etc/vmagent/scrape.yml \
  -remoteWrite.url=http://victoriametrics:8428/api/v1/write \
  -remoteWrite.url=http://backup-region-vm:8428/api/v1/write \
  -remoteWrite.tmpDataPath=/var/lib/vmagent   # on-disk buffer when a target is down
```

```yaml
# /etc/vmagent/scrape.yml
global:
  scrape_interval: 30s
scrape_configs:
  - job_name: node
    static_configs:
      - targets: ["node-exporter:9100"]
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: "node_scrape_collector_.*"
        action: drop            # drop noisy series before they cost you storage
```

On Kubernetes, the `victoria-metrics-k8s-stack` Helm chart plus the VictoriaMetrics Operator gives you `VMAgent`, `VMSingle`/`VMCluster`, `VMAlert`, and can convert existing `ServiceMonitor`/`PodMonitor`/`PrometheusRule` CRDs automatically.

### 3. Alerting with `vmalert` and a taste of MetricsQL

```bash
vmalert \
  -datasource.url=http://victoriametrics:8428 \
  -remoteWrite.url=http://victoriametrics:8428 \   # persist recording rules + alert state
  -notifier.url=http://alertmanager:9093 \
  -rule=/etc/vmalert/rules/*.yml
```

Rule files use the same format as Prometheus:

```yaml
groups:
  - name: api
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{code=~"5.."}[5m])) by (service)
            / sum(rate(http_requests_total[5m])) by (service) > 0.05
        for: 10m
        labels: { severity: page }
```

MetricsQL is a superset of PromQL. A few handy extensions:

```promql
# WITH templates to avoid repeating selectors
WITH (f = {service="api", env="prod"})
  sum(rate(http_requests_total{f}[5m])) / sum(rate(http_requests_total{f}[5m] offset 1d))

# Keep the metric name after a function (PromQL drops it)
rate(http_requests_total[5m]) keep_metric_names

# Fill gaps with a default and cap outliers
(rate(errors_total[5m]) default 0)
```

For the cluster version, the same tools work but URLs include a tenant ID:
write to `http://vminsert:8480/insert/0/prometheus/api/v1/write`, query `http://vmselect:8481/select/0/prometheus/`.

## Common pitfalls

- **Default retention is 1 month.** Set `-retentionPeriod` explicitly (`6`, `2y`, `90d`, etc.) on day one. Changing it later only affects data going forward from the next partition.
- **MetricsQL is *not* byte-for-byte PromQL.** `rate()`/`increase()` don't extrapolate like Prometheus and take the previous sample into account, so numbers can differ slightly (VictoriaMetrics argues they're more accurate). If you're running dual backends during a migration, expect small dashboard and alert-threshold differences; check alert thresholds that sit near a boundary.
- **HA Prometheus/vmagent pairs need deduplication.** If two replicas write the same series, set `-dedup.minScrapeInterval` (e.g. equal to the scrape interval) on single-node or on `vmselect`/`vmstorage` in a cluster, otherwise you store and sum duplicates.
- **Cluster replication is off by default.** `vmstorage` nodes shard data but don't replicate unless you set `-replicationFactor=N` on `vminsert` (and run at least `2N-1` storage nodes for safety). Losing a node without replication means losing that shard's data. Many teams instead run two independent clusters/regions fed by `vmagent` fan-out.
- **Cardinality still costs you.** It's more efficient than Prometheus, not immune. Unbounded labels (user IDs, request IDs, pod names on short-lived jobs) will blow up memory and index size. Use the cardinality explorer in `vmui`, `-search.maxUniqueTimeseries`, and `-storage.maxHourlySeries`/`-storage.maxDailySeries` limits, and drop labels in `vmagent` relabeling.
- **Backups need snapshots, not `cp`.** Copying the data dir of a running instance can produce an inconsistent backup. Use `vmbackup`, which works off instant snapshots (`/snapshot/create`) and supports incremental uploads to S3/GCS.
- **Some features are Enterprise-only.** Downsampling, per-tenant/per-label retention, and some security features (e.g. advanced auth in `vmauth`, mTLS between cluster components) require the commercial edition. Confirm what you need before committing to an architecture that depends on them.
- **Single-node → cluster is a migration, not a flag.** Data layouts differ; plan to use `vmctl` (or dual-write via `vmagent`) if you think you'll outgrow one node. Conversely, don't jump to the cluster version prematurely — single-node is simpler and scales vertically a long way.
- **Deleting series is for cleanup, not space reclaim.** `/api/v1/admin/tsdb/delete_series` hides data immediately, but disk space is only freed during background merges. Treat it as a last resort, and protect admin endpoints (e.g. behind `vmauth` or `-deleteAuthKey`).

## When *not* to pick it

If you need effectively unlimited, cheap retention on object storage, or your org has standardized on the Grafana LGTM stack, Grafana Mimir or Thanos may fit better. If you're tiny and Prometheus with 15 days of local retention is fine, you don't need anything else yet.
