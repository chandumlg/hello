# Trino

Trino (formerly PrestoSQL) is a distributed SQL query engine that lets you run fast, interactive SQL queries across data sitting in many different systems — S3/Iceberg tables, PostgreSQL, Kafka, Cassandra, Elasticsearch, and more — without first copying it all into one warehouse.

## Primary use cases

- **Federated analytics**: join a Postgres `orders` table with an Iceberg `events` table and a Kafka topic in a single query, no ETL required.
- **Data lake query layer**: sit on top of S3/GCS + Iceberg or Hive tables as the SQL engine for BI tools (Superset, Tableau, Looker) and ad-hoc analyst queries, avoiding vendor lock-in to a proprietary warehouse.
- **Decoupling storage from compute**: teams that want to keep data in open formats (Parquet/Iceberg on object storage) but still get warehouse-grade SQL performance, and scale query compute independently of storage.
- **Cross-team self-serve querying**: a platform team stands up one Trino cluster with connectors into every backing system so other teams query through a single endpoint instead of getting direct access to production databases.

A team typically adopts Trino once they have data spread across 3+ systems and are tired of writing one-off ETL jobs just to join it, or once storage costs push them toward object storage + open table formats but they still want fast interactive SQL.

## Basic usage

**1. Run a local Trino cluster with Docker:**
```bash
docker run -d --name trino -p 8080:8080 trinodb/trino
docker exec -it trino trino
```

**2. Query with the CLI once connected:**
```sql
SHOW CATALOGS;

SELECT nationkey, name
FROM tpch.tiny.nation
ORDER BY nationkey
LIMIT 5;
```

**3. Federate a join across two catalogs** (e.g. a Postgres catalog named `pg` and an Iceberg catalog named `iceberg`, both defined in `etc/catalog/*.properties`):
```sql
SELECT o.order_id, o.total, e.event_type
FROM pg.public.orders o
JOIN iceberg.analytics.events e
  ON o.order_id = e.order_id
WHERE e.event_date > DATE '2026-09-01';
```

Each catalog is just a properties file in `etc/catalog/`, e.g. `etc/catalog/iceberg.properties`:
```properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=http://rest-catalog:8181
```

## Common pitfalls

- **It's not a database.** Trino has no persistent storage of its own — every query re-reads from the underlying source. Repeated heavy queries against a slow connector (e.g. a live OLTP Postgres) can hammer that system; add caching or materialize hot tables into Iceberg/Parquet instead.
- **Cross-connector joins are only as fast as the slowest source.** A join between a partitioned Iceberg table and an unindexed Postgres table will bottleneck on the Postgres scan — push filters down where possible and check `EXPLAIN ANALYZE`.
- **Memory tuning matters early.** Default worker memory settings are conservative; large joins/aggregations without pushdown can blow past `query.max-memory-per-node` and fail rather than spill gracefully in older versions. Size the cluster and configure spill-to-disk before running production workloads.
- **Coordinator is a single point of failure** for query planning — it doesn't hold data, but if it goes down no new queries can start. Plan for HA/restart automation in production.
- **Connector semantics differ from source engines.** Things like transaction isolation, `UPDATE`/`DELETE` support, and data type coercion vary a lot per connector — don't assume Postgres-via-Trino behaves identically to native Postgres.
