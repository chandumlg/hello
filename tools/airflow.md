# Apache Airflow

Apache Airflow is a platform for programmatically authoring, scheduling, and monitoring workflows, and it solves the problem of running complex, interdependent batch jobs (think: "extract data, wait for it to land, transform it, then notify five downstream systems") reliably and observably instead of gluing them together with cron and shell scripts.

## Primary use cases

Airflow's core abstraction is the **DAG** (directed acyclic graph): a Python-defined pipeline made of tasks with explicit dependencies. Each DAG run gets scheduled, retried on failure, and tracked with full history in a web UI. Teams reach for it when:

- **ETL/ELT orchestration** — nightly or hourly jobs that pull data from APIs/databases, load it into a warehouse, and trigger transformations (often paired with dbt: Airflow schedules `dbt run`, dbt does the SQL).
- **ML pipeline scheduling** — retraining models on a cadence, orchestrating feature computation, or chaining a training job to a deployment step.
- **Cross-system coordination** — a pipeline that touches S3, Snowflake, Kubernetes, and a Slack webhook, where you need a single place to see what ran, what failed, and why.
- **Backfills and reprocessing** — Airflow's scheduler natively understands "run this DAG for every day between 2024-01-01 and today," which ad hoc scripts handle poorly.

A team adopts Airflow once cron jobs stop being manageable — when failures need automatic retries and alerting, when jobs depend on other jobs finishing successfully, or when non-engineers need visibility into whether last night's pipeline actually ran.

## Basic usage

**1. Install and start a local instance (Astro CLI or plain pip):**
```bash
pip install apache-airflow
airflow standalone   # initializes the DB, creates an admin user, starts webserver + scheduler
# UI now available at http://localhost:8080
```

**2. Define a DAG** (Python file dropped into your `dags/` folder):
```python
from airflow.sdk import dag, task
from datetime import datetime

@dag(schedule="@daily", start_date=datetime(2026, 1, 1), catchup=False)
def etl_pipeline():

    @task
    def extract():
        return {"rows": 1000}

    @task
    def load(data: dict):
        print(f"loading {data['rows']} rows")

    load(extract())

etl_pipeline()
```
This uses the TaskFlow API (modern Airflow) — Python functions become tasks, and calling one with another's return value wires up the dependency automatically.

**3. Trigger and inspect runs from the CLI:**
```bash
airflow dags trigger etl_pipeline
airflow tasks list etl_pipeline
airflow dags backfill etl_pipeline --start-date 2026-01-01 --end-date 2026-01-07
```

## Common pitfalls

- **The scheduler re-parses every DAG file repeatedly.** Heavy top-level code (API calls, large imports, DB queries) outside of task functions runs on every parse cycle and can slow the whole scheduler down. Keep DAG-file top level lightweight; do real work inside tasks.
- **`start_date` and `catchup` surprises.** Setting `catchup=True` (or leaving Airflow's old default) on a DAG with a `start_date` in the past will trigger a burst of backfill runs the moment you deploy it — usually not what you want for a brand-new pipeline.
- **Tasks should be idempotent.** Airflow retries failed tasks and reruns backfills, so a task that appends rows or sends a notification without checking for prior execution will duplicate work on retry.
- **XCom is for small data, not payloads.** Passing DataFrames or large blobs between tasks via XCom (Airflow's task-communication mechanism) bloats the metadata database; write large intermediate data to object storage and pass a reference instead.
- **Local executor doesn't scale.** The default `SequentialExecutor`/`LocalExecutor` is fine for development, but production workloads need `CeleryExecutor` or `KubernetesExecutor` with a real message broker — plan the executor choice before you have hundreds of concurrent tasks.
- **Version upgrades can break DAGs.** Airflow 2.x's TaskFlow API and Airflow 3.x's further API changes deprecate older patterns (classic `PythonOperator` boilerplate, `airflow.models` imports); pin your Airflow version and test upgrades against a DAG staging environment rather than upgrading in place on production.
