# OpenTelemetry

OpenTelemetry (often shortened to "OTel") is a vendor-neutral standard and set of SDKs/APIs for generating, collecting, and exporting traces, metrics, and logs from your applications, solving the problem of every observability vendor requiring its own proprietary instrumentation that locks you in and has to be re-done if you switch backends.

## Primary use cases and when to adopt it

- **Distributed tracing across microservices.** When a single user request fans out across a dozen services, OTel's context propagation lets you follow one request end-to-end and see exactly where latency or errors are introduced.
- **Vendor-neutral instrumentation.** Teams instrument their code once with the OTel API/SDK, then route the data to whichever backend they want (Jaeger, Tempo, Honeycomb, Datadog, New Relic, an in-house stack) via the OpenTelemetry Collector — swapping backends becomes a config change, not a re-instrumentation project.
- **Unifying traces, metrics, and logs.** Rather than running separate agents/libraries for each signal type, OTel gives you one SDK and one collector pipeline for all three, which platform teams appreciate for reducing operational surface area.
- **LLM/AI application observability.** OTel's emerging GenAI semantic conventions standardize how to capture prompts, completions, token counts, and model latency, so AI engineers building agents or RAG pipelines get structured tracing of LLM calls alongside the rest of their infra — this is why it shows up heavily in tools like LangSmith-alternatives, Traceloop, and Arize built on top of OTel.

A team typically adopts OTel when they're past a handful of services and manual `print`/log-grepping stops scaling, or when they want to avoid being locked into a single observability vendor's SDK before they've picked a long-term backend.

## Basic usage examples

**1. Auto-instrument a Python service with zero code changes:**
```bash
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap --action=install

opentelemetry-instrument \
  --traces_exporter otlp \
  --metrics_exporter otlp \
  --service_name my-service \
  python app.py
```
This wraps common libraries (Flask, requests, psycopg2, etc.) automatically and ships spans to an OTLP endpoint.

**2. Manual instrumentation for a custom span (Node.js):**
```javascript
const { trace } = require('@opentelemetry/api');
const tracer = trace.getTracer('checkout-service');

async function chargeCard(orderId) {
  return tracer.startActiveSpan('charge-card', async (span) => {
    span.setAttribute('order.id', orderId);
    try {
      const result = await paymentGateway.charge(orderId);
      return result;
    } finally {
      span.end();
    }
  });
}
```

**3. Run a local Collector to receive and fan out telemetry (Docker):**
```bash
docker run -p 4317:4317 -p 4318:4318 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otelcol/config.yaml \
  otel/opentelemetry-collector:latest
```
```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols: { grpc: {}, http: {} }
exporters:
  logging: {}
  otlp:
    endpoint: "jaeger:4317"
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [logging, otlp]
```

## Common pitfalls

- **Sampling surprises.** Head-based sampling (the default in many SDKs) decides whether to keep a trace before it knows if the request errored — you can end up dropping exactly the slow or failing traces you cared about. Tail-based sampling (usually done in the Collector) fixes this but adds infrastructure complexity.
- **Cardinality explosions in metrics.** Attaching high-cardinality attributes (user IDs, request IDs) to metrics rather than traces/logs will blow up costs and can crash time-series backends — that data belongs on spans, not metric labels.
- **Version/API churn.** OTel's semantic conventions (especially the GenAI ones for LLM tracing) are still evolving and have had breaking renames; pin SDK versions and expect occasional migration work.
- **The Collector becomes a single point of failure if misconfigured.** Running it without retry/queueing (`sending_queue`, `retry_on_failure`) means a backend outage can silently drop telemetry instead of buffering it.
- **Auto-instrumentation isn't free performance-wise.** Wrapping every library call adds overhead; in latency-sensitive hot paths, profile before/after and consider selectively disabling instrumentors you don't need.
- **Context propagation gaps.** Trace context has to be manually threaded through async boundaries, message queues, and background jobs (Kafka, Celery, etc.) — if a team forgets this, traces silently break into disconnected fragments instead of one continuous request.
