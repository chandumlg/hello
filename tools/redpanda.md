# Redpanda

Redpanda is a Kafka-API-compatible streaming platform written in C++ that solves the operational weight of running Apache Kafka — no ZooKeeper, no JVM, no separate broker/controller tuning — while remaining a drop-in replacement for existing Kafka clients and tooling.

## Why teams adopt it

Staff and platform engineers reach for Redpanda when they want Kafka's event-streaming semantics (topics, partitions, consumer groups, exactly-once-ish processing via idempotent producers) without the multi-process, multi-JVM operational surface that makes Kafka expensive to run well at small-to-mid scale. Redpanda ships as a single statically linked binary per broker, uses the Raft consensus protocol internally instead of ZooKeeper or a separate KRaft controller quorum, and is engineered around a thread-per-core architecture that squeezes far more throughput out of the same hardware — teams typically report needing a fraction of the nodes to hit the same latency and throughput targets.

Common adoption triggers:
- Migrating off Kafka to cut broker count, JVM GC pauses, and ZooKeeper operational burden, without rewriting producers/consumers — any Kafka client library, Kafka Connect connector, or tool that speaks the Kafka wire protocol works against Redpanda unmodified.
- Running event streaming in resource-constrained environments (edge, on-prem, cost-sensitive cloud) where Kafka's minimum viable footprint (3+ ZooKeeper nodes plus 3+ brokers) is disproportionate to the actual event volume.
- Teams that want built-in HTTP/schema-registry-compatible tooling (Redpanda ships its own schema registry and REST proxy) without standing up Confluent's separate components.
- New streaming platforms being built greenfield where Kafka-protocol compatibility is valued for ecosystem reach (Debezium, Kafka Connect, ksqlDB-like tooling) but the team doesn't want to inherit Kafka's operational lineage.
- Low-latency, tail-latency-sensitive workloads (trading systems, real-time bidding, gaming) where Redpanda's C++ implementation avoids JVM GC-induced latency spikes.

## Basic usage

**1. Run a single-node cluster (Docker, for local dev):**
```bash
docker run -d --name redpanda -p 9092:9092 -p 9644:9644 \
  docker.redpanda.com/redpandadata/redpanda:latest \
  redpanda start --smp 1 --overprovisioned --node-id 0 \
  --kafka-addr PLAINTEXT://0.0.0.0:9092 \
  --advertise-kafka-addr PLAINTEXT://localhost:9092
```

**2. Use `rpk`, Redpanda's CLI, to create a topic and produce/consume — the same operations Kafka's `kafka-topics.sh`/`kafka-console-producer.sh` do, in one tool:**
```bash
# create a topic with 6 partitions
rpk topic create orders --partitions 6 --replicas 1

# produce a message
echo '{"order_id": "1234"}' | rpk topic produce orders

# consume from the beginning
rpk topic consume orders --offset start
```

**3. Point any existing Kafka client at it — nothing changes but the bootstrap address (Python example):**
```python
from kafka import KafkaProducer, KafkaConsumer

producer = KafkaProducer(bootstrap_servers="localhost:9092")
producer.send("orders", b'{"order_id": "1234"}')
producer.flush()

consumer = KafkaConsumer("orders", bootstrap_servers="localhost:9092",
                          auto_offset_reset="earliest", group_id="order-processor")
for msg in consumer:
    process(msg.value)
```

## Common pitfalls

- **`--overprovisioned --smp 1` is for laptops, not production.** Redpanda's thread-per-core design wants dedicated CPU cores and pinned memory; running it under-resourced (shared cores, small `--smp`) in a "just testing prod-like behavior" environment gives misleadingly poor throughput numbers and hides real capacity planning issues.
- **Kafka-protocol compatibility is broad but not 100%.** Most client libraries and Kafka Connect connectors work unmodified, but some Confluent-proprietary extensions (certain licensed connectors, some Confluent Cloud-specific APIs) aren't supported — verify any exotic tooling against Redpanda before committing to a migration, not after.
- **Raft-based replication still needs an odd node count for quorum**, same constraint as any Raft or ZooKeeper-based system — a 2-node cluster doesn't give you the availability a 3-node one does, and undersizing here reintroduces the exact operational fragility teams migrate to Redpanda to avoid.
- **Tiered storage (offloading old log segments to S3/object storage) is a separate feature to opt into**, not the default — teams expecting "infinite retention for free" out of the box are surprised when local disk fills up because tiered storage was never configured.
- **Schema registry and REST proxy are separate listeners/ports that need their own configuration** — teams porting a Confluent Schema Registry setup sometimes assume Redpanda's built-in registry is a transparent proxy to their existing one, when it's actually Redpanda's own compatible implementation that needs its data migrated or repointed.
- **Community edition licensing has real limits at scale** (e.g., features like certain enterprise RBAC/audit capabilities, and historically some scale thresholds have been Enterprise-gated) — check current licensing terms against cluster size and required features before assuming full feature parity with Kafka's ecosystem is free indefinitely.
