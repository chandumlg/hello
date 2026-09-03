# Apache Flink

Flink is a distributed engine for computing over unbounded streams of data with low latency and exactly-once correctness, so teams can produce continuously updated results (aggregates, joins, alerts) instead of waiting for the next batch job to run.

## Primary use cases

Kafka (or Pulsar, Kinesis) gets events from producers to consumers; Flink is what you reach for when a consumer needs to do more than read and forward — when it needs to hold state across events and time. Typical adopters:

- **Platform/data engineering teams** building real-time pipelines that join, aggregate, or deduplicate streams — e.g., computing rolling session metrics, joining a clickstream against a slowly-changing dimension table, or maintaining materialized views that stay fresh as source data changes.
- **Staff engineers replacing brittle "poll-and-batch" cron jobs** that reprocess a full table every N minutes with an incremental job that updates only what changed, cutting both latency and compute cost.
- **AI/ML platform teams** computing streaming features (e.g., "orders in the last 10 minutes per user") that need to be correct and low-latency for both training and online inference, avoiding train/serve skew between a batch feature pipeline and a real-time one.
- **Fraud detection, monitoring, and alerting systems** that need to evaluate complex event patterns (CEP) — "did X happen without Y within 30 seconds" — as events arrive rather than after the fact.

A team typically adopts Flink once `KTable`-style aggregation in a lighter framework (Kafka Streams, ksqlDB) hits its ceiling — multi-stream joins, custom windowing, very large keyed state, or the need to run the same job in both streaming and batch mode — or once cron-based batch reprocessing can no longer keep up with freshness requirements.

## Basic usage

**1. Run a local cluster and submit a job (via Docker):**
```bash
docker run -d --name flink-jm -p 8081:8081 flink:latest jobmanager
docker run -d --name flink-tm --link flink-jm:jobmanager flink:latest taskmanager

# Web UI for job status, checkpoints, backpressure: http://localhost:8081
```

**2. A minimal streaming word-count job (Java/Flink DataStream API), submitted with the CLI:**
```java
StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

env.fromSource(kafkaSource, WatermarkStrategy.noWatermarks(), "orders")
   .keyBy(order -> order.getUserId())
   .window(TumblingEventTimeWindows.of(Time.minutes(10)))
   .sum("amount")
   .sinkTo(kafkaSink);

env.execute("rolling-order-totals");
```
```bash
./bin/flink run -c com.example.RollingOrderTotals target/job.jar
```

**3. The same kind of job in SQL (Flink SQL, often the fastest way to prototype or the interface data teams actually use):**
```sql
CREATE TABLE orders (
  user_id STRING,
  amount DECIMAL(10,2),
  event_time TIMESTAMP(3),
  WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
) WITH ('connector' = 'kafka', 'topic' = 'orders', 'format' = 'json', ...);

SELECT user_id, TUMBLE_START(event_time, INTERVAL '10' MINUTE) AS window_start, SUM(amount)
FROM orders
GROUP BY user_id, TUMBLE(event_time, INTERVAL '10' MINUTE);
```

## Common pitfalls

- **Event time vs. processing time.** Using processing time (wall-clock time at the operator) instead of event time (the timestamp embedded in the record) silently produces wrong results whenever events arrive late or out of order — which is the normal case for anything crossing a network. Set watermarks and use event time for anything correctness-sensitive.
- **Unbounded state growth.** Keyed state (e.g., "all orders per user, ever") grows forever unless you set state TTL or use bounded windows. Unbounded state blows up checkpoint size and recovery time long before it causes an out-of-memory error, so it fails quietly until it doesn't.
- **Checkpointing cost vs. recovery guarantees.** Exactly-once semantics come from periodic checkpoints (consistent snapshots of all operator state). Checkpointing too infrequently means more reprocessing on failure; too frequently, especially with large state and a remote state backend, adds latency and I/O pressure. This needs tuning per job, not a default left alone.
- **Backpressure hides upstream, not downstream, problems.** A slow sink (e.g., a database that can't keep up with writes) shows up in the UI as backpressure on upstream operators, which can mislead engineers into optimizing the wrong stage of the pipeline.
- **The DataStream API and Flink SQL are not perfectly equivalent.** SQL is easier to write and read but has less control over state and timers; teams that start in SQL for speed sometimes have to drop to the DataStream API later for one operator, and mixing the two adds operational complexity.
- **Savepoints are not automatic.** Upgrading a running job's code (changing an operator, adding a field) without a compatible savepoint can silently lose state or fail to restore, since Flink matches state by operator UID, not by code structure.
