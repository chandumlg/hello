# Feast

## What it is

Feast (Feature Store) is an open-source feature store that solves the training/serving skew problem in ML: the features used to train a model are typically computed in a batch pipeline (Spark, SQL, a notebook), while the features used to serve predictions in production need to come back in single-digit milliseconds from a live request — and if those two computations drift apart even slightly, the model silently degrades in ways that are brutal to debug.

## Primary use cases

- **Consistent offline/online features** — a feature is defined once (e.g. `user_7day_avg_spend`) and Feast materializes it to both an offline store (for generating training datasets via point-in-time joins) and an online store (for low-latency lookups at inference time), so the model sees the same feature logic in both places.
- **Point-in-time correctness** — when building a training dataset, Feast joins entity timestamps against feature timestamps so a row from three months ago only pulls feature values that were actually known at that point, preventing the classic "leaked the future into training" bug.
- **Feature reuse and discovery** — features become named, versioned, documented objects in a registry instead of being recomputed ad hoc by every model team, so a second team building a churn model can reuse the same `user_7day_avg_spend` feature another team already built and validated.
- **Decoupling ML services from data infra** — the serving layer calls Feast's SDK/API for features by name; it doesn't need to know whether the values live in Redis, DynamoDB, Snowflake, or BigQuery underneath, so the storage backend can change without touching model-serving code.

A team adopts Feast once it has more than one model consuming overlapping features, or once training/serving skew has already caused a production incident — it's rarely the first tool reached for on a single model with a handful of hand-computed features, since the registry and materialization pipeline are overhead a small project doesn't need yet.

## Basic usage

**1. Initialize a feature repo**

```bash
pip install feast
feast init my_feature_repo
cd my_feature_repo/feature_repo
# generates example feature definitions + feature_store.yaml (offline/online store config)
```

**2. Define an entity and a feature view**

```python
# example_repo.py
from feast import Entity, FeatureView, Field, FileSource
from feast.types import Float32
from datetime import timedelta

user = Entity(name="user_id", join_keys=["user_id"])

spend_source = FileSource(
    path="data/user_spend.parquet",
    timestamp_field="event_timestamp",
)

user_spend_view = FeatureView(
    name="user_spend",
    entities=[user],
    ttl=timedelta(days=7),
    schema=[Field(name="avg_spend_7d", dtype=Float32)],
    source=spend_source,
)
```

**3. Apply the definitions, materialize, and read features**

```bash
feast apply          # registers entities/feature views in the registry
feast materialize-incremental $(date +%Y-%m-%d)   # push latest values into the online store
```

```python
from feast import FeatureStore

store = FeatureStore(repo_path=".")

# training: point-in-time correct join against an entity dataframe
training_df = store.get_historical_features(
    entity_df=entity_df,  # columns: user_id, event_timestamp, label
    features=["user_spend:avg_spend_7d"],
).to_df()

# serving: low-latency online lookup
online_features = store.get_online_features(
    features=["user_spend:avg_spend_7d"],
    entity_rows=[{"user_id": 1001}],
).to_dict()
```

## Common pitfalls

- **The online store is only as fresh as your last materialization run.** `materialize-incremental` doesn't run itself — it needs to be scheduled (Airflow, cron, a Feast-managed job), and a missed or failed run means production silently serves stale features with no error raised.
- **Point-in-time joins are easy to defeat by accident.** If `entity_df` timestamps are wrong, missing, or in the wrong timezone, `get_historical_features` will still return a result — just one that leaks future data into training without any obvious signal that something went wrong.
- **Choosing an online store is a real infra decision, not a config toggle.** The default local (SQLite/file) setup is fine for a laptop but not for production; Redis, DynamoDB, or Datastore each bring their own latency, cost, and ops profile, and switching later means re-materializing everything.
- **The registry is a single point of coordination.** Two teams editing feature definitions and running `feast apply` against the same registry without a review process will step on each other — treat feature definitions like schema migrations (PR review, CI) rather than free-for-all edits.
- **Feast computes nothing itself.** It stores and serves feature *values*; the actual transformation logic still has to live somewhere (a Spark job, a SQL view, a streaming pipeline) that keeps the offline source up to date — Feast won't tell you if that upstream pipeline breaks.
