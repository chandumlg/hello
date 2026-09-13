# Jaeger

Jaeger is an open-source distributed tracing system that lets you follow a single request as it hops across dozens of microservices, so you can see exactly where time is being spent and where things break.

## Why teams adopt it

Once a system grows past a handful of services, "why is this request slow?" or "which service actually failed?" stops being answerable from logs and dashboards alone — you need to see the request's full call graph with per-hop timing. Jaeger (originally built at Uber, now a CNCF graduated project) collects spans emitted by instrumented services, stitches them into traces, and gives you a UI and API to search and visualize them.

Platform and staff engineers typically bring in Jaeger when:
- A monolith has been split into enough microservices that latency and failures are hard to attribute to a single service from logs alone.
- They've already adopted OpenTelemetry for instrumentation and need a backend to store and query the resulting traces (Jaeger is one of the most common OTel trace backends, alongside Tempo, Zipkin, and vendor SaaS options).
- They need to debug intermittent latency spikes or cascading failures that only show up under real production traffic, not in isolated service tests.
- They want dependency-graph visibility — which services call which, and how often — without maintaining that graph by hand.

It pairs naturally with Prometheus/Grafana (metrics) and structured logging: metrics tell you *that* something's wrong, traces tell you *where* in the request path it's happening.

## Basic usage

**1. Run the all-in-one Jaeger instance locally (for evaluation/dev):**
```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```
This bundles the collector, query service, and UI in one container. Open `http://localhost:16686` for the UI.

**2. Instrument a service with OpenTelemetry and export to Jaeger (Python example):**
```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

trace.set_tracer_provider(TracerProvider())
exporter = OTLPSpanExporter(endpoint="localhost:4317", insecure=True)
trace.get_tracer_provider().add_span_processor(BatchSpanProcessor(exporter))

tracer = trace.get_tracer(__name__)
with tracer.start_as_current_span("handle_order"):
    with tracer.start_as_current_span("call_payment_service"):
        ...  # your business logic
```

**3. Query traces via the HTTP API (useful for scripting or CI checks):**
```bash
curl "http://localhost:16686/api/traces?service=order-service&limit=20&lookback=1h"
```

## Pitfalls to watch out for

- **Sampling matters a lot.** Tracing every request at high volume is expensive to store and query; most production setups use head- or tail-based sampling (e.g., always sample errors and slow requests, sample a small percentage of everything else). Getting this wrong means either missing the interesting traces or drowning in storage costs.
- **Traces are only as good as your instrumentation.** A single uninstrumented service or a dropped context-propagation header breaks the trace chain, leaving you with disconnected fragments instead of one end-to-end trace.
- **Storage backend choice matters for production.** The all-in-one image uses in-memory storage and loses everything on restart — production deployments need Elasticsearch, Cassandra, or Kafka-backed storage, each with its own operational overhead.
- **Jaeger's native protocol is being phased out in favor of OpenTelemetry.** New deployments should instrument with the OTel SDK/Collector and use Jaeger purely as a backend, rather than using Jaeger's legacy client libraries directly.
- **Clock skew across hosts** can make span timings look inconsistent (a child span appearing to start before its parent) if NTP isn't well synchronized across your fleet.
