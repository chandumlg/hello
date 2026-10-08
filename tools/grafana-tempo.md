# Grafana Tempo

Tempo is a distributed tracing backend that stores traces in cheap object storage and looks them up by trace ID (plus TraceQL search), solving the problem of tracing becoming too expensive and operationally heavy to run at full sampling rates.

## Primary use cases

- **High-volume tracing without a search-index cluster.** Tempo needs no Elasticsearch or Cassandra. Blocks of traces are written in Parquet to S3/GCS/Azure Blob, so storage cost scales with object-storage pricing and you can keep far more traces (or sample far less aggressively).
- **Metrics/logs/traces correlation in Grafana.** Exemplars on Prometheus metrics link to a trace ID, and Loki log lines containing a trace ID link straight to the trace. Tempo's metrics-generator also derives RED metrics and service graphs from spans.
- **Ad-hoc trace exploration with TraceQL.** Query by span attributes, durations and structure (e.g. "traces where a `db` span under `checkout` took >500ms") without pre-defining indexed tags.
- **Drop-in OpenTelemetry backend.** It accepts OTLP, Jaeger and Zipkin protocols, so existing instrumentation keeps working.

A team typically adopts Tempo when it already runs the Grafana stack (Prometheus, Loki), is standardizing on OpenTelemetry, and finds that Jaeger-on-Elasticsearch or a vendor APM is too costly or heavy to operate at its trace volume.

## Basic usage examples

**1. Run Tempo locally (single binary, local storage)**

```yaml
# tempo.yaml
server:
  http_listen_port: 3200
distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
storage:
  trace:
    backend: local
    local:
      path: /var/tempo/blocks
    wal:
      path: /var/tempo/wal
```

```bash
docker run -d --name tempo -p 3200:3200 -p 4317:4317 -p 4318:4318 \
  -v $(pwd)/tempo.yaml:/etc/tempo.yaml \
  grafana/tempo:latest -config.file=/etc/tempo.yaml
```

**2. Send traces from an OpenTelemetry-instrumented app**

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_SERVICE_NAME=checkout
# run your app with its OTel SDK or auto-instrumentation agent
```

**3. Query by ID and with TraceQL**

```bash
# fetch a trace by ID
curl http://localhost:3200/api/traces/<traceID>

# TraceQL search: slow checkout requests that hit an erroring span
curl -G http://localhost:3200/api/search \
  --data-urlencode 'q={ resource.service.name="checkout" && duration > 500ms && status = error }'
```

In Grafana, add Tempo as a data source and use Explore → TraceQL, and configure "trace to logs" / "trace to metrics" to enable one-click pivots.

## Common pitfalls

- **Search is not free.** Lookup by trace ID is fast; TraceQL search scans blocks. Unbounded time ranges on large datasets are slow and costly, so narrow the window and use Parquet-friendly dedicated attribute columns for hot attributes.
- **No trace ID, no entry point.** Tempo is designed around jumping in from metrics exemplars or logs. If your logs don't include trace IDs (or exemplars aren't enabled), the UX degrades to search-only.
- **Sampling still matters.** Tail sampling (e.g. in the OpenTelemetry Collector) must route all spans of a trace to the same collector instance, or traces get truncated. Plan load-balancing by trace ID.
- **Late spans and incomplete traces.** Spans arriving after the ingester's trace-idle period split a trace across blocks. Tune `trace_idle_period` and `max_block_duration` for long-running traces.
- **Local backend is for demos.** Production needs object storage, microservices or scalable single-binary mode, a retention setting on the compactor, and attention to ingester WAL durability (persistent volumes, replication factor).
- **Metrics-generator cardinality.** Span-derived metrics with high-cardinality dimensions (user IDs, URLs) can blow up your Prometheus/Mimir series count; restrict dimensions.
