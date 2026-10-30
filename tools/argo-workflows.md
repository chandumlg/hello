# Argo Workflows

Argo Workflows is a Kubernetes-native workflow engine that runs multi-step jobs (DAGs or sequences of containers) as custom resources, so batch pipelines, CI, and ML jobs get scheduling, retries, and parallelism from the cluster itself.

## Why teams adopt it

Running multi-step batch work on Kubernetes usually starts as a pile of `Job`s and shell scripts, then grows into needing dependencies, fan-out/fan-in, retries, artifact passing, and a UI to see what failed. Argo Workflows (a CNCF graduated project) models each step as a pod and the whole pipeline as a `Workflow` CRD, with a controller that drives it to completion.

Typical use cases:
- **Data and ML pipelines**: fan out over shards or hyperparameters, then aggregate.
- **CI/CD and build pipelines** on Kubernetes (often alongside Argo CD and Argo Events).
- **Infrastructure and maintenance jobs**: migrations, backfills, cluster-wide batch tasks.
- **Scheduled work** via `CronWorkflow`, reusable steps via `WorkflowTemplate`.

Platform teams adopt it when they already live on Kubernetes, want pipelines defined declaratively in Git, and don't want to run a separate scheduler (Airflow) or a code-first durable engine (Temporal). Trade-off: workflows are YAML-first and each step is a pod, so it suits coarse-grained batch steps, not millisecond-level tasks or long-lived business logic.

## Core concepts

- **Template**: a unit of work (`container`, `script`, `resource`, `dag`, `steps`).
- **DAG vs steps**: `dag` declares dependencies per task; `steps` is sequential groups with parallelism inside a group.
- **Artifacts and parameters**: parameters are small strings; artifacts are files stored in S3/GCS/MinIO.
- **Controller / Argo Server**: the controller reconciles Workflows; the server provides UI, API, and SSO.

## Getting started

Install into a cluster (pin a version in real use):

```bash
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v3.6.0/install.yaml
kubectl -n argo port-forward deploy/argo-server 2746:2746   # UI at https://localhost:2746
```

**1. A minimal DAG with fan-out.** Save as `fanout.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: fanout-
spec:
  entrypoint: main
  serviceAccountName: workflow-runner
  templates:
    - name: main
      dag:
        tasks:
          - name: plan
            template: plan
          - name: process
            template: process
            dependencies: [plan]
            arguments:
              parameters: [{name: shard, value: "{{item}}"}]
            withParam: "{{tasks.plan.outputs.result}}"
          - name: report
            template: report
            dependencies: [process]
    - name: plan
      script:
        image: python:3.12-alpine
        command: [python]
        source: |
          import json; print(json.dumps([1, 2, 3]))
    - name: process
      inputs: {parameters: [{name: shard}]}
      container:
        image: alpine:3.20
        command: [sh, -c]
        args: ["echo processing shard {{inputs.parameters.shard}}"]
    - name: report
      container: {image: alpine:3.20, command: [echo], args: ["done"]}
```

```bash
argo submit -n argo --watch fanout.yaml
argo logs -n argo @latest
```

**2. Retries, timeouts, and resource limits on a step.**

```yaml
- name: flaky-step
  retryStrategy:
    limit: "3"
    retryPolicy: OnError
    backoff: {duration: 10s, factor: 2}
  activeDeadlineSeconds: 600
  container:
    image: myorg/job:1.4.2
    resources:
      requests: {cpu: "1", memory: 2Gi}
```

**3. Reusable templates and scheduling.**

```bash
argo template create -n argo etl-template.yaml                 # WorkflowTemplate
argo submit -n argo --from workflowtemplate/etl --parameter date=2026-10-30
argo cron create -n argo nightly-etl.yaml                       # CronWorkflow
```

## Pitfalls

- **etcd size and CRD bloat**: each Workflow stores its full node status in the CR (1 MB etcd object limit). Large fan-outs (thousands of nodes) can fail. Enable node-status offloading to Postgres/MySQL and use `ttlStrategy` / `podGC` to clean up.
- **Pod-per-step overhead**: scheduling and image pull latency make tiny steps slow and expensive. Batch work into fewer, bigger steps, and pre-pull or slim images.
- **RBAC for the executor**: the workflow service account needs permissions (e.g. `workflowtaskresults`) or steps hang/fail at the end; default SA is often too restrictive. Create a dedicated one.
- **Artifact repository not configured**: artifacts fail until a default repository (S3/GCS) is set in `workflow-controller-configmap`. Don't pass large data through parameters.
- **Controller is a single point of throughput**: tune `--workflow-workers`, `--qps`/`--burst`, and parallelism limits at high volume; set `parallelism` and semaphores/mutexes to avoid overwhelming downstream systems.
- **YAML sprawl**: complex logic in expressions gets unreadable. Consider the Hera (Python) SDK, or Couler, to author workflows in code.
- **Argo Server auth**: don't run it with `--auth-mode=server` exposed publicly; use SSO or client auth.
- **Not durable execution**: a failed controller restart resumes fine, but there's no event-sourced replay semantics like Temporal; design steps to be idempotent.
