# Dagster

Dagster is a data orchestrator that solves the problem of building, scheduling, and monitoring data pipelines as declarative, testable **software-defined assets** rather than as loose, imperative task graphs.

## What problem it solves

Tools like cron or hand-rolled scripts get pipelines running, but they don't tell you what data exists, where it came from, or whether it's fresh and correct. Airflow-style orchestrators improved on cron by giving you DAGs of tasks, but a task graph still describes *execution order*, not the *data* being produced — you can't easily ask "what depends on this table?" or "is this dataset up to date?" without inferring it from task names. Dagster's core idea is to make the data itself — a table, a file, an ML model, a report — the first-class unit you declare, with dependencies expressed as data lineage between assets. That gives you a queryable catalog of your pipelines' outputs, built-in freshness and data-quality checks, and local testability without needing a live orchestrator running.

## Primary use cases and when to adopt it

- **Data platform teams** building and maintaining ELT/ETL pipelines (e.g., raw ingestion → dbt transformations → BI-ready tables) who want lineage and observability across the whole chain, not just per-tool.
- **ML platform / AI engineering teams** who need to track derived assets like feature tables, training datasets, and model artifacts, and want to know what's downstream of a given upstream change.
- Teams already using **dbt**, since Dagster has first-class dbt integration that turns each dbt model into a Dagster asset automatically, unifying dbt lineage with upstream Python-based ingestion.
- Teams that outgrew ad hoc scripts or cron and need retries, backfills, partitioning (e.g., per-day or per-region datasets), and a UI to see pipeline health — but find Airflow's task-centric model awkward for reasoning about data freshness and lineage.
- Adopt it when your pain point is specifically "I don't know what depends on what" or "I can't easily test a pipeline step locally" — if you just need simple time-based job scheduling with no asset lineage need, a lighter tool may suffice.

## Basic usage examples

**1. Install and scaffold a project**

```bash
pip install dagster dagster-webserver
dagster project scaffold --name my-project
cd my-project
dagster dev   # launches the Dagster UI at localhost:3000
```

**2. Define a couple of dependent assets**

```python
# my_project/assets.py
import pandas as pd
from dagster import asset

@asset
def raw_orders() -> pd.DataFrame:
    return pd.read_csv("https://example.com/orders.csv")

@asset
def orders_by_customer(raw_orders: pd.DataFrame) -> pd.DataFrame:
    return raw_orders.groupby("customer_id").size().reset_index(name="order_count")
```

Dagster infers the dependency `raw_orders -> orders_by_customer` from the function argument name, and both show up in the UI's asset graph with lineage automatically drawn.

**3. Materialize assets and add a schedule**

```bash
dagster asset materialize --select "*" -m my_project
```

```python
from dagster import define_asset_job, ScheduleDefinition

daily_job = define_asset_job("daily_orders_job", selection="*")
daily_schedule = ScheduleDefinition(job=daily_job, cron_schedule="0 6 * * *")
```

## Common pitfalls

- **Over-modeling assets**: not every intermediate value needs to be an `@asset`. Splitting logic too finely creates dozens of tiny assets that clutter the lineage graph and add materialization overhead — use plain functions/ops for internal computation and reserve `@asset` for things you actually want tracked as persistent data.
- **IO manager confusion**: assets return Python objects, and a configured `IOManager` decides how they're persisted (local pickle by default, which is fine for dev but not for production). Teams forget to set up a proper IO manager (e.g., for S3/Snowflake) before going to production and are surprised where data actually lands.
- **Partitions add real complexity**: partitioned assets (daily/hourly partitions) are powerful for backfills but require care with partition mappings between upstream/downstream assets of different granularity — mismatched partition definitions are a common source of confusing backfill failures.
- **Resource/config sprawl in large repos**: as asset counts grow into the hundreds, teams that don't invest early in shared `Definitions` organization, resource groups, and asset groups/tags end up with an unnavigable single global asset graph.
- **dbt integration version drift**: the `dagster-dbt` integration ties closely to your dbt project's manifest; upgrading dbt or restructuring dbt projects without regenerating the Dagster manifest mapping can silently break asset dependency inference.
