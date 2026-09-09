# Apache Iceberg

## What it is and what problem it solves

Apache Iceberg is an open table format that brings database-like reliability — ACID transactions, schema evolution, time travel, and safe concurrent writes — to tables of files sitting in a data lake (S3, GCS, ADLS, HDFS), so that many different engines (Spark, Flink, Trino, Snowflake, DuckDB, etc.) can read and write the same table without corrupting it or requiring a costly full rewrite when the schema or partitioning changes.

## Primary use cases and when to adopt it

Iceberg matters once "just files in object storage" stops being good enough:

- **A lakehouse serving multiple compute engines.** Instead of locking data into a single warehouse (Snowflake, BigQuery), teams land data as Iceberg tables in cheap object storage and let Spark, Flink, Trino, and Snowflake all query the same physical data through the Iceberg catalog. No copies, no engine lock-in.
- **Tables that outgrow Hive-style partitioning.** Classic Hive tables bake partition columns into directory paths (`/year=2024/month=01/`), so changing the partitioning scheme means rewriting the whole table. Iceberg tracks partitioning as metadata ("hidden partitioning") and lets you evolve it without touching existing data.
- **Correctness under concurrent writers.** Multiple pipelines (a streaming Flink job and a nightly Spark batch job, say) writing to the same table need atomic, isolated commits. Iceberg's snapshot-based commit protocol gives that instead of ad hoc file-listing races.
- **Auditing, rollback, and reproducible ML/analytics.** Every write creates a new immutable snapshot, so you can time-travel to a prior state, diff two snapshots, or roll back a bad backfill — valuable for both debugging pipelines and reproducing a training dataset.
- **Schema evolution at scale.** Adding, renaming, or reordering columns, or widening types, is a metadata-only operation — no rewriting petabytes of Parquet files.

A team typically adopts Iceberg when it has (or is building) a multi-engine data platform and existing Hive tables or raw Parquet dumps have started causing partition-management pain, slow "read then overwrite the whole partition" writes, or duplicate storage across warehouses.

## Basic usage examples

**1. Create and query an Iceberg table from Spark SQL** (via a configured Iceberg catalog, e.g. backed by a Hive metastore, AWS Glue, or the REST catalog):

```sql
CREATE TABLE local.db.events (
  id bigint,
  event_type string,
  event_time timestamp,
  payload string
)
USING iceberg
PARTITIONED BY (days(event_time));

INSERT INTO local.db.events VALUES
  (1, 'click', current_timestamp(), '{}');

SELECT * FROM local.db.events WHERE event_type = 'click';
```

Note `PARTITIONED BY (days(event_time))` — a transform, not a literal column. Iceberg computes the partition value and hides it from query predicates, so you never write `WHERE year=2024 AND month=1` by hand.

**2. Evolve schema and partitioning without rewriting data:**

```sql
ALTER TABLE local.db.events ADD COLUMN user_id bigint;
ALTER TABLE local.db.events REPLACE PARTITION FIELD days(event_time) WITH hours(event_time);
```

Both are metadata-only changes; existing data files are untouched, and old snapshots remain queryable under the old partition spec.

**3. Time travel to inspect or roll back to a previous snapshot:**

```sql
-- list the history of commits
SELECT * FROM local.db.events.snapshots;

-- query the table as of a specific snapshot or timestamp
SELECT * FROM local.db.events VERSION AS OF 3821550127947089009;
SELECT * FROM local.db.events TIMESTAMP AS OF '2026-09-01 00:00:00';

-- roll the table back to a known-good snapshot
CALL local.system.rollback_to_snapshot('db.events', 3821550127947089009);
```

## Common pitfalls

- **The catalog is not optional.** Iceberg's transactional guarantees come from atomic pointer swaps in a catalog (Hive metastore, AWS Glue, Nessie, or the REST catalog spec) — pointing multiple engines at the same S3 prefix without a shared catalog reintroduces the races Iceberg exists to prevent.
- **Small-file and metadata bloat from streaming writers.** High-frequency micro-batch or streaming commits (e.g. from Flink) create many small data and manifest files, which slows planning and reads. Schedule regular `rewrite_data_files` / `rewrite_manifests` compaction, and periodic `expire_snapshots` to reclaim storage and shrink the snapshot log.
- **Not every engine speaks every Iceberg feature at the same version.** Row-level deletes, branching/tagging, and the newest table spec version aren't uniformly supported across Spark, Flink, Trino, and vendor engines — check the specific engine/connector version against the Iceberg spec version before relying on a feature in a multi-engine setup.
- **Hidden partitioning still needs a sane transform choice.** Over-partitioning (e.g. `hours()` on a low-volume table) produces excessive small files just as Hive-style over-partitioning did; the hidden-partition mechanism removes query-syntax pain, not the need to size partitions sensibly.
- **Orphan files after failed or abandoned writes.** Failed jobs can leave data files that no snapshot references. Run `remove_orphan_files` periodically rather than manually deleting from the underlying object store, which can race with concurrent readers.
