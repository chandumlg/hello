# Vitess

Vitess is a CNCF-graduated database clustering system that adds horizontal sharding, connection pooling, and online schema management on top of MySQL, so you can scale a relational database far past a single primary without rewriting your application.

## Primary use cases

- **Horizontally sharding MySQL**: split a large keyspace across many shards by a "vindex" (sharding key function) while apps keep speaking the MySQL protocol to a stateless router, `vtgate`.
- **Connection scaling**: `vtgate`/`vttablet` multiplex thousands of app connections onto a small pool of MySQL connections, protecting the database from connection storms (common with serverless or many-replica microservices).
- **Online schema changes**: managed, throttled, revertible `ALTER TABLE` via online DDL (`ghost`/`vitess` strategies) without long locks.
- **Live resharding and data migration**: `MoveTables` and `Reshard` workflows copy data, keep it in sync via the binlog, let you verify (`VDiff`), then cut traffic over with a reversible switch.
- **Query safety**: query consolidation, row limits, and blocklists in `vttablet` stop runaway queries from taking down the primary.

Teams adopt Vitess when a single MySQL primary is hitting limits (storage, write throughput, connection count, painful migrations) and they want to stay on MySQL semantics rather than migrate to a different database. Fits especially well on Kubernetes (via the Vitess operator) or with PlanetScale as a managed option. It is overkill for a database that comfortably fits on one primary plus replicas.

Architecture in brief: apps -> **vtgate** (stateless proxy/query planner) -> **vttablet** (sidecar next to each mysqld) -> **MySQL**. Topology metadata (keyspaces, shards, tablets) lives in a topo server (etcd, ZooKeeper, or Consul). **vtctld** is the control plane; `vtorc` handles automated failover.

## Basic usage

**1. Try it locally (the examples cluster)**

```bash
git clone https://github.com/vitessio/vitess && cd vitess/examples/local
./101_initial_cluster.sh        # starts topo, vtctld, mysqld, vttablets, vtgate
mysql -h 127.0.0.1 -P 15306     # connect to vtgate like any MySQL server
```

**2. Define a sharded keyspace with a VSchema**

```json
{
  "sharded": true,
  "vindexes": { "hash": { "type": "hash" } },
  "tables": {
    "orders": {
      "column_vindexes": [{ "column": "customer_id", "name": "hash" }]
    }
  }
}
```

```bash
vtctldclient ApplyVSchema --vschema-file vschema_commerce.json commerce
```

Rows for the same `customer_id` land on the same shard; queries that include it route to one shard, others scatter to all.

**3. Move data online with a workflow**

```bash
# Copy tables to a new keyspace while the source keeps serving traffic
vtctldclient MoveTables --workflow commerce2customer --target-keyspace customer \
  create --source-keyspace commerce --tables "customer,corder"

vtctldclient MoveTables --workflow commerce2customer --target-keyspace customer show
vtctldclient VDiff --workflow commerce2customer --target-keyspace customer create
vtctldclient MoveTables --workflow commerce2customer --target-keyspace customer switchtraffic
# Roll back if needed:  ... reversetraffic    Finish with: ... complete
```

Online DDL is similar in spirit:

```sql
SET @@ddl_strategy = 'vitess';
ALTER TABLE orders ADD COLUMN notes TEXT;   -- returns a migration UUID
SHOW VITESS_MIGRATIONS;
```

## Common pitfalls

- **Choosing the sharding key is the real decision.** A poor primary vindex causes hot shards or leaves most queries scatter-gathering. Pick a high-cardinality key present in nearly every hot-path query, and plan for how you will reshard later.
- **Scatter queries are expensive.** Queries without the sharding key hit every shard; latency becomes the slowest shard and cost grows with shard count. Watch `vtgate` query plans (`VEXPLAIN`, `EXPLAIN FORMAT=vtexplain`).
- **MySQL compatibility is high, not total.** Cross-shard joins, subqueries, and aggregations are planned in vtgate and some shapes are unsupported or slow; cross-shard transactions default to best-effort unless you opt into 2PC (with a latency cost). Test your real query mix early.
- **Unique constraints and auto-increment.** Global uniqueness across shards needs lookup vindexes or sequences (a separate unsharded table); naive `AUTO_INCREMENT` collides across shards.
- **Operational surface area.** You now run topo, vtctld, vtgate, vttablets, and vtorc. Invest in upgrade runbooks (Vitess is strict about version skew order), backups per shard, and monitoring of replication lag; consider the operator or a managed offering.
- **Resharding isn't free.** It needs spare capacity for the copy phase and binlog retention long enough for catch-up; always run `VDiff` before `switchtraffic`, and keep reverse replication until you're confident.
- **Not a drop-in for Postgres users.** Vitess is MySQL-only; if you're on Postgres look at Citus or Neon-style alternatives instead.
