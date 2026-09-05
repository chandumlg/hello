# Apache Pulsar

Pulsar is a distributed pub/sub messaging and streaming platform that solves the problem of running Kafka-like durable event streaming *and* traditional queue-style messaging on one system, with storage separated from serving so the two scale independently.

## Why teams adopt it

Pulsar's defining architectural choice is splitting compute from storage: brokers are stateless and handle routing/serving, while message data lives in a separate log storage layer (Apache BookKeeper). That split is what drives most adoption decisions:

- **Independent scaling of throughput vs. retention.** Need to keep years of data cheaply without over-provisioning brokers? Add bookies (storage nodes) without touching the broker tier, or vice versa. Kafka couples the two — scaling storage means scaling the broker/partition layer with it.
- **Multi-tenancy as a first-class concept.** Pulsar has built-in tenants and namespaces with per-namespace quotas, isolation, and auth — useful for platform teams running one cluster as shared infrastructure for many product teams, rather than standing up a Kafka cluster per team.
- **Unified queuing and streaming.** Pulsar supports exclusive, shared (competing-consumer/queue-style), failover, and key-shared subscription modes on the *same* topic — so you don't need both a queue (RabbitMQ/SQS) and a log (Kafka) to cover both patterns.
- **Tiered storage out of the box.** Older segments offload automatically to S3/GCS/Azure Blob, so "keep everything forever" doesn't mean "keep everything on local disk forever."
- **Geo-replication** is built into the broker, useful for multi-region event distribution without bolting on MirrorMaker-style tooling.

Teams typically reach for Pulsar over Kafka when they're running multi-tenant shared messaging infrastructure at a platform-engineering level, need mixed queue/stream consumption semantics on the same data, or want storage and compute to scale on separate cost curves. Teams reach for it over something like NATS when they need long-term durable retention and geo-replication rather than a lightweight low-latency bus.

## Basic usage

**1. Run a local standalone cluster (broker + BookKeeper + ZooKeeper in one process, for dev):**
```bash
docker run -it -p 6650:6650 -p 8080:8080 \
  apachepulsar/pulsar:3.3.0 \
  bin/pulsar standalone
```

**2. Produce and consume with the CLI:**
```bash
# Terminal 1: consume from a topic (creates it if it doesn't exist)
bin/pulsar-client consume orders -s "my-subscription" -n 0

# Terminal 2: produce a message
bin/pulsar-client produce orders --messages '{"order_id": "1234"}'
```

**3. Shared subscription for competing consumers (Java client):**
```java
PulsarClient client = PulsarClient.builder()
    .serviceUrl("pulsar://localhost:6650")
    .build();

Consumer<byte[]> consumer = client.newConsumer()
    .topic("orders")
    .subscriptionName("order-workers")
    .subscriptionType(SubscriptionType.Shared) // queue-style load balancing across consumers
    .subscribe();

Message<byte[]> msg = consumer.receive();
process(msg);
consumer.acknowledge(msg);
```

**4. Set up tiered storage offload (S3) on a namespace:**
```bash
bin/pulsar-admin namespaces set-offload-threshold \
  --size 10G my-tenant/my-namespace
```

## Common pitfalls

- **Operational surface area.** A production Pulsar deployment means running three coordinated systems — brokers, BookKeeper bookies, and ZooKeeper — versus Kafka's now-simpler KRaft (no separate ZooKeeper) setup. Don't underestimate the ops burden of the extra moving part unless a managed service (StreamNative Cloud, DataStax Astra Streaming) is on the table.
- **Key-shared subscriptions and ordering.** Shared/key-shared modes give you consumer-group-style scaling but the ordering guarantees are weaker and easier to misconfigure than Kafka's partition-per-consumer model — verify your ordering requirements against the specific subscription type before committing.
- **Smaller ecosystem than Kafka.** Fewer Kafka-Streams/ksqlDB-equivalent tools, fewer battle-tested Debezium-style CDC connectors, and a smaller hiring pool of engineers who already know it — factor migration and onboarding cost into the decision, not just the architecture win.
- **BookKeeper tuning is its own discipline.** Journal/ledger disk separation, ensemble/write-quorum/ack-quorum settings, and GC of old ledgers all need attention at scale; treating it like "just a broker" leads to surprises under load.
- **Schema evolution and compaction** are powerful but have edge cases (e.g., compacted topics plus tiered offload interacting) that are easy to hit only once you're already relying on them in production — test the exact combination of features you plan to use before depending on it.
