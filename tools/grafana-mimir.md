# Grafana Mimir

Mimir is a horizontally scalable, multi-tenant, long-term storage backend for Prometheus metrics, solving the problem that a single Prometheus server cannot hold billions of active series, retain data for years, or be made highly available on its own.

## Primary use cases

- **Long-term, durable metrics retention.** Prometheus keeps data on local disk, typically for days or weeks. Mimir stores blocks in object storage (S3/GCS/Azure), so retention of a year or more is cheap and survives node loss.
- **Global view across many clusters.** Each cluster runs Prometheus (or Grafana Alloy) in agent mode and `remote_write`s to one Mimir. Dashboards and alerts then query a single endpoint instead of federating dozens of servers.
- **Multi-tenant metrics platform.** A platform team can offer metrics-as-a-service: each team or environment gets a tenant ID (`X-Scope-OrgID`) with its own limits (series cap, ingestion rate) and isolation, so one noisy team cannot take down everyone.
- **Scale beyond a single Prometheus.** Mimir is designed for 1B+ active series by sharding ingestion, storage and queries across components, while staying PromQL-compatible.

A team usually adopts Mimir when Prometheus is already core to its observability but it hits one of: memory exhaustion from high cardinality, a need for HA/deduplicated data, retention requirements longer than local disk allows, or a central platform team that needs per-tenant quotas. If a single Prometheus (or a Prometheus pair) comfortably handles your load, Mimir is unnecessary operational weight.

## Basic usage examples

**1. Run a single-binary Mimir locally (monolithic mode):**

```bash
docker run -d --name mimir -p 9009:9009 \
  grafana/mimir:latest \
  -target=all -auth.multitenancy-enabled=false \
  -server.http-listen-port=9009
# Remote-write endpoint: http://localhost:9009/api/v1/push
# Prometheus-compatible query API: http://localhost:9009/prometheus
```

**2. Point Prometheus at Mimir with `remote_write`:**

```yaml
# prometheus.yml
remote_write:
  - url: http://localhost:9009/api/v1/push
    headers:
      X-Scope-OrgID: team-payments   # tenant ID (needed when multitenancy is on)
    queue_config:
      max_samples_per_send: 2000
```

**3. Query it with PromQL (and wire it into Grafana):**

```bash
curl -s -H 'X-Scope-OrgID: team-payments' \
  'http://localhost:9009/prometheus/api/v1/query' \
  --data-urlencode 'query=sum by (job) (rate(http_requests_total[5m]))'
```

In Grafana, add a regular **Prometheus** data source with URL `http://mimir:9009/prometheus` and a custom header `X-Scope-OrgID`. For production, deploy via the `mimir-distributed` Helm chart, configure object storage in `blocks_storage`, and use `mimirtool` to load rules and check cardinality.

## Common pitfalls

- **Cardinality is still your enemy.** Mimir scales far, but unbounded labels (user IDs, request IDs, pod hashes) still blow up cost and ingester memory. Set per-tenant `max_global_series_per_user` and monitor with `mimirtool analyze`.
- **Default limits reject data.** Out-of-the-box per-tenant ingestion rate, series and label-size limits are conservative; you will see `429`/`err-mimir-*` errors on `remote_write` until limits are tuned via runtime overrides.
- **Tenant ID mismatches look like missing data.** Writing with one `X-Scope-OrgID` and querying with another (or with multitenancy disabled) returns empty results, not an error. Standardize header injection (e.g. via an auth proxy).
- **Operational complexity.** Distributor, ingester, compactor, store-gateway, querier, query-frontend, ruler and more, plus memcached caches. Start with monolithic or read/write mode and move to microservices only when needed.
- **Ingesters are stateful.** Replication factor 3 and zone-aware replication matter; rolling restarts must be paced, and ingester disk/WAL loss reduces durability for recent, not-yet-flushed data.
- **Out-of-order and old samples.** Samples older than the allowed window are dropped unless `out_of_order_time_window` is configured, which matters for delayed `remote_write` after outages.
- **Remote-write adds lag and cost.** Dashboards on Mimir are slightly behind real time, and egress/queue memory on the sender grows during outages; tune `queue_config` and keep local Prometheus alerting for critical paths if needed.
