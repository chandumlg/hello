# Thanos

Thanos is a set of components that extend Prometheus with durable long-term storage in object storage (S3/GCS/Azure), a global query view across many Prometheus instances, and downsampling — so you get unlimited retention and a single PromQL endpoint without abandoning the Prometheus ecosystem.

## Why teams adopt it

Prometheus is intentionally a single-node, local-disk system. That is great until you run many clusters or regions, need a year of metrics for capacity planning, or want HA Prometheus pairs without duplicate series in dashboards. Thanos layers on top of existing Prometheus servers rather than replacing them.

Common adoption triggers:
- Local retention (typically 15-30 days) is too short, and disks are getting expensive or fragile.
- You operate multiple Prometheus servers (per cluster, per region, per tenant) and want one query/alerting entry point.
- You run HA Prometheus replicas and need deduplicated results.
- You want cheap, long-range queries (months/years) through downsampling (5m and 1h resolutions).
- Compared with Grafana Mimir/Cortex (push-based, centralized TSDB), Thanos is a pull/sidecar-friendly, incremental path: teams already running Prometheus can adopt it piece by piece.

Core components: **Sidecar** (ships TSDB blocks to object storage, serves recent data), **Store Gateway** (serves historical blocks from the bucket), **Querier** (fans out PromQL, deduplicates), **Compactor** (compacts and downsamples blocks, enforces retention), **Ruler** (evaluates rules against the global view), and optionally **Receive** (remote-write ingest) and **Query Frontend** (caching and query splitting).

## Basic usage

**1. Object storage config shared by components (`bucket.yml`):**
```yaml
type: S3
config:
  bucket: my-thanos-metrics
  endpoint: s3.us-east-1.amazonaws.com
  region: us-east-1
```

**2. Run a sidecar next to Prometheus (Prometheus needs fixed external labels and `--storage.tsdb.min-block-duration=2h --storage.tsdb.max-block-duration=2h`):**
```bash
# prometheus.yml: global.external_labels: { cluster: prod-us-east-1, replica: A }
thanos sidecar \
  --tsdb.path /var/prometheus \
  --prometheus.url http://localhost:9090 \
  --objstore.config-file bucket.yml \
  --grpc-address 0.0.0.0:10901
```

**3. Run a querier that fans out to sidecars and store gateways, deduplicating HA replicas:**
```bash
thanos store --objstore.config-file bucket.yml --grpc-address 0.0.0.0:10905 &
thanos query \
  --http-address 0.0.0.0:9091 \
  --endpoint sidecar-a:10901 --endpoint sidecar-b:10901 \
  --endpoint store-gw:10905 \
  --query.replica-label replica
# Point Grafana at http://thanos-query:9091 as a normal Prometheus data source.
```
Run exactly one `thanos compact` per bucket to handle compaction, downsampling and retention (`--retention.resolution-raw=90d`).

## Pitfalls

- **External labels must be unique and stable.** Missing or colliding labels (`cluster`, `replica`) cause overlapping blocks, bad deduplication, or compactor halts.
- **Run only one compactor per bucket.** Concurrent compactors corrupt data; a halted compactor (due to overlapping/ malformed blocks) silently stops downsampling until fixed.
- **Downsampling does not save storage.** It stores additional resolutions to make long-range queries fast; raw data is what costs space, so set retention per resolution deliberately.
- **Store Gateway cost and latency.** It needs local disk for index caches and can be slow cold; add index/chunk caching (memcached/Redis) and a Query Frontend for big ranges.
- **Cardinality is still your problem.** Thanos stores whatever Prometheus scrapes; high-cardinality series inflate bucket size, store memory and query time.
- **Sidecar uploads lag by ~2h** (block cut time); recent data comes from the sidecar's live Prometheus, so keep sidecars reachable from queriers.
- **Operational surface area.** Many components, gRPC networking and TLS between them; consider Receive or a managed/Mimir-style system if you'd rather centralize than federate.
