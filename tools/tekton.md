# Tekton

## What it is
Tekton is a Kubernetes-native framework for building CI/CD systems, where pipelines, tasks, and runs are defined as custom resources and executed as pods in your cluster.

## Primary use cases and when to adopt it
- **Building an internal delivery platform.** Tekton is a toolkit, not a finished CI product. Platform teams use it as the execution engine behind a golden-path "build, scan, sign, deploy" experience for many product teams.
- **Supply-chain security.** Tekton Chains observes TaskRuns and produces signed SLSA provenance and attestations for the images and artifacts they create.
- **Cluster-resident builds.** Builds run next to the registries, secrets, and private networks they need, with no separate runner fleet to patch. Capacity comes from the cluster's autoscaler (e.g. Karpenter).
- **Portable, reusable steps.** Tasks are versioned and shared through the Tekton Hub or bundles in an OCI registry. They can be pulled in with *resolvers* (git, hub, bundles, cluster).

Adopt it when you are already Kubernetes-first, need to standardize CI/CD for many teams, and have platform engineers who can own the abstraction. Skip it if you want a turnkey UI and low operational overhead. Hosted CI such as GitHub Actions is usually cheaper to run for a small team.

Core objects:
- **Step**: one container.
- **Task**: an ordered list of steps that run in a single pod and share a filesystem.
- **Pipeline**: a DAG of tasks.
- **TaskRun / PipelineRun**: an execution of a Task or Pipeline.
- **Workspace**: shared storage (PVC, emptyDir, secret, configmap).
- **Triggers**: an EventListener, TriggerBinding and TriggerTemplate that turn webhooks into PipelineRuns.

## Getting started

### 1. Install the controllers and CLI
```bash
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl get pods -n tekton-pipelines --watch

# CLI (macOS)
brew install tektoncd-cli
tkn version
```

### 2. Run a Task
```yaml
# hello-task.yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: hello
spec:
  params:
    - name: who
      type: string
      default: world
  steps:
    - name: say
      image: alpine:3.20
      script: |
        #!/bin/sh
        echo "hello $(params.who)"
```
```bash
kubectl apply -f hello-task.yaml
tkn task start hello -p who=platform --showlog
```

### 3. Compose a Pipeline with a shared workspace
```yaml
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: build-and-test
spec:
  workspaces:
    - name: source
  tasks:
    - name: fetch
      taskRef:
        resolver: hub
        params:
          - {name: catalog, value: Tekton}
          - {name: type, value: artifact}
          - {name: kind, value: task}
          - {name: name, value: git-clone}
          - {name: version, value: "0.9"}
      workspaces:
        - {name: output, workspace: source}
      params:
        - {name: url, value: https://github.com/example/app}
    - name: test
      runAfter: [fetch]
      workspaces:
        - {name: src, workspace: source}
      taskSpec:
        workspaces: [{name: src}]
        steps:
          - name: unit
            image: golang:1.23
            workingDir: $(workspaces.src.path)
            script: go test ./...
```
```bash
tkn pipeline start build-and-test \
  -w name=source,volumeClaimTemplateFile=pvc.yaml --showlog
tkn pipelinerun list
```
The hub resolver's catalog, name, and version values above are illustrative. Check the current Tekton catalog, since hub task versions change.

## Common pitfalls
- **It is a toolkit, not a product.** Expect to build the UX yourself: templates, a dashboard (Tekton Dashboard is basic), RBAC, naming conventions. Budget platform-team time.
- **Workspace storage is the usual pain point.** PVC-backed workspaces tie a run to one zone and one access mode. ReadWriteOnce means parallel tasks on different nodes can't share one. The affinity assistant helps but constrains scheduling. Orphaned PVCs accumulate unless you use `volumeClaimTemplate`.
- **Startup overhead.** Each task is a pod, so scheduling and image pull add seconds per task. Group related work into steps within one task, and pre-pull or cache images.
- **Result and parameter size limits.** Task results are stored in the termination message (about 4 KB total per pod by default). Pass large data through workspaces instead.
- **Version drift between `v1beta1` and `v1`.** Older tutorials use the beta API. Field names and behaviors changed (e.g. `taskRef` resolvers replacing `bundle`), so pin your controller version and migrate manifests deliberately.
- **Unbounded history.** PipelineRuns and TaskRuns are objects in etcd. Set up pruning (the Tekton pruner, or Tekton Results to offload records and logs) before you have thousands of them.
- **Security defaults.** Steps run whatever image you give them, and Tekton Triggers endpoints accept webhooks. Validate payloads with interceptors, scope service accounts tightly, and avoid mounting broad secrets into untrusted PR builds.
