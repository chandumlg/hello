# CockroachDB

CockroachDB is a distributed, PostgreSQL-wire-compatible SQL database that gives you horizontal scale, automatic replication, and serializable transactions across nodes, zones, and regions, so a node, zone, or even region failure doesn't take your system down.

## Primary use cases

- **Highly available OLTP**: systems of record (payments, orders, identity, inventory) where downtime or data loss is unacceptable and you want survival of node/AZ failures without hand-built failover.
- **Horizontal write scaling**: workloads that outgrow a single Postgres primary. Add nodes and the cluster rebalances ranges automatically, instead of sharding at the application layer.
- **Multi-region applications**: data-locality and latency requirements (e.g. GDPR residency). Table localities such as `REGIONAL BY ROW` pin rows to a region, and `GLOBAL` tables give fast reads everywhere.
- **Zero-downtime operations**: rolling upgrades, online schema changes, and elastic scaling with no maintenance windows.

Teams typically adopt it when they have hit the ceiling of a single-primary Postgres/MySQL (write throughput, failover complexity, cross-region needs) but still want SQL, strong consistency, and ecosystem compatibility (Postgres drivers, ORMs). If a single well-provisioned Postgres with replicas handles your load, it is usually simpler and cheaper to stay there.

How it works: data is split into ~512 MiB *ranges*, each replicated (default 3x) via Raft consensus. One replica is the *leaseholder* that serves reads and coordinates writes. Any node can accept SQL connections and acts as a gateway, routing work to the relevant leaseholders. Transactions use MVCC with serializable isolation by default.

## Getting started

**1. Run a local 3-node cluster (Docker)**

```bash
docker network create roach
for i in 1 2 3; do
  docker run -d --name roach$i --hostname roach$i --net roach \
    cockroachdb/cockroach:latest start \
    --insecure --join=roach1,roach2,roach3 \
    $( [ $i = 1 ] && echo "-p 26257:26257 -p 8080:8080" )
done
docker exec -it roach1 ./cockroach init --insecure
docker exec -it roach1 ./cockroach sql --insecure
```

(Publish ports on the first container with `-p`; adjust as needed. `--insecure` is for local dev only.) The DB Console is at http://localhost:8080.

**2. Use it like Postgres, with a few distributed-aware tweaks**

```sql
CREATE DATABASE shop;
USE shop;

CREATE TABLE orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_id UUID NOT NULL,
  total DECIMAL(12,2) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  INDEX (customer_id, created_at DESC)
);

INSERT INTO orders (customer_id, total) VALUES (gen_random_uuid(), 42.50);
SHOW RANGES FROM TABLE orders;   -- see replicas and leaseholders
EXPLAIN ANALYZE SELECT * FROM orders WHERE customer_id = '...';
```

Any Postgres driver works: `postgresql://root@localhost:26257/shop?sslmode=disable`.

**3. Go multi-region**

```sql
ALTER DATABASE shop PRIMARY REGION "us-east1";
ALTER DATABASE shop ADD REGION "europe-west1";
ALTER DATABASE shop SURVIVE REGION FAILURE;

ALTER TABLE orders ADD COLUMN region crdb_internal_region NOT NULL DEFAULT default_to_database_primary_region(gateway_region());
ALTER TABLE orders SET LOCALITY REGIONAL BY ROW;
```

(Nodes must be started with `--locality=region=us-east1,zone=a`. Multi-region features have licensing implications, see below.)

## Common pitfalls

- **Sequential primary keys create hotspots**: `SERIAL`/auto-increment keys concentrate writes on one range. Prefer `UUID`, or hash-sharded indexes (`USING HASH`) for ordered keys such as timestamps.
- **Serialization retries are expected**: under contention, transactions can fail with SQLSTATE `40001`. Clients must retry (or use a driver/ORM retry helper). Keep transactions short and avoid long-running read-modify-write patterns.
- **Latency is bounded by consensus and distance**: cross-region writes pay round-trip time to a quorum. "Postgres but distributed" does not mean "Postgres-fast"; single-row point reads and writes are a few ms, but chatty multi-statement transactions add up. Batch statements, and use `RETURNING` and multi-row inserts.
- **Postgres compatibility is not total**: wire protocol and most SQL work, but some extensions, stored-procedure features, triggers, and `pg_catalog` details differ or are missing. Test your ORM and migrations early.
- **Schema changes are online but not free**: backfills consume resources and run as background jobs. Monitor them via `SHOW JOBS` and avoid mixing large schema changes with peak load.
- **Licensing**: CockroachDB moved from open source to the CockroachDB Software License (enterprise features, including many multi-region and backup capabilities, require a license; free tiers exist for small orgs). Check current terms before committing, and note that the managed CockroachDB Cloud is an alternative to self-hosting.
- **Operational sizing**: keep at least 3 nodes across failure domains, use fast local SSD/NVMe, set `--locality` correctly from day one, and watch range count, replication lag, and clock offset (nodes with badly skewed clocks will self-terminate to preserve consistency).
