# Apache Kafka

Kafka is a distributed, durable event log that lets services publish and subscribe to streams of records at very high throughput, decoupling producers from consumers so systems can scale, replay, and fail independently instead of calling each other directly.

## Why teams adopt it

Staff, platform, and AI engineers reach for Kafka when point-to-point integrations (service A calls service B calls service C) start to buckle: a slow or down consumer shouldn't block a producer, multiple independent teams need the same event stream, and you need to replay history — not just process it once and discard it, the way a typical message queue does. Because Kafka retains records on disk for a configurable period (or forever, with compaction), new consumers can join later and read from the beginning, and existing consumers can reprocess data after a bug fix.

Common adoption triggers:
- Fan-out event distribution — one write (an order placed, a user signed up) needs to reach many independent downstream systems (analytics, fraud detection, email, search indexing) without the producer knowing or caring who's listening.
- Decoupling microservices so producers and consumers can deploy, scale, and fail independently, with the broker absorbing backpressure instead of callers timing out on each other.
- Building event-sourced or CDC (change-data-capture) pipelines, often paired with Debezium, where every state change is captured as an immutable, replayable event.
- Streaming data pipelines feeding data warehouses, search indexes, or ML feature stores, where throughput (millions of events/sec) and ordering-per-key guarantees matter more than a traditional queue's fire-and-acknowledge model.
- Real-time stream processing with Kafka Streams or ksqlDB, where you need to join, aggregate, or window events as they arrive rather than batch-processing them later.

## Basic usage

**1. Start a local broker (KRaft mode, no ZooKeeper needed in modern Kafka):**
```bash
# Using the Kafka binaries
bin/kafka-storage.sh format -t $(bin/kafka-storage.sh random-uuid) -c config/kraft/server.properties
bin/kafka-server-start.sh config/kraft/server.properties

# Or via Docker
docker run -d --name kafka -p 9092:9092 apache/kafka:latest
```

**2. Create a topic and produce/consume from the CLI:**
```bash
bin/kafka-topics.sh --create --topic orders --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1

bin/kafka-console-producer.sh --topic orders --bootstrap-server localhost:9092
# type messages, one per line, then Ctrl-D

bin/kafka-console-consumer.sh --topic orders --from-beginning --bootstrap-server localhost:9092
```

**3. Produce and consume from application code (Python, `confluent-kafka`):**
```python
from confluent_kafka import Producer, Consumer

producer = Producer({"bootstrap.servers": "localhost:9092"})
producer.produce("orders", key="order-123", value="order-placed")
producer.flush()

consumer = Consumer({
    "bootstrap.servers": "localhost:9092",
    "group.id": "order-processors",
    "auto.offset.reset": "earliest",
})
consumer.subscribe(["orders"])

while True:
    msg = consumer.poll(1.0)
    if msg is None or msg.error():
        continue
    print(f"{msg.key()}: {msg.value()}")
```

## Pitfalls to watch out for

- **Partition count is a scaling decision you can't easily undo.** Consumers scale up to one-per-partition (extra consumers in a group sit idle), and you can add partitions later but doing so reshuffles key-to-partition mapping, breaking ordering guarantees for existing keys. Under-provisioning early is a common regret.
- **Ordering is only guaranteed within a partition.** If two events for the same entity (e.g., the same order) land on different partitions, consumers can see them out of order — always key by the entity whose events must stay ordered.
- **"At-least-once" is the default, not exactly-once.** Consumer crashes or rebalances after processing but before committing an offset cause reprocessing; downstream logic needs to be idempotent unless you've deliberately configured Kafka's transactional/exactly-once semantics (which add complexity and overhead).
- **Consumer lag is easy to miss until it's a crisis.** A slow consumer doesn't get an error — it just falls further behind, and if it falls behind the topic's retention window, it silently loses data. Monitor lag explicitly (Burrow, Kafka Lag Exporter, or broker-native metrics).
- **Unbounded retention plus high throughput means real disk and cost planning.** Compacted topics (keyed "latest value wins" topics) and time/size-based retention need to be chosen deliberately per topic — the default settings that work for a low-volume topic can quietly fill disks on a high-volume one.
- **It's a heavier operational commitment than a simple queue.** Running Kafka well (broker sizing, partition rebalancing, monitoring ISR/under-replicated partitions) is real operational work; many teams start on a managed service (Confluent Cloud, MSK, Redpanda Cloud) rather than self-hosting from day one.
