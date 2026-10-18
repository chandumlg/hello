# Polars

Polars is a Rust-based DataFrame library (with Python, Node, and Rust APIs) that uses a columnar memory layout, multithreaded execution, and a lazy query optimizer to process tabular data far faster and with less memory than pandas.

## Why teams adopt it

Polars sits where pandas starts to hurt but a distributed engine like Spark is overkill. Staff and data/ML platform engineers use it to speed up feature pipelines, evaluation-set preparation, log analysis, and ETL jobs that run on a single (large) machine. Its expression API is composable and declarative, so transformations are easier to reason about, test, and reuse than chains of imperative pandas calls.

Common adoption triggers:
- A pandas job is slow, single-threaded, or runs out of memory on a few-GB-to-hundreds-of-GB dataset.
- You want a query optimizer (predicate/projection pushdown, common-subexpression elimination) without standing up a SQL engine.
- You process Parquet/CSV/NDJSON on disk or in S3 and want to scan only the columns and row groups you need.
- You need strict typing and a predictable null model in pipelines, interoperating with Apache Arrow.
- Datasets exceed RAM and you want the streaming engine to process them in batches.

## Basic usage

**1. Install and eager read/transform:**
```bash
pip install polars
```
```python
import polars as pl

df = pl.read_parquet("events.parquet")
out = (
    df.filter(pl.col("status") == "error")
      .group_by("service")
      .agg(
          pl.len().alias("errors"),
          pl.col("latency_ms").quantile(0.99).alias("p99_latency"),
      )
      .sort("errors", descending=True)
)
print(out)
```

**2. Lazy mode: scan, let the optimizer plan, then collect.** Only the needed columns and filtered row groups are read:
```python
lazy = (
    pl.scan_parquet("s3://my-bucket/events/*.parquet")
      .filter(pl.col("ts") >= pl.datetime(2026, 10, 1))
      .with_columns(pl.col("latency_ms").rolling_mean(window_size=50).over("service").alias("smoothed"))
      .select("service", "ts", "smoothed")
)
print(lazy.explain())          # inspect the optimized plan
result = lazy.collect()        # execute
# Larger-than-memory: lazy.sink_parquet("out.parquet") or collect(engine="streaming")
```

**3. Interop with SQL and pandas/Arrow:**
```python
ctx = pl.SQLContext(events=df)
ctx.execute("SELECT service, COUNT(*) AS n FROM events GROUP BY service", eager=True)

pdf = df.to_pandas()           # needs pyarrow
back = pl.from_pandas(pdf)
```

## Pitfalls and things to watch

- **Don't think in pandas.** There is no index, no in-place mutation, and row-wise `apply`/`map_elements` drops you into slow Python-callback land. Express logic with native expressions; reach for `map_elements` only as a last resort.
- **Eager vs. lazy.** `read_*` loads everything immediately. Prefer `scan_*` + `.collect()` for anything non-trivial, or you forfeit pushdown optimizations.
- **Memory spikes on collect.** Joins, sorts, and wide group-bys can still blow RAM. Use the streaming engine or `sink_*`, but note not every operation is streaming-capable and it may silently fall back to in-memory; check with `explain`.
- **Strict typing surprises.** Mixed-type columns, integer-vs-float inference from the first N rows (`infer_schema_length`), and null vs. NaN distinctions cause schema errors. Pass explicit `schema`/`schema_overrides` in production.
- **API churn.** Polars is pre-/early-1.x in spirit and renames have occurred (e.g. `groupby` → `group_by`, `streaming=True` → `engine="streaming"`). Pin versions and read changelogs on upgrade.
- **Not a distributed engine.** It scales up, not out. If data truly exceeds one machine, consider Spark, DataFusion-based systems, or Polars Cloud.
- **Ecosystem gaps.** Some libraries (scikit-learn, plotting) expect pandas/NumPy; convert at the boundary and avoid round-tripping in hot loops.
