# Vector

Vector is a high-performance, vendor-neutral observability data pipeline (written in Rust) that collects, transforms, and routes logs, metrics, and traces from any source to any destination, solving the problem of telemetry being tightly coupled to a specific backend and agent.

## Primary use cases

- **Replacing a zoo of agents.** Instead of running Filebeat, Fluentd, Telegraf, and a vendor agent side by side, one Vector binary tails files, scrapes Prometheus endpoints, reads journald, and receives syslog/OTLP/Kafka.
- **Cost control before data hits the vendor.** Filter, sample, dedupe, drop noisy fields, and aggregate logs into metrics in-flight, so Datadog/Splunk/Elastic bills reflect only the data you actually need.
- **Backend portability and dual-writing.** Route the same stream to S3 for cheap archive and to Loki/Elasticsearch/Datadog for search, making a vendor migration a config change rather than a re-instrumentation project.
- **Centralized enrichment and redaction.** Run Vector as an aggregator tier to scrub PII, add Kubernetes metadata, normalize schemas, and enforce routing policy in one place.

Teams typically adopt Vector when log/telemetry spend becomes a line item, when they're migrating between observability vendors, or when platform engineering wants one consistent, testable pipeline instead of per-team agent configs. Common topologies are agent-per-node (DaemonSet) feeding an aggregator deployment, or a single stateless aggregator behind a load balancer.

## Basic usage examples

**1. Install and run a minimal pipeline** (`vector.yaml`):

```yaml
sources:
  app_logs:
    type: file
    include: ["/var/log/myapp/*.log"]

transforms:
  parse:
    type: remap
    inputs: [app_logs]
    source: |
      . = parse_json!(.message)
      .env = "prod"
      del(.debug_blob)

sinks:
  console:
    type: console
    inputs: [parse]
    encoding: { codec: json }
```

```bash
curl --proto '=https' --tlsv1.2 -sSfL https://sh.vector.dev | bash
vector validate vector.yaml     # check config (add --no-environment to skip connectivity checks)
vector --config vector.yaml
```

**2. Filter, route, and fan out to multiple sinks:**

```yaml
transforms:
  split:
    type: route
    inputs: [parse]
    route:
      errors: '.level == "error"'
  sample_info:
    type: sample
    inputs: [split._unmatched]
    rate: 10            # keep 1 in 10 non-error events

sinks:
  search:
    type: loki
    inputs: [split.errors, sample_info]
    endpoint: http://loki:3100
    labels: { service: "{{ service }}" }
    encoding: { codec: json }
  archive:
    type: aws_s3
    inputs: [parse]     # full-fidelity copy
    bucket: my-log-archive
    key_prefix: "date=%F/"
    compression: gzip
    encoding: { codec: json }
```

**3. Unit-test your transforms (VRL) in CI:**

```yaml
tests:
  - name: drops debug blob and tags env
    inputs:
      - insert_at: parse
        type: log
        log_fields: { message: '{"msg":"hi","debug_blob":"x"}' }
    outputs:
      - extract_from: parse
        conditions:
          - type: vrl
            source: '.env == "prod" && !exists(.debug_blob)'
```

```bash
vector test vector.yaml
vector top        # live per-component throughput (needs the API enabled)
```

## Common pitfalls

- **VRL is fallible by design.** Functions like `parse_json` return errors that must be handled (`!` to abort the event, `?? default`, or explicit error branches). Unhandled aborts silently drop or route events to the `dropped` output, so configure `drop_on_abort`/`reroute_dropped` deliberately and alert on `component_errors_total`.
- **Delivery guarantees are opt-in.** Without end-to-end acknowledgements (`acknowledgements.enabled: true` on sinks) and disk buffers (`buffer.type: disk`), a crash or downstream outage can lose data. The default memory buffer is fast but volatile; pick `when_full: block` vs `drop_newest` consciously, since blocking applies backpressure upstream.
- **High-cardinality metrics.** Converting logs to metrics (`log_to_metric`) with unbounded tag values (user IDs, request IDs) will blow up your metrics backend; Vector itself also holds aggregation state in memory.
- **Aggregator scaling and statefulness.** Stateful transforms (`reduce`, `dedupe`, aggregation) are per-instance, so load-balancing across multiple aggregators gives different results than a single instance unless you shard consistently.
- **Config sprawl and version drift.** Large pipelines get hard to reason about; split config into multiple files/directories, run `vector validate` and `vector test` in CI, and pin versions since component options and VRL functions evolve between releases.
- **Not a storage or query engine.** Vector moves and shapes data; it has no durable query layer, so pair it with a real backend (Loki, ClickHouse, Elasticsearch, S3 + query engine).
