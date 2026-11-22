# Redpanda Connect (formerly Benthos)

**Redpanda Connect is a single-binary, declarative stream processor that moves and transforms data between message brokers, databases, APIs, and files, so you don't write and run a bespoke Go/Java consumer service for every "read from X, reshape, write to Y" pipeline.**

It started life as the open-source project **Benthos** and was acquired by Redpanda in 2024 and renamed. The core engine is still MIT-licensed. Some connectors are under the Redpanda Community License, and a few are enterprise-only. It works with any Kafka-compatible broker, not just Redpanda, and with dozens of systems that have nothing to do with Kafka.

---

## 1. What problem does it solve?

Most data platforms end up with a lot of "glue" services: a consumer that reads Kafka, drops some fields, calls an enrichment API, and writes to S3. Another one that fans out webhooks into NATS. Another that copies Postgres CDC events into Elasticsearch. Each one is a small codebase with its own retry logic, batching, metrics, deploy pipeline, and on-call burden.

Redpanda Connect replaces that glue with **config**. A pipeline is a YAML file with three parts:

```
input  ──►  pipeline (processors)  ──►  output
```

The runtime takes care of the hard parts that hand-rolled consumers usually get wrong: back-pressure, at-least-once delivery (a message is only acked upstream once the output confirms it), batching, retries with backoff, dead-letter routing, graceful shutdown, and Prometheus metrics and OpenTelemetry tracing out of the box.

Transformations are written in **Bloblang**, a small, purpose-built mapping language that is much safer than embedding arbitrary scripts.

## 2. Primary use cases and when to adopt it

| Use case | Example |
|---|---|
| **Broker-to-sink ingestion** | Kafka topic → batched Parquet/JSON files in S3/GCS; Kafka → ClickHouse/Snowflake/Elasticsearch |
| **Message bridging/migration** | RabbitMQ or SQS → Kafka during a platform migration; MQTT (IoT) → NATS |
| **Lightweight stream transforms** | Redact PII, flatten nested JSON, normalize schemas, route by field to different topics |
| **Enrichment** | Look up each event in Redis, Postgres, or an HTTP service and merge the result in, with caching |
| **Webhook/HTTP gateway** | Accept HTTP POSTs, validate, and publish to a broker with proper acking |
| **CDC fan-out** | Consume Debezium/Postgres CDC events and write them to multiple downstream stores |
| **AI pipelines** | Newer processors call embedding/LLM endpoints (OpenAI, Ollama, Bedrock, etc.) and write vectors to pgvector/Qdrant |

**Adopt it when:**
- You have (or are about to write) several small "consume → reshape → produce" services and the logic is mostly *stateless* per message.
- Platform teams want a **standard, observable pattern** for data movement instead of N different hand-rolled consumers.
- You need to bridge heterogeneous systems (queues, HTTP, files, DBs) without pulling in a heavy framework.

**Don't adopt it when:**
- You need real **stateful stream processing**: windowed joins across streams, exactly-once aggregations over large state, event-time watermarks. Use Flink, Kafka Streams, or RisingWave for that.
- The connector you need is a first-class Kafka Connect connector with complex semantics (e.g., Debezium itself). Redpanda Connect can *consume* Debezium output, but Kafka Connect is often the better source.

## 3. Getting started

Install it (it's one static binary):

```bash
# macOS
brew install redpanda-data/tap/redpanda

# or the standalone binary via Docker
docker run --rm -it redpandadata/connect --help

# or via the rpk CLI
rpk connect --help
```

### Example 1: Hello world, then stdin → transform → stdout

```bash
rpk connect create stdin/bloblang/stdout > hello.yaml   # scaffolds a config
```

Edit `hello.yaml`:

```yaml
input:
  stdin: {}

pipeline:
  processors:
    - mapping: |
        root.message = this.msg.uppercase()
        root.length  = this.msg.length()
        root.seen_at = now()

output:
  stdout: {}
```

Run it:

```bash
echo '{"msg":"hello world"}' | rpk connect run hello.yaml
# {"length":11,"message":"HELLO WORLD","seen_at":"2026-11-22T09:14:03.112Z"}
```

`this` is the incoming document and `root` is the output document. Bloblang mappings are pure, so you can test them in isolation with `rpk connect blobl server` (a local web playground) before you wire up real brokers.

### Example 2: Kafka → redact PII → batched files in S3, with a dead-letter queue

```yaml
input:
  kafka_franz:
    seed_brokers: [ "${KAFKA_BROKERS}" ]
    topics: [ "user-events" ]
    consumer_group: "events-to-s3"

pipeline:
  processors:
    - mapping: |
        # Drop internal heartbeat events entirely
        root = if this.type == "heartbeat" { deleted() } else { this }
        # Redact PII
        root.email = this.email.hash("sha256").encode("hex")
        root.ip    = deleted()
        root.event_date = this.timestamp.ts_parse("2006-01-02T15:04:05Z07:00").ts_format("2006-01-02")

output:
  fallback:
    - aws_s3:
        bucket: analytics-raw
        path: 'user-events/dt=${! json("event_date") }/${! uuid_v4() }.jsonl'
        batching:
          count: 5000
          period: 30s
          processors:
            - archive:
                format: lines
    # If S3 writes keep failing, route to a DLQ topic instead of blocking forever
    - kafka_franz:
        seed_brokers: [ "${KAFKA_BROKERS}" ]
        topic: user-events-dlq

metrics:
  prometheus: {}
```

Notes:
- `${KAFKA_BROKERS}` is environment-variable interpolation. `${! ... }` is per-message interpolation evaluated at runtime.
- Offsets are only committed after the S3 write (or the DLQ write) succeeds. That gives you at-least-once delivery without writing any ack code.
- `fallback` tries outputs in order. Other output brokers include `switch` (content-based routing), `fan_out`, and `round_robin`.

### Example 3: Enrichment with a cache, and unit tests for the pipeline

```yaml
input:
  nats_jetstream:
    urls: [ "nats://localhost:4222" ]
    subject: orders.created
    durable: enricher

pipeline:
  processors:
    - branch:
        request_map: 'root = this.customer_id'
        processors:
          - cached:
              cache: customers
              key: '${! content() }'
              ttl: 10m
              processors:
                - http:
                    url: 'http://customer-svc/customers/${! content() }'
                    verb: GET
        result_map: 'root.customer = this'

output:
  kafka_franz:
    seed_brokers: [ "localhost:9092" ]
    topic: orders-enriched
    key: '${! json("order_id") }'

cache_resources:
  - label: customers
    memory:
      default_ttl: 10m
```

`branch` sends a *derived* message (just the customer ID) to the enrichment step and merges the result back. The original message isn't mangled if the lookup shape is different.

Pipelines can be unit tested. Put this in `enrich_test.yaml` next to the config:

```yaml
tests:
  - name: redacts and tags
    target_processors: '/pipeline/processors'
    input_batch:
      - content: '{"msg":"hi"}'
    output_batches:
      - - json_equals: { "message": "HI", "length": 2 }
```

```bash
rpk connect lint ./pipelines/*.yaml   # static validation, good for CI
rpk connect test ./pipelines/...      # runs the YAML-defined tests
```

## 4. Common pitfalls and things to watch out for

1. **It's at-least-once, not exactly-once.** Retries and restarts *will* produce duplicates. Make sinks idempotent: deterministic keys, upserts, `MERGE` on load, or dedupe downstream. The `dedupe` processor with a cache helps but is bounded by cache size and TTL.

2. **Ordering versus throughput.** Raising `max_in_flight` on outputs or adding parallel `threads` boosts throughput but can reorder messages within a partition. If per-key ordering matters (e.g., CDC), keep `max_in_flight: 1` for that pipeline or partition by key and accept lower throughput.

3. **Not a stateful engine.** People stretch it into windowed aggregations using caches and `group_by`. It works for small cases, but state lives in the process (or an external cache) with no checkpointing semantics like Flink's. If you're reaching for joins and windows, you've outgrown it.

4. **Infinite-retry stalls.** By default, a failing output retries forever and back-pressures the whole input, which looks like "the consumer stopped." Decide on a policy for each pipeline: `fallback` to a DLQ, `retry` with `max_retries`, or `drop_on` for specific errors. Alert on rising consumer lag.

5. **Processor errors don't drop messages, they flag them.** A failed `mapping` or `http` processor marks the message as errored and **passes it on**. Unless you handle it with `catch`, `try`, a `switch` output with `check: errored()`, or `reject_errored`, bad data can quietly flow to your sink.

6. **Bloblang `null` and missing-field semantics.** Referencing a missing field on `this` causes a mapping error in many contexts. Use `.or(default)`, `.catch(default)`, or `this.exists("field")` deliberately, and test edge cases with the `blobl` playground.

7. **Licensing and naming confusion.** Since the rename, docs, Docker images, and binaries have changed names (`benthos` → `redpanda-connect` / `rpk connect`), and some connectors moved to the Redpanda Community License or Enterprise. Check the license of each connector you depend on. If you need a purely MIT/community fork, look at **Bento** (WarpStream's fork), which kept the original Benthos codebase.

8. **Config sprawl.** One giant YAML with dozens of `switch` branches gets unreadable fast. Use `resources` (named, reusable inputs, processors, and outputs), split configs with `-r resources/*.yaml`, and treat pipelines like code: lint and test them in CI, review them, and version them.

9. **Observe it like a service.** Scrape the Prometheus endpoint and alert on `output_error`, `processor_error`, and input lag. Set `http.address` for the `/ready` and `/ping` health checks in Kubernetes probes. The streams mode (`rpk connect streams`) can run many pipelines in one process. That's convenient, but a noisy pipeline can then starve the others, so isolate critical flows into their own deployments.

---

**TL;DR:** If your platform has a growing pile of small "read, reshape, write" services, Redpanda Connect lets you replace them with linted, tested, observable YAML, with delivery guarantees handled for you. Keep it for stateless and lightly enriched flows, and hand true stateful stream processing to Flink or Kafka Streams.
