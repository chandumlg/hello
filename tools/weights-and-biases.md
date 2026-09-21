# Weights & Biases (W&B)

Weights & Biases is a machine learning experiment tracking and collaboration platform that solves the problem of ML training runs being unreproducible and incomparable — without it, teams end up with hyperparameters and metrics scattered across spreadsheets, terminal logs, and tribal memory, making it nearly impossible to answer "which run produced this checkpoint, and why did it work better?"

## Primary use cases

- **Experiment tracking**: log metrics, hyperparameters, gradients, and system stats (GPU utilization, memory) for every training run, then compare runs side by side in a dashboard.
- **Hyperparameter optimization**: W&B Sweeps run grid, random, or Bayesian search across a hyperparameter space, parallelizing trials across multiple machines/GPUs automatically.
- **Artifact and model versioning**: datasets, model checkpoints, and evaluation results are versioned as lineage-tracked artifacts, so you can trace exactly which data and code produced a given model.
- **LLM-specific workflows**: W&B Weave adds tracing and evaluation for LLM applications (prompts, chains, agent calls), similar in spirit to Langfuse.
- **Team collaboration**: shareable reports mix live charts, tables, and markdown so a team can write up findings without exporting screenshots.

A team typically adopts W&B once more than one person is training models and "which run was that?" starts costing real time, or once hyperparameter sweeps need to scale beyond a single machine.

## Basic usage

Install and log in:

```bash
pip install wandb
wandb login   # pastes an API key from wandb.ai/authorize
```

Instrument a training loop:

```python
import wandb

wandb.init(project="image-classifier", config={"lr": 0.001, "batch_size": 64})

for epoch in range(config.epochs):
    loss, acc = train_one_epoch()
    wandb.log({"loss": loss, "accuracy": acc, "epoch": epoch})

wandb.finish()
```

Run a hyperparameter sweep:

```bash
# sweep.yaml defines the search space and metric to optimize
wandb sweep sweep.yaml
wandb agent <sweep_id>   # run on as many machines/GPUs as you like
```

Log and version a model artifact:

```python
artifact = wandb.Artifact("model-checkpoint", type="model")
artifact.add_file("model.pt")
wandb.log_artifact(artifact)
```

## Common pitfalls

- **Logging too much, too often**: logging every batch instead of every epoch (or logging large tensors/images at high frequency) bloats run size and slows down the dashboard; log at a sensible cadence and downsample images.
- **Forgetting `wandb.finish()`** in notebooks or scripts that don't exit cleanly — orphaned runs show as "still running" and can confuse teammates checking status.
- **Config drift**: if hyperparameters are hardcoded rather than passed through `wandb.config`, sweeps silently don't take effect since the agent updates `config` but the code never reads from it.
- **Cost and data residency**: the hosted SaaS tier logs data to W&B's cloud, which is a problem for sensitive data — self-hosted (W&B Server) or a private cloud deployment is available but adds operational overhead.
- **Artifact storage growth**: versioning every checkpoint as an artifact adds up in storage costs quickly; prune old versions or only version checkpoints that pass an evaluation gate.
