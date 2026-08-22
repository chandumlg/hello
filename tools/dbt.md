# dbt (data build tool)

dbt lets data and platform teams turn raw tables sitting in a warehouse into tested, documented, version-controlled datasets by writing plain SQL `SELECT` statements instead of hand-rolled ETL scripts or fragile scheduled stored procedures.

## Primary use cases

dbt owns the "T" in ELT: once raw data has landed in a warehouse (Snowflake, BigQuery, Databricks, Redshift, DuckDB, ClickHouse, and others via adapters), dbt compiles and runs the SQL that transforms it into clean, analysis-ready models. Typical adopters are:

- **Analytics engineering teams** who want their transformation logic under version control, code review, and CI instead of buried in scheduled queries or BI-tool calculated fields.
- **Platform/data platform teams** standardizing how every team in the org defines core business entities (e.g., "an active customer") so five teams stop building five slightly different answers.
- **Staff/AI engineers building feature pipelines or ML training sets** who need reproducible, testable SQL transformations feeding into a warehouse-backed feature store or reporting layer.

A team typically adopts dbt once transformation logic has outgrown ad hoc SQL scripts — when nobody can confidently say which query is the source of truth, when a schema change breaks downstream dashboards without warning, or when onboarding a new analyst means explaining an undocumented tangle of views.

## Basic usage

Install and initialize a project:

```bash
pip install dbt-core dbt-snowflake   # swap adapter for your warehouse
dbt init my_project
cd my_project
```

Define a model as a `.sql` file containing just a `SELECT` — dbt handles the `CREATE TABLE`/`CREATE VIEW` wrapping:

```sql
-- models/staging/stg_orders.sql
select
    id as order_id,
    customer_id,
    status,
    cast(created_at as timestamp) as created_at
from {{ source('raw', 'orders') }}
where status is not null
```

Chain models together by referencing them with `ref()` instead of hardcoding table names — this is how dbt builds its dependency graph and run order:

```sql
-- models/marts/fct_customer_orders.sql
select
    customer_id,
    count(*) as order_count,
    sum(amount) as lifetime_value
from {{ ref('stg_orders') }}
group by 1
```

Run and test the project:

```bash
dbt run          # builds all models in dependency order
dbt test         # runs schema/data tests defined in .yml files
dbt docs generate && dbt docs serve   # browsable lineage graph + docs
```

## Common pitfalls

- **`dbt run` doesn't mean "correct."** It only means the SQL compiled and executed. Skipping `dbt test` (uniqueness, not-null, referential integrity checks defined in YAML) is the most common way teams end up with silently broken models.
- **Full refreshes get expensive fast.** Incremental models (`materialized='incremental'`) avoid rebuilding entire fact tables on every run, but a badly written incremental filter can silently under- or double-count data — verify incremental logic against a full-refresh baseline before trusting it in production.
- **The DAG can rot without ownership.** As a project grows past a few hundred models, undocumented or unowned models turn into the same "nobody knows what this feeds" problem dbt was meant to solve — enforce naming conventions (`stg_`, `int_`, `fct_`/`dim_`) and require descriptions/owners in `.yml` files from day one.
- **Jinja macros are powerful but easy to overuse.** Heavy macro abstraction makes models harder to read and debug for anyone who isn't already deep in the codebase; prefer plain SQL until duplication genuinely earns a macro.
- **dbt transforms, it doesn't extract or load.** It assumes data is already in the warehouse — pairing it with an EL tool (Fivetran, Airbyte, custom pipelines) is still necessary, and confusing dbt for a full ETL replacement leads to gaps in the pipeline.
