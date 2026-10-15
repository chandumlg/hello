# Apache DataFusion

Apache DataFusion is an embeddable, extensible SQL and DataFrame query engine written in Rust on top of Apache Arrow, letting you build analytical databases and data pipelines without writing a query planner and vectorized executor from scratch.

## Primary use cases

- **Embedding a query engine in your product**: observability backends, time-series stores, and data platforms (InfluxDB 3, for example) use it as the SQL layer over their own storage format rather than writing a parser, optimizer, and executor.
- **Querying files in place**: run SQL over Parquet, CSV, JSON, and Avro in local disk or object storage (S3/GCS/Azure) via the `object_store` crate, with projection and predicate pushdown into Parquet.
- **Custom data sources and functions**: implement `TableProvider` to expose any system (an API, a log segment format, a key-value store) as a SQL table, and register scalar, aggregate, window, or table UDFs.
- **Pipeline and ETL building blocks**: the DataFrame API and logical/physical plan abstractions suit transformation services, and Ballista or DataFusion Comet (Spark acceleration) distribute or accelerate it.
- **Ad hoc analytics**: `datafusion-cli` is a quick local SQL shell, and the Python bindings offer a DuckDB-like experience with different extension points.

Teams adopt it when they are building a data system rather than just using one. If you only want fast local analytics, DuckDB is usually the simpler choice. If you need to own the storage layer, plan optimization rules, or deployment shape, and want to write it in Rust, DataFusion is the building kit.

How it works: SQL or DataFrame calls become a logical plan, which is optimized by rule-based passes (predicate pushdown, projection pruning, constant folding, join reordering). It is then lowered to a physical plan of streaming, multi-threaded operators that process Arrow `RecordBatch`es, executed on Tokio.

## Basic usage

**1. Interactive SQL over files with the CLI**

```bash
cargo install datafusion-cli      # or: brew install datafusion

datafusion-cli
> CREATE EXTERNAL TABLE trips STORED AS PARQUET LOCATION 's3://my-bucket/trips/';
> SELECT passenger_count, avg(fare_amount) FROM trips GROUP BY 1 ORDER BY 1;
> EXPLAIN SELECT * FROM trips WHERE fare_amount > 100;   -- inspect pushdown
```

**2. Embedding in Rust**

```rust
use datafusion::prelude::*;

#[tokio::main]
async fn main() -> datafusion::error::Result<()> {
    let ctx = SessionContext::new();
    ctx.register_parquet("events", "data/events/", ParquetReadOptions::default()).await?;

    let df = ctx
        .sql("SELECT user_id, count(*) AS n FROM events WHERE ts > now() - interval '1 day' GROUP BY user_id")
        .await?;
    df.show().await?;

    // Equivalent DataFrame API
    let df = ctx.table("events").await?
        .filter(col("ts").gt(lit("2026-01-01")))?
        .aggregate(vec![col("user_id")], vec![count(lit(1)).alias("n")])?;
    df.write_parquet("out/", Default::default(), None).await?;
    Ok(())
}
```

**3. Python bindings**

```python
# pip install datafusion
from datafusion import SessionContext

ctx = SessionContext()
ctx.register_parquet("events", "data/events/")
ctx.sql("SELECT user_id, count(*) n FROM events GROUP BY user_id").show()
```

## Common pitfalls

- **It is a library, not a database**: no server, catalog persistence, transactions, or storage management out of the box. You assemble those (often with Iceberg/Delta/Hudi table providers or your own catalog).
- **Fast-moving API**: releases are frequent (roughly monthly major versions) with breaking changes. Pin versions and budget time for upgrades, especially if you implement custom `TableProvider`, `ExecutionPlan`, or optimizer rules.
- **Memory and spilling**: operators like sort, hash join, and aggregate can be memory-hungry. Configure a `RuntimeEnv` with a `MemoryPool` and a disk manager so queries spill instead of getting OOM-killed.
- **Parallelism tuning**: the default partition count equals the CPU count. Many tiny files or skewed partitions hurt throughput, so check `target_partitions` and file layout (right-sized Parquet row groups, sensible sort order).
- **Embedding inside an existing Tokio app**: CPU-heavy query work can starve latency-sensitive tasks. Run DataFusion on a dedicated runtime, as the project's own guidance recommends.
- **SQL dialect gaps**: coverage is broad but not identical to Postgres or Spark, and some functions or type coercions differ. Test your real queries, and use `EXPLAIN VERBOSE` to understand surprising plans.
- **Python vs. Rust expectations**: the Python package exposes a subset, so custom providers and deep extensions generally still mean writing Rust.
