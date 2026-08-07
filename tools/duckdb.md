# DuckDB

## What is it?

DuckDB is an embedded, in-process analytical (OLAP) SQL database that lets you run fast, columnar SQL queries directly against local files (CSV, Parquet, JSON) or in-memory data — without standing up a server, cluster, or warehouse.

## Why teams adopt it

DuckDB fills the gap between "just use pandas" and "spin up a Postgres/Snowflake/BigQuery instance." It's the tool staff and AI engineers reach for when they need real SQL analytics but the data fits on one machine (or one laptop) and provisioning infrastructure would be overkill.

Common adoption triggers:

- **Ad-hoc data exploration**: querying gigabytes of Parquet/CSV logs or event data without loading everything into memory or setting up a database.
- **Data pipeline / ETL glue**: DuckDB is often used as a lightweight transformation engine inside Python or Rust pipelines — read from S3, transform with SQL, write back to Parquet.
- **Local development and testing**: teams that run Postgres/Snowflake in production sometimes use DuckDB locally for fast, dependency-free unit and integration tests of SQL logic.
- **Analytics inside applications**: embedding DuckDB directly in a backend service or notebook to serve dashboards or reports without a network hop to an external warehouse.
- **ML/AI feature engineering**: AI engineers use it to quickly aggregate, join, and filter training data stored as Parquet/CSV before feeding it into a model, since it's dramatically faster than pandas for large tabular joins/group-bys and doesn't require a Spark cluster.

It's not a replacement for a distributed warehouse when data no longer fits on one node, and it's not built for high-concurrency multi-user OLTP workloads — that's Postgres/MySQL territory.

## Basic usage

**1. Query a CSV or Parquet file directly, no loading step required (CLI or Python):**

```bash
# CLI: install with `curl https://install.duckdb.org | sh`, then:
duckdb -c "SELECT category, COUNT(*), AVG(price) FROM 'sales.parquet' GROUP BY category ORDER BY 2 DESC"
```

```python
import duckdb

# Query a file glob across many Parquet files as if it were one table
df = duckdb.sql("""
    SELECT user_id, COUNT(*) AS events
    FROM 's3://my-bucket/events/*.parquet'
    GROUP BY user_id
    ORDER BY events DESC
    LIMIT 10
""").df()
```

**2. Query a pandas DataFrame with SQL instead of pandas syntax:**

```python
import duckdb
import pandas as pd

orders = pd.read_csv("orders.csv")

result = duckdb.sql("""
    SELECT customer_id, SUM(amount) AS total_spent
    FROM orders
    WHERE order_date >= '2026-01-01'
    GROUP BY customer_id
    HAVING SUM(amount) > 1000
""").df()
```

**3. Persist to an on-disk database file and build a small local warehouse:**

```python
import duckdb

con = duckdb.connect("analytics.duckdb")
con.execute("CREATE TABLE events AS SELECT * FROM read_parquet('raw/*.parquet')")
con.execute("CREATE INDEX idx_user ON events(user_id)")
print(con.execute("SELECT COUNT(*) FROM events").fetchone())
```

## Pitfalls to watch out for

- **Single-writer, embedded model**: only one process can hold a read-write connection to a DuckDB file at a time. It's not designed for concurrent multi-user writes — don't treat it as a shared production OLTP database.
- **Memory usage on large joins/aggregations**: DuckDB spills to disk but can still surprise you with high memory consumption on wide joins over very large datasets; watch it on memory-constrained containers (e.g., CI runners, Lambda).
- **Not a replacement for a distributed engine**: once data meaningfully exceeds single-node RAM/disk, you need Spark, Trino, BigQuery, or similar — DuckDB scales vertically, not horizontally.
- **Extension loading**: features like S3/httpfs access, JSON support, or full-text search require explicitly `INSTALL`/`LOAD`-ing extensions; forgetting this is a common source of "why doesn't this work" confusion, especially in fresh environments or Docker images without network access to fetch extensions.
- **File locking surprises**: if a DuckDB file is left open by a crashed process or another tool, you may hit lock errors on reconnect — closing connections explicitly avoids this in long-running services.
- **Version compatibility of on-disk files**: the on-disk storage format has changed across major versions; don't assume a `.duckdb` file written by one version opens cleanly in an older/newer client without checking release notes.
