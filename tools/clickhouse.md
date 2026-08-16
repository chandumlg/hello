# ClickHouse

ClickHouse is a column-oriented SQL database built for running aggregation queries over billions of rows in sub-second time, solving the problem that row-oriented databases (Postgres, MySQL) fall over when you need to scan and aggregate massive, append-heavy datasets like logs, events, or metrics.

## Primary use cases

ClickHouse earns its place when a team has a firehose of structured, mostly-immutable data and needs interactive-speed answers over it:

- **Product & event analytics** — funnels, retention, and ad-hoc segmentation over billions of clickstream/event rows (companies like Cloudflare, Uber, and PostHog run it at this scale).
- **Observability backends** — storing logs and traces where you need `GROUP BY` and `WHERE` over huge time ranges (this is what powers systems like Signoz and parts of Sentry).
- **Real-time dashboards** — customer-facing or internal dashboards that need to aggregate fresh data (seconds old) without pre-computing every rollup.
- **Time-series / metrics at scale** — an alternative to Prometheus' storage layer when cardinality or retention outgrows TSDB-native stores.

A team typically reaches for it once Postgres aggregate queries start taking tens of seconds to minutes on a table with hundreds of millions+ rows, and pre-aggregating (rollup tables, materialized views in the OLTP db) has become its own maintenance burden. It is a poor fit for OLTP workloads — frequent single-row updates/deletes and highly transactional access patterns fight its columnar, immutable-part storage model.

## Basic usage

**1. Run it locally with Docker:**

```bash
docker run -d --name clickhouse-server -p 8123:8123 -p 9000:9000 \
  clickhouse/clickhouse-server

# connect with the native client
docker exec -it clickhouse-server clickhouse-client
```

**2. Create a table with a `MergeTree` engine (the workhorse storage engine) and load data:**

```sql
CREATE TABLE events (
    event_time DateTime,
    user_id    UInt64,
    event_type LowCardinality(String),
    value      Float64
)
ENGINE = MergeTree
ORDER BY (event_type, event_time);

INSERT INTO events VALUES
    (now(), 1001, 'page_view', 1),
    (now(), 1002, 'purchase', 49.99);
```

`ORDER BY` here defines the primary/sort key ClickHouse physically sorts data by on disk — it's the single most important tuning decision, since it determines which queries get fast, index-assisted scans.

**3. Run an aggregation that would choke a row store:**

```sql
SELECT
    event_type,
    toStartOfHour(event_time) AS hour,
    count() AS events,
    sum(value) AS total_value
FROM events
WHERE event_time >= now() - INTERVAL 7 DAY
GROUP BY event_type, hour
ORDER BY hour;
```

Over hundreds of millions of rows this typically returns in well under a second, because ClickHouse only reads the columns referenced (`event_time`, `event_type`, `value`) rather than full rows, and processes them in vectorized batches.

## Common pitfalls

- **Treating it like an OLTP database.** `UPDATE`/`DELETE` exist (`ALTER TABLE ... UPDATE/DELETE`) but are asynchronous, heavyweight "mutations" — not point updates. Frequent single-row writes or updates will degrade performance badly; batch inserts (thousands of rows per `INSERT`, not one row at a time).
- **Small, frequent inserts create "part" explosion.** Each insert creates a new on-disk part; too many small inserts overwhelm the background merge process and trigger `Too many parts` errors. Buffer and batch writes (or use a buffer table / tool like `clickhouse-kafka` connector) instead of inserting row-by-row from an application.
- **Picking the wrong `ORDER BY`/sort key.** Unlike an OLTP index, this is chosen once per table and is expensive to change; queries that don't filter on a prefix of the sort key fall back to full scans. Design it around your most common `WHERE` clauses up front.
- **No real support for transactions or foreign keys.** Joins work but are not the primary design center — denormalizing (wide tables) is the idiomatic ClickHouse pattern, which is a mindset shift for teams coming from normalized relational schemas.
- **Replication/sharding is manual to reason about.** `ReplicatedMergeTree` and distributed tables give you HA and horizontal scale, but they require ZooKeeper/ClickHouse Keeper and deliberate cluster topology design — it's not automatic like a managed cloud database, unless you use a managed offering (ClickHouse Cloud, Altinity).
