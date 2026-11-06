# Delta Lake

Delta Lake is an open-source storage layer that adds ACID transactions, schema enforcement, and time travel to plain Parquet files on object storage (S3, GCS, ADLS), so a data lake behaves like a reliable transactional table store instead of a pile of files.

## Why teams adopt it

A raw Parquet "lake" breaks down as soon as there are concurrent writers, failed jobs, or a need to update or delete rows: readers see half-written output, a crashed job leaves duplicates, and GDPR deletes mean rewriting partitions by hand. Delta Lake solves this with a transaction log (`_delta_log/`) of JSON commit files and periodic Parquet checkpoints, sitting next to the data files. Every write is an atomic commit to that log, and readers resolve a consistent snapshot from it using optimistic concurrency control.

Typical use cases:
- **Lakehouse tables** that need `MERGE`/`UPDATE`/`DELETE` (CDC ingestion from Debezium/Kafka, upserts, GDPR erasure).
- **Reproducible ML and analytics**: time travel lets you pin a training run or a report to an exact table version.
- **Unified batch + streaming**: the same table can be a Spark Structured Streaming sink and a batch source.
- **Auditing and rollback**: `DESCRIBE HISTORY` and `RESTORE` give you an operational undo button.

Adopt it when you already have a Parquet lake and are hitting correctness or maintenance pain. If you're all-in on Trino/Flink or want hidden partitioning and broader engine neutrality, compare it with Apache Iceberg (already in this directory). Delta has the deepest Spark/Databricks integration and a growing engine-neutral ecosystem via `delta-rs` and Delta UniForm.

## Basic usage

**1. Spark (PySpark): create, upsert, time travel**
```python
from delta import configure_spark_with_delta_pip
from delta.tables import DeltaTable
import pyspark

builder = (pyspark.sql.SparkSession.builder.appName("demo")
    .config("spark.sql.extensions", "io.delta.sql.DeltaSparkSessionExtension")
    .config("spark.sql.catalog.spark_catalog", "org.apache.spark.sql.delta.catalog.DeltaCatalog"))
spark = configure_spark_with_delta_pip(builder).getOrCreate()

spark.range(5).write.format("delta").save("/tmp/events")

updates = spark.createDataFrame([(3,), (10,)], ["id"])
(DeltaTable.forPath(spark, "/tmp/events").alias("t")
    .merge(updates.alias("u"), "t.id = u.id")
    .whenNotMatchedInsertAll()
    .execute())

# Read the table as it was at version 0
spark.read.format("delta").option("versionAsOf", 0).load("/tmp/events").show()
```

**2. No JVM needed: `deltalake` (delta-rs) from Python**
```python
# pip install deltalake pandas
import pandas as pd
from deltalake import DeltaTable, write_deltalake

write_deltalake("./orders", pd.DataFrame({"id": [1, 2], "amt": [10.0, 20.0]}))
write_deltalake("./orders", pd.DataFrame({"id": [3], "amt": [5.0]}), mode="append")

dt = DeltaTable("./orders")
print(dt.version(), dt.history())
print(dt.to_pandas())
```
This is handy for services, CI checks, and lightweight pipelines; DuckDB, Polars, and DataFusion can also read these tables.

**3. Operations: history, restore, compaction, cleanup (Spark SQL)**
```sql
DESCRIBE HISTORY delta.`/tmp/events`;
RESTORE TABLE delta.`/tmp/events` TO VERSION AS OF 0;
OPTIMIZE delta.`/tmp/events`;          -- compact small files
VACUUM delta.`/tmp/events` RETAIN 168 HOURS;  -- delete unreferenced files
```

## Common pitfalls

- **The small-files problem**: frequent streaming or micro-batch appends produce thousands of tiny files. Schedule `OPTIMIZE` (and consider liquid clustering or Z-ordering) and enable optimized writes/auto compaction where available.
- **`VACUUM` vs. time travel**: vacuum permanently deletes old files, so time travel and `RESTORE` stop working past the retention window (default 7 days). Don't shorten retention below the longest-running reader or stream, and never disable the safety check casually.
- **S3 concurrent writes**: S3 historically lacked atomic put-if-absent, so multi-writer setups needed DynamoDB-based log store coordination. Check the current guidance for your connector version, or you risk lost commits.
- **Schema evolution is opt-in**: writes with mismatched schemas fail by design. Use `mergeSchema` deliberately, and remember that column mapping is required for renames/drops without rewrites.
- **Protocol/feature upgrades are one-way**: enabling features like deletion vectors or column mapping bumps reader/writer protocol versions, and older clients (including older delta-rs or engines) can no longer read the table. Pin versions across your engines before upgrading.
- **Conflicts under concurrency**: optimistic concurrency means overlapping `MERGE`/`UPDATE` jobs can fail with `ConcurrentModificationException`. Partition or predicate your writes so jobs touch disjoint files, and add retries.
- **Log growth**: very long histories rely on checkpoints; avoid tweaking checkpoint/log retention unless you understand the effect on time travel.
