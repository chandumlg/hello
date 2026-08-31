# Debezium

Debezium is an open-source platform that captures every row-level change happening inside a database and streams it out as an ordered, durable event log in near real time, so downstream systems never have to poll the database or trust brittle dual-writes to stay in sync.

## What problem it solves

Most systems eventually need other services to know "this row changed" — a search index that has to mirror Postgres, a cache that needs invalidating, an analytics warehouse that wants a live copy of production data, or a saga that needs to react when an order's status flips. The naive approaches are all leaky: polling with `updated_at` timestamps misses deletes and hard-to-order updates, and "write to the DB, then publish an event" is not atomic — a crash between the two leaves the event log and the database disagreeing forever (the classic dual-write problem).

Debezium solves this with change data capture (CDC): it reads the database's own replication/transaction log (Postgres logical replication, MySQL binlog, MongoDB oplog, SQL Server CDC tables, etc.) — the same mechanism the database uses for its own replicas — and turns each committed change into an event. Because it reads what the database already durably wrote, there's no dual-write, no missed deletes, and events appear in true commit order.

## Primary use cases and when to adopt it

- **Outbox pattern / reliable event publishing**: a service writes business data and an "outbox" row in the same local transaction; Debezium tails the outbox table and publishes to Kafka, giving exactly the atomicity a distributed transaction would have promised, without one.
- **Keeping derived stores in sync**: propagating a system-of-record database into Elasticsearch/OpenSearch, a cache (Redis), a data warehouse, or a lakehouse table (via Kafka Connect sinks or Debezium Server) without batch ETL lag.
- **Strangler-fig migrations**: streaming changes from a legacy monolith's database while a new service is built alongside it, so both stay consistent during a gradual cutover.
- **Cross-region or cross-database replication** with transformation in between, when native database replication isn't flexible enough.

Adopt Debezium once "near-real-time, ordered, exactly-once-ish downstream sync" actually matters — if a nightly batch job or a simple cron poll is good enough, the operational cost of running Kafka Connect plus Debezium isn't worth it yet.

## Basic usage examples

**1. Enable logical replication on Postgres** (source database prerequisite):

```sql
ALTER SYSTEM SET wal_level = logical;
-- restart Postgres, then:
CREATE PUBLICATION dbz_publication FOR ALL TABLES;
```

**2. Register a Postgres connector with Kafka Connect** (the most common deployment: Debezium as a Kafka Connect plugin):

```bash
curl -X POST http://connect:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "orders-connector",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres",
      "database.dbname": "shop",
      "database.user": "debezium",
      "database.password": "secret",
      "topic.prefix": "shop",
      "table.include.list": "public.orders",
      "plugin.name": "pgoutput"
    }
  }'
```

**3. Consume the change events** — each Kafka message contains a `before`/`after` payload plus an `op` field (`c`=create, `u`=update, `d`=delete, `r`=initial snapshot read):

```bash
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic shop.public.orders --from-beginning
```

For lighter setups without Kafka, **Debezium Server** ships events directly to Kinesis, Pulsar, Google Pub/Sub, or a plain HTTP sink, cutting out Kafka Connect entirely.

## Common pitfalls

- **Schema changes break consumers silently.** Adding/dropping a column changes the event shape; pair Debezium with a schema registry (Avro/Protobuf + Confluent Schema Registry) so consumers get compatibility checks instead of runtime surprises.
- **Initial snapshots on large tables are expensive.** The first run has to snapshot existing rows before streaming new changes, which can lock or heavily load a large production table — use incremental snapshotting (supported since Debezium 1.6+) to avoid long locks.
- **Replication slots need care on Postgres.** If a connector goes down and nobody notices, the replication slot keeps accumulating WAL on the primary until disk fills up. Monitor slot lag, not just connector uptime.
- **Ordering is per-partition, not global.** Events for a single row/key are ordered, but if you fan out across Kafka partitions, don't assume global ordering across different tables or keys.
- **Tombstones and deletes need explicit handling.** Downstream consumers (especially compacted-topic sinks) must handle delete events and Kafka's tombstone records, or "deleted" rows can linger forever in derived stores.
- **It captures committed changes only** — long-running transactions delay when their changes appear, which can surprise teams expecting sub-second latency from a transaction that took minutes to commit.
