# DuckDB

DuckDB is an embedded, in-process SQL database engine built for fast analytical queries over local or remote data — think "SQLite, but column-oriented and built for analytics" — so you can run OLAP-style aggregations directly against CSV, Parquet, or JSON files without spinning up a database server or a cluster.

## Why teams adopt it

DuckDB fills the gap between "a spreadsheet" and "a full data warehouse." Staff and platform engineers reach for it when they need to explore, transform, or validate large local datasets (millions to billions of rows) without provisioning infrastructure. AI/ML engineers use it to slice and dice training data, feature stores, or evaluation logs stored as Parquet files. It's also increasingly used as an embedded query engine inside applications and CLI tools, and as the execution layer in data pipelines (dbt now supports it as a target) or notebooks where spinning up Spark or a Postgres instance would be overkill.

Common adoption triggers:
- You're doing exploratory analysis on Parquet/CSV files sitting in S3 or on disk and pandas is too slow or memory-hungry.
- You want SQL analytics inside a Python/CLI/Node application without a network round-trip to an external DB.
- You're building a local-first analytics tool, a CI data-validation step, or a lightweight ETL job and don't want the operational overhead of Postgres/Spark.
- You want a fast, zero-dependency way to query data lake files (including directly over S3/HTTP via extensions) for ad hoc investigation.

## Basic usage

**1. CLI — query a Parquet or CSV file directly, no schema setup required:**
```bash
duckdb -c "SELECT status, COUNT(*) FROM 'events.parquet' GROUP BY status ORDER BY 2 DESC"
```

**2. Python — query a pandas DataFrame or file with SQL, get a DataFrame back:**
```python
import duckdb

df = duckdb.sql("SELECT * FROM 'logs/*.parquet' WHERE level = 'ERROR'").df()

# Or query an existing in-memory DataFrame directly
duckdb.sql("SELECT user_id, COUNT(*) FROM my_dataframe GROUP BY user_id")
```

**3. Persistent database file + querying remote data via an extension:**
```sql
-- Open (or create) a persistent DB instead of the default in-memory one
ATTACH 'analytics.duckdb' AS db;

-- Query a file straight from S3 (httpfs extension autoloads in recent versions)
SELECT * FROM read_parquet('s3://my-bucket/data/*.parquet') LIMIT 10;
```

## Pitfalls to watch out for

- **It's single-node.** DuckDB scales vertically (it's very good at using all your cores and RAM efficiently) but it isn't a distributed engine — if your dataset genuinely exceeds one machine's memory/disk, you still need Spark, BigQuery, Snowflake, etc.
- **Concurrent writes are limited.** One process can write to a DuckDB file at a time; it's designed for single-writer, embedded use, not as a shared multi-writer application database like Postgres.
- **In-memory by default.** Running `duckdb` with no filename creates a transient in-memory database — data vanishes when the process exits unless you explicitly attach a file.
- **Extension loading**: features like S3/HTTP access (`httpfs`), Postgres scanning, or full-text search live in extensions that may need an explicit `INSTALL`/`LOAD` depending on your version and environment (some auto-install, some don't in restricted/offline environments).
- **Version churn**: DuckDB is developed very actively; syntax and extension behavior can shift between minor versions, so pin a version in production pipelines rather than always pulling latest.
