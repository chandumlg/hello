# Grafana Loki

Loki is a horizontally scalable log aggregation system that indexes only metadata (labels) instead of full log text, solving the problem of log storage and search costs exploding as services and log volume grow.

## Primary use cases

- **Centralized logging for microservices.** Instead of SSHing into individual hosts or containers to `tail` logs, teams ship all service logs to Loki and query them from one place, correlated by labels like `namespace`, `pod`, or `service`.
- **Kubernetes-native log collection.** Loki pairs with Promtail, Grafana Alloy, or Fluent Bit to auto-discover pods and attach the same labels Prometheus uses (`namespace`, `pod`, `container`), so a team already running Prometheus/Grafana gets logs "for free" in the same UI and label scheme.
- **Cost-sensitive log retention at scale.** Because Loki doesn't full-text index log bodies (it indexes only a small set of labels and stores compressed chunks in object storage like S3/GCS), it's dramatically cheaper to run at high volume than Elasticsearch-style systems — a common reason teams migrate off ELK once log volume gets expensive.
- **Metrics-to-logs correlation during incidents.** Since Loki is a first-class Grafana data source, an engineer looking at a Prometheus panel spike can jump straight to the exact logs for that service and time window without switching tools or re-authenticating.

A team typically adopts Loki once it already runs Prometheus and Grafana and wants logs in the same stack without paying for a heavyweight full-text search cluster, or once existing log storage costs (Elasticsearch, Splunk, a cloud logging service) have become a real line item.

## Basic usage examples

**1. Run Loki locally with Docker:**

```bash
docker run -d --name=loki -p 3100:3100 grafana/loki:latest
# Loki's HTTP API is now available at http://localhost:3100
```

**2. Ship logs into Loki with Promtail (the standard log-shipping agent):**

```yaml
# promtail-config.yaml
server:
  http_listen_port: 9080
positions:
  filename: /tmp/positions.yaml
clients:
  - url: http://localhost:3100/loki/api/v1/push
scrape_configs:
  - job_name: system
    static_configs:
      - targets: [localhost]
        labels:
          job: varlogs
          __path__: /var/log/*.log
```

```bash
docker run -v $(pwd)/promtail-config.yaml:/etc/promtail/config.yaml \
  -v /var/log:/var/log grafana/promtail:latest -config.file=/etc/promtail/config.yaml
```

**3. Query logs with LogQL, Loki's PromQL-inspired query language:**

```logql
# All error-level logs from the checkout service in the last hour
{namespace="prod", app="checkout"} |= "level=error"

# Rate of 5xx log lines per second, as a metric — pipe log queries into aggregations
sum(rate({app="checkout"} |= "status=5" [5m]))
```

**4. Add Loki as a Grafana data source (provisioned, so it's reproducible):**

```yaml
# provisioning/datasources/loki.yaml
apiVersion: 1
datasources:
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
```

## Common pitfalls

- **Over-labeling causes cardinality explosions.** Loki's index is built from label combinations ("streams"), so putting a high-cardinality value (request ID, user ID, raw timestamp) into a label instead of the log body creates millions of tiny streams and destroys performance. Keep labels to low-cardinality dimensions (`namespace`, `app`, `env`) and search everything else with LogQL filters.
- **Loki is not a full-text search engine.** Unindexed line filters (`|= "text"`) scan chunks at query time rather than hitting an inverted index, so broad, unlabeled searches across huge time ranges can be slow. Narrow by label first, then filter by text.
- **Out-of-order and old logs can be rejected.** By default Loki enforces that log entries within a stream arrive roughly in order and rejects entries older than the configured retention/ingestion window — batch jobs or backfills that replay old logs need `reject_old_samples` and ingestion limits tuned accordingly.
- **Single-binary mode doesn't scale — plan for microservices mode or SSD.** The simple all-in-one deployment is fine for getting started, but production workloads usually need Loki's scalable "simple scalable" or microservices deployment mode with object storage, which has real operational overhead (compactor, ruler, distributor/ingester tuning).
- **Retention and compaction need explicit configuration.** Without a configured retention period and a running compactor, chunks accumulate in object storage indefinitely and costs grow silently — set `retention_period` and enable the compactor from day one rather than after the storage bill shows up.
