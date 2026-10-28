# Apache Arrow

Apache Arrow is a language-independent, in-memory columnar data format (plus libraries and the Flight RPC protocol) that lets different systems and languages share tabular data without serializing, copying, or re-parsing it.

## Why teams adopt it

Every data system used to have its own in-memory representation, so moving data between pandas, Spark, a database driver, and an ML framework meant paying a serialization tax at each boundary. Arrow defines one standard memory layout for columnar data, so a buffer produced by Rust can be read by Python, C++, or Java with zero copies, and vectorized (SIMD-friendly) compute works directly on it.

Staff, data, and AI engineers meet Arrow most often as plumbing underneath other tools: Polars, DuckDB, DataFusion, pandas 2.x (optional backend), Parquet readers, Hugging Face Datasets, and many warehouse drivers all speak it. Teams adopt it directly when:
- They are building a service or pipeline that hands tabular data between languages or processes and JSON/CSV/pickle has become the bottleneck.
- They need high-throughput data transport: Arrow Flight (gRPC-based) and Flight SQL replace row-oriented JDBC/ODBC for large result sets.
- They want memory-mapped, larger-than-RAM access to datasets (Arrow IPC/Feather files, Parquet via `pyarrow.dataset`).
- They are writing a query engine, feature store, or ML data loader and want a standard columnar substrate instead of inventing one.

## Basic usage

**1. Build a table and write/read Parquet with PyArrow:**
```python
import pyarrow as pa
import pyarrow.parquet as pq

table = pa.table({"user_id": [1, 2, 3], "country": ["US", "DE", "IN"], "spend": [10.5, 3.2, 8.0]})
pq.write_table(table, "users.parquet", compression="zstd")
print(pq.read_table("users.parquet", columns=["country", "spend"]))
```

**2. Zero-copy interop with other engines (DuckDB, pandas, Polars):**
```python
import duckdb, polars as pl

top = duckdb.sql("SELECT country, sum(spend) AS total FROM table GROUP BY 1").arrow()  # Arrow in, Arrow out
df_pl = pl.from_arrow(top)         # typically zero-copy
df_pd = top.to_pandas(types_mapper=__import__("pandas").ArrowDtype)  # keep Arrow-backed dtypes
```

**3. Query a larger-than-memory partitioned dataset lazily, with predicate/column pushdown:**
```python
import pyarrow.dataset as ds
import pyarrow.compute as pc

dataset = ds.dataset("s3://bucket/events/", format="parquet", partitioning="hive")
result = dataset.to_table(
    columns=["user_id", "ts"],
    filter=(pc.field("date") >= "2026-10-01") & (pc.field("event") == "purchase"),
)
```

Serving data over Arrow Flight is a few lines more (subclass `pyarrow.flight.FlightServerBase`, implement `do_get`, and return a `RecordBatchStream`), and clients then pull record batches over gRPC in parallel.

## Pitfalls and things to watch

- **"Zero-copy" has conditions.** Conversion to pandas/NumPy is only zero-copy for primitive types without nulls; strings, nested types, and nullable ints trigger copies. Check with `to_pandas(zero_copy_only=True)` when it matters.
- **Immutability.** Arrow arrays are immutable. Row-by-row appends or in-place updates are an anti-pattern; build batches with builders or construct columns in bulk.
- **Memory pools and ownership.** Arrow memory lives outside the Python heap; large tables can look fine in Python profiling while RSS climbs. Release references promptly, and be careful when passing buffers across the C Data Interface, where lifetime/release callbacks matter.
- **Batch sizing.** Tiny record batches destroy throughput (per-batch overhead, poor vectorization); huge ones hurt latency and memory. Aim for roughly 64K–1M rows per batch.
- **Type fidelity across systems.** Timestamps (units and time zones), decimals, dictionary-encoded columns, and `large_string` vs `string` are common sources of subtle mismatches between engines and between Parquet writer versions.
- **Arrow is not a database or a storage format.** It is a memory/transport layer. Use Parquet (or Iceberg/Delta on top) for durable storage; Arrow IPC files are best for ephemeral or memory-mapped use.
- **Version drift.** Mixed pyarrow versions across a pipeline's components (especially with C Data Interface or Flight) can cause hard-to-diagnose crashes; pin versions consistently.
