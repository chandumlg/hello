# SQLMesh

SQLMesh is a data transformation framework that versions your SQL models, computes exactly what a change affects, and promotes it to production by swapping pointers to already-built tables, so you stop rebuilding whole warehouses to ship a one-line change.

## Primary use cases

SQLMesh occupies the same space as dbt (SQL transformations on a warehouse or lakehouse: Snowflake, BigQuery, Databricks, Postgres, DuckDB, Trino, and others), but treats transformations more like deployable software artifacts:

- **Virtual data environments.** Each environment (`dev`, `prod`, a PR environment) is a set of views pointing at versioned physical tables. Creating a dev environment copies no data, and promoting to prod is a pointer swap.
- **Plan/apply workflow.** Like Terraform, `sqlmesh plan` shows what changes and what downstream models are affected, and classifies each change as breaking or non-breaking before anything runs.
- **Incremental models with state.** SQLMesh tracks which time intervals have been processed per model, so backfills, late-arriving data, and gaps are handled by the framework instead of hand-written `is_incremental()` logic.
- **Column-level lineage and semantic SQL understanding.** It parses SQL with SQLGlot, which enables column lineage, transpilation between dialects, and catching errors at plan time instead of at runtime.

Teams adopt it when warehouse costs or CI times are dominated by rebuilding models, when incremental logic has become a source of bugs, or when they want safe, reviewable promotion of data changes across environments. Staff and platform engineers like it because it gives data pipelines the plan/apply discipline they already use for infrastructure.

## Basic usage

**1. Install and scaffold a project** (DuckDB is the zero-setup default for trying it):

```bash
pip install sqlmesh
sqlmesh init duckdb
```

**2. Define a model.** The `MODEL` block holds metadata. This one is incremental by time, and SQLMesh injects the interval bounds:

```sql
-- models/daily_orders.sql
MODEL (
  name analytics.daily_orders,
  kind INCREMENTAL_BY_TIME_RANGE (time_column order_date),
  start '2026-01-01',
  cron '@daily',
  grain [order_date, customer_id]
);

SELECT
  order_date,
  customer_id,
  SUM(amount) AS revenue
FROM raw.orders
WHERE order_date BETWEEN @start_ds AND @end_ds
GROUP BY 1, 2
```

**3. Plan, review, and apply.** The first plan to a dev environment backfills it. Later plans show the diff and downstream impact:

```bash
sqlmesh plan dev        # build a virtual dev environment, shows diff + impact
sqlmesh plan            # promote to prod; reuses tables already built in dev
sqlmesh table_diff prod:dev analytics.daily_orders   # compare data across envs
sqlmesh test            # run unit tests defined in tests/*.yaml
```

## Common pitfalls

- **State database is critical.** SQLMesh stores snapshots and interval state in a state connection. The default local DuckDB or SQLite is fine for experiments but wrong for teams; use a shared Postgres or your warehouse, and back it up.
- **Breaking vs. non-breaking classification matters.** A change misclassified as non-breaking can skip backfilling downstream models. Review the categorization at plan time, and use forward-only plans deliberately for very large tables.
- **Incremental models need a correct `grain` and time column.** Wrong settings give silent duplicates or gaps, so write audits (`not_null`, `unique_values`) and unit tests.
- **Cron versus plan.** Models only run on their cron cadence via `sqlmesh run`, which is separate from `plan`. You need a scheduler (its built-in one, Airflow, Dagster, or CI cron) invoking `run` regularly.
- **Ecosystem gravity.** dbt has far more packages, integrations, and hiring familiarity. SQLMesh can read existing dbt projects, which is a lower-risk way to evaluate it, but expect some Jinja or macro edge cases.
- **Dev data is virtual, not sampled.** Dev environments reuse prod-scale tables when unchanged, but changed models are backfilled in full over their date range unless you limit it (for example `--start`/`--end` or a dev-specific model start window), which can be costly.
