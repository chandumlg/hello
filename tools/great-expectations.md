# Great Expectations (GX)

Great Expectations is an open-source Python framework for data quality testing: you declare assertions about your data ("Expectations" such as *this column is never null*, *values are unique*, *row count is between X and Y*) and run them against DataFrames, warehouses, or files so bad data is caught before it reaches dashboards, models, or downstream services.

## Primary use cases and when teams adopt it

Unit tests protect code; nothing protects the data flowing through it. GX fills that gap, and it shows up in data platform, analytics engineering, and ML teams that have been burned by silent data breakage.

- **Pipeline gates**: validate a batch after ingestion or a transform step (in Airflow, Dagster, Prefect, etc.) and stop the pipeline, or quarantine the batch, when it fails.
- **Source/contract monitoring**: catch upstream schema drift, such as a renamed column, a type change, or a new enum value in an event payload, at the boundary instead of three joins later.
- **ML feature and training-data checks**: assert distributions, ranges, and null rates on features before training or batch inference so a broken feed doesn't quietly degrade a model.
- **Data Docs**: generated HTML reports of what was checked and what failed, useful for audits and for showing data consumers what is guaranteed.

**When to adopt:** you have several pipelines, more than one team consumes the data, and incidents keep tracing back to "the data was wrong and nobody noticed." If you already run dbt, its built-in tests (plus `dbt-expectations`) may be enough for warehouse-only checks. GX earns its place when you need checks outside the warehouse (Pandas/Spark DataFrames, files, multiple databases), richer statistical expectations, or one validation layer shared across many tools.

**Core concepts (GX 1.x):**
- **Data Source → Data Asset → Batch Definition**: where the data lives, which table or DataFrame, and how to slice it (whole table, daily partition, and so on).
- **Expectation / Expectation Suite**: individual assertions, grouped together.
- **Validation Definition**: binds a suite to a batch definition.
- **Checkpoint**: runs one or more validation definitions and triggers **Actions** (update Data Docs, Slack alert, custom code).

## Getting started

```bash
uv add great_expectations        # or: pip install great_expectations
# add extras for SQL backends, e.g. great_expectations[postgresql] / [snowflake]
```

### 1. Ad-hoc validation of a Pandas DataFrame

The quickest way to try it out, in a notebook or a script:

```python
import great_expectations as gx
import pandas as pd

df = pd.read_parquet("orders.parquet")

context = gx.get_context()   # ephemeral, in-memory context
batch = (
    context.data_sources.add_pandas("pandas")
    .add_dataframe_asset(name="orders")
    .add_batch_definition_whole_dataframe("all_rows")
    .get_batch(batch_parameters={"dataframe": df})
)

result = batch.validate(
    gx.expectations.ExpectColumnValuesToNotBeNull(column="order_id")
)
print(result.success, result.result)   # includes unexpected counts / sample values
```

### 2. A reusable suite plus a checkpoint (pipeline gate)

```python
import great_expectations as gx

context = gx.get_context(mode="file")   # persists config under ./gx/

ds = context.data_sources.add_postgres(
    "warehouse", connection_string="${PG_CONN}"   # substituted from env / config_variables.yml
)
asset = ds.add_table_asset("orders", table_name="orders")
batch_def = asset.add_batch_definition_whole_table("full_table")

suite = context.suites.add(gx.ExpectationSuite(name="orders_suite"))
suite.add_expectation(gx.expectations.ExpectColumnValuesToBeUnique(column="order_id"))
suite.add_expectation(gx.expectations.ExpectColumnValuesToBeBetween(
    column="amount", min_value=0, max_value=100_000))
suite.add_expectation(gx.expectations.ExpectColumnValuesToBeInSet(
    column="status", value_set=["pending", "paid", "refunded", "cancelled"]))
suite.add_expectation(gx.expectations.ExpectTableRowCountToBeBetween(
    min_value=1_000, max_value=None))

vd = context.validation_definitions.add(
    gx.ValidationDefinition(name="orders_vd", data=batch_def, suite=suite))

checkpoint = context.checkpoints.add(gx.Checkpoint(
    name="orders_checkpoint",
    validation_definitions=[vd],
    actions=[gx.checkpoint.UpdateDataDocsAction(name="docs")],
    result_format={"result_format": "SUMMARY"},
))
```

Then, in the orchestrator task:

```python
import sys
import great_expectations as gx

result = gx.get_context(mode="file").checkpoints.get("orders_checkpoint").run()
if not result.success:
    sys.exit(1)   # fail the Airflow/Dagster task so downstream stops
```

### 3. Tolerances and severity, so failures carry useful signal

```python
# Allow up to 1% nulls in an optional field instead of a hard 0% rule
gx.expectations.ExpectColumnValuesToNotBeNull(column="coupon_code", mostly=0.99)

# Parameterize thresholds at run time rather than hard-coding them
gx.expectations.ExpectTableRowCountToBeBetween(
    min_value={"$PARAMETER": "min_rows"}, max_value=None)
# checkpoint.run(expectation_parameters={"min_rows": 5000})
```

Commit the `gx/` directory (suites, checkpoints, and config, but not secrets or `uncommitted/`) so expectations are code-reviewed like any other test.

## Common pitfalls

- **Version churn.** GX 1.0 (2024) was a breaking rewrite. Blog posts, Stack Overflow answers, and LLM-generated snippets often show 0.x APIs (`get_validator`, `RuntimeBatchRequest`, `great_expectations init`, YAML-heavy configs). Pin your version and check examples against the 1.x docs.
- **Alert fatigue from over-strict rules.** Hundreds of auto-profiled expectations with hard thresholds will fire constantly and get ignored. Start with a few high-value checks (keys unique and not null, schema, row-count bounds, enum sets), use `mostly=` for tolerances, and only add a rule when you know who acts on its failure.
- **Validation cost on big tables.** Expectations compile to SQL or Spark scans. Running a whole-table suite on a multi-TB fact table every hour costs real warehouse money. Validate the new partition (daily or partitioned batch definitions) rather than full history, and keep heavy statistical checks on a slower schedule.
- **Failing without stopping anything.** A checkpoint that only updates Data Docs catches nothing in practice. Decide per suite whether failure should halt the pipeline, quarantine the batch, or just alert, and wire that up explicitly (exit code, orchestrator sensor, Slack/PagerDuty action).
- **Secrets in config.** In file mode, connection strings end up in `great_expectations.yml`. Use `${ENV_VAR}` substitution or `uncommitted/config_variables.yml`, and keep `uncommitted/` out of git (Data Docs can also contain sample rows, which may include PII).
- **Treating it as observability.** GX checks rules you wrote. It won't detect anomalies you didn't anticipate. Teams often pair it with freshness/volume anomaly monitoring, or with GX Cloud's features, rather than expecting it to cover everything.
- **Overlap with dbt tests.** Running both without a clear split leads to duplicated, drifting rules. A common split: dbt tests for in-warehouse model invariants, and GX at ingestion boundaries, for non-SQL data, and for ML feature checks.
