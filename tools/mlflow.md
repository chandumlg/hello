# MLflow

## What it is

MLflow is an open-source platform for managing the machine learning lifecycle — it solves the problem of ML experiments being untraceable and models being un-deployable: without it, teams lose track of which code, data, and hyperparameters produced which model, and "it worked on my laptop" becomes a permanent state for anything touching a model.

## Primary use cases

MLflow is built around four loosely-coupled components, and teams typically adopt it for one or two of them before growing into the rest:

- **Tracking** — log parameters, metrics, and artifacts (plots, model files, datasets) for every training run, then compare runs in a UI or via API. This is the component most teams start with.
- **Model Registry** — a central store for model versions with stage transitions (`Staging` → `Production` → `Archived`), approval workflows, and lineage back to the run that produced them.
- **Model packaging (MLmodel format)** — a standard way to save a model so it can be served identically regardless of which framework (scikit-learn, PyTorch, XGBoost, transformers, etc.) trained it.
- **Projects / Deployments** — reproducible run definitions and one-command deployment to serving targets (local REST server, Kubernetes, SageMaker, Databricks).

A team adopts MLflow when more than one person is training models and nobody can answer "which run is currently in production, and can we reproduce it?" It's also the default choice when you need a framework-agnostic model registry that isn't locked to a single cloud's ML platform — LLM fine-tuning pipelines, classical ML, and deep learning all log to the same tracking server.

## Basic usage

**1. Install and log a run**

```bash
pip install mlflow
```

```python
import mlflow

mlflow.set_experiment("churn-model")

with mlflow.start_run():
    mlflow.log_param("n_estimators", 200)
    mlflow.log_param("max_depth", 6)

    model = train_model(n_estimators=200, max_depth=6)
    accuracy = evaluate(model)

    mlflow.log_metric("accuracy", accuracy)
    mlflow.sklearn.log_model(model, "model")
```

**2. Launch the tracking UI and compare runs**

```bash
mlflow ui --port 5000
# open http://localhost:5000 — filter, sort, and diff runs side by side
```

Point multiple machines at a shared backend instead of the local default:

```bash
export MLFLOW_TRACKING_URI=http://mlflow-server:5000
```

**3. Register and serve a model**

```python
result = mlflow.register_model(
    "runs:/<run_id>/model", "churn-model"
)
```

```bash
mlflow models serve -m "models:/churn-model/Production" -p 1234
# now POST to http://localhost:1234/invocations for predictions
```

## Common pitfalls

- **Local file store by default.** Without `MLFLOW_TRACKING_URI` set, MLflow writes runs to `./mlruns` on whatever machine ran the script — fine for a laptop, useless for a team. Stand up a real tracking server (Postgres backend + S3/GCS artifact store) before more than one person is logging runs.
- **Artifact store vs. backend store are separate concerns.** Metrics/params go in the backend DB; model files and plots go in the artifact store (typically object storage). Misconfiguring only one leads to a UI that shows metrics but 404s on model downloads.
- **Unbounded experiment/run growth.** Teams that log every hyperparameter sweep iteration as a separate run end up with tens of thousands of runs and a slow UI — use nested runs or `mlflow.log_metric` with `step` for sweeps instead of one run per trial when possible, and set retention/cleanup policies.
- **Model Registry stage transitions are not access-controlled by default** in open-source MLflow — anything that can reach the tracking server can promote a model to `Production`. If that matters, put it behind auth (Databricks-managed MLflow or a reverse proxy with SSO) rather than assuming the stage tag is a real gate.
- **Autologging is convenient but opaque.** `mlflow.autolog()` captures a lot automatically, but it can silently log things you didn't intend (full datasets as artifacts, verbose framework internals) and bloat storage — check what it captures for your framework before relying on it in CI.
