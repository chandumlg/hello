# Apache Spark

Apache Spark is a distributed compute engine that solves one core problem: transforming and analyzing datasets too large (or too slow) to process on a single machine, by splitting the work across a cluster of machines while giving you a single, mostly-unified API for batch, streaming, SQL, and ML workloads.

## Primary use cases

- **Large-scale ETL** — reading raw data from object storage (S3, GCS, ADLS) or warehouses, transforming it, and writing it back out as Parquet/Iceberg/Delta tables. This is the bread-and-butter use case for most platform and data engineering teams.
- **Batch analytics on data too big for a single node** — joins, aggregations, and window functions over billions of rows where a single-process tool (pandas, DuckDB) would run out of memory or take hours.
- **Structured Streaming** — near-real-time pipelines (micro-batch, sub-second to a few seconds of latency) reading from Kafka/Kinesis and writing to sinks, using the same DataFrame API as batch jobs.
- **Distributed ML feature engineering and training** — MLlib for classical ML at scale, or Spark as the data-prep layer feeding into training frameworks like Ray or PyTorch.
- **Ad hoc / interactive analysis** — via `pyspark` shell, Jupyter, or Databricks/EMR notebooks, when a data scientist needs to explore a dataset that doesn't fit in memory.

A team adopts Spark when a single-node engine (pandas, DuckDB, Polars) genuinely can't handle the data volume or the job needs to scale elastically across a cluster — not by default. It's a bigger operational commitment (cluster management, JVM tuning, shuffle behavior) than those alternatives, so the usual trigger is "our nightly job doesn't fit in memory anymore" or "we need sub-hour freshness on a multi-TB streaming pipeline."

## Basic usage examples

**1. Local PySpark DataFrame job (no cluster needed for development):**

```python
from pyspark.sql import SparkSession
from pyspark.sql.functions import col, sum as spark_sum

spark = SparkSession.builder.appName("orders-summary").getOrCreate()

orders = spark.read.parquet("s3a://my-bucket/orders/")

summary = (
    orders
    .filter(col("status") == "completed")
    .groupBy("customer_id")
    .agg(spark_sum("amount").alias("total_spent"))
    .orderBy(col("total_spent").desc())
)

summary.write.mode("overwrite").parquet("s3a://my-bucket/orders-summary/")
```

**2. Submitting a job to a cluster:**

```bash
spark-submit \
  --master yarn \
  --deploy-mode cluster \
  --num-executors 20 \
  --executor-memory 8g \
  --executor-cores 4 \
  jobs/orders_summary.py
```

**3. Spark SQL directly (useful for analysts or quick checks):**

```python
orders.createOrReplaceTempView("orders")
spark.sql("""
    SELECT customer_id, SUM(amount) AS total_spent
    FROM orders
    WHERE status = 'completed'
    GROUP BY customer_id
    ORDER BY total_spent DESC
    LIMIT 100
""").show()
```

## Common pitfalls

- **Shuffle explosions.** Operations like `groupBy`, `join`, and `distinct` trigger a shuffle (data movement across the cluster). Skewed keys (one customer_id with 40% of all rows) cause a handful of tasks to take far longer than the rest — watch the Spark UI's "task duration" spread, not just job duration.
- **Too many small files.** Writing output with high parallelism and low data volume per partition produces thousands of tiny files, which kills downstream read performance. Use `.coalesce(n)` or `.repartition(n)` before writing, or adopt a table format (Iceberg/Delta/Hudi) that compacts automatically.
- **Lazy evaluation surprises.** Transformations (`filter`, `select`, `groupBy`) don't execute until an action (`write`, `collect`, `show`) is called. Debugging "why is this slow" means looking at the physical plan (`.explain()`) for the whole chain, not any single line.
- **Driver OOM from `.collect()`.** Pulling a large distributed DataFrame back into the driver's single JVM with `.collect()` or `.toPandas()` is a common way to crash a job that was otherwise scaling fine.
- **Under- or over-provisioned executors.** Too few executors leaves the cluster underutilized; too many small executors wastes memory on JVM overhead per executor. There's no universal formula — it depends on data volume, join strategy, and shuffle partition count (`spark.sql.shuffle.partitions`, default 200, often wrong for both very small and very large jobs).
- **Silent schema drift.** Reading Parquet/JSON without an explicit schema lets Spark infer one from a sample of files, which can silently change (or fail) when upstream data evolves. Pin schemas explicitly for production pipelines.
