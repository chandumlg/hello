# Argo CD

Argo CD is a GitOps continuous delivery tool for Kubernetes that continuously
reconciles the live state of your cluster against the desired state declared
in a Git repository, so deployments become "merge a PR" instead of "run a
deploy script."

## Why teams adopt it

Once you're running more than a handful of services on Kubernetes, `kubectl
apply` and ad-hoc CI deploy scripts stop scaling: nobody can answer "what's
actually running in prod right now, and why?" without SSHing into a cluster
or digging through CI logs. Argo CD flips the model — Git becomes the single
source of truth, and a controller running inside (or alongside) the cluster
continuously watches for drift between the repo and the live cluster state,
then reconciles it.

Typical adoption triggers:

- **Multi-cluster / multi-environment sprawl.** Managing staging, prod, and
  regional clusters by hand becomes error-prone past a few clusters. Argo CD
  gives you one dashboard and one reconciliation loop across all of them.
- **Audit and compliance needs.** Every change to production is a Git commit
  with an author, a timestamp, and a diff — useful when you need to answer
  "who changed this and when" without digging through kubectl history.
- **Drift detection.** Someone `kubectl edit`s a Deployment directly in prod
  during an incident. Argo CD flags it as `OutOfSync` (or auto-heals it),
  instead of that change silently persisting until the next deploy overwrites
  it and nobody understands why.
- **Progressive delivery.** Paired with Argo Rollouts, teams get canary and
  blue-green deployments with automated analysis and rollback, driven by the
  same GitOps model.

It's most valuable for platform teams running Kubernetes at a scale where
"what's deployed where" is no longer a question one person can answer from
memory.

## Getting started

**1. Install Argo CD into a cluster:**

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

**2. Register an application** pointing at a Git repo/path — either via the
CLI or a declarative `Application` manifest checked into Git itself (the
"app of apps" pattern):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: payments-service
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/acme/k8s-manifests.git
    targetRevision: main
    path: apps/payments-service
  destination:
    server: https://kubernetes.default.svc
    namespace: payments
  syncPolicy:
    automated:
      prune: true      # remove resources deleted from Git
      selfHeal: true    # revert manual cluster edits back to Git state
```

```bash
kubectl apply -f payments-service-app.yaml
```

**3. Check sync status from the CLI:**

```bash
argocd login argocd.internal.acme.com
argocd app get payments-service
argocd app sync payments-service   # manual sync, if automated isn't enabled
```

From here, a deploy is just a commit to `apps/payments-service` (a new image
tag, a config change) — Argo CD picks it up and reconciles the cluster
within its poll/webhook interval.

## Common pitfalls

- **`selfHeal` surprises during incidents.** If self-healing is on, an
  on-call engineer's emergency `kubectl edit` or `kubectl scale` gets
  reverted automatically the next reconciliation pass. Know this behavior
  going in, or you'll "fix" an incident and watch it silently regress
  minutes later. Some teams pause auto-sync during active incidents.
- **Secrets don't belong in the Git repo.** Argo CD renders whatever's in
  Git, so raw Secret manifests checked into a repo are a real leak risk.
  Pair it with Sealed Secrets, External Secrets Operator, or Vault
  integration rather than committing plaintext.
- **Helm/Kustomize value sprawl.** As the number of applications grows,
  values files and overlays multiply fast. Without a clear convention (e.g.,
  a strict app-of-apps directory layout), it becomes hard to tell which
  values apply to which environment.
- **Sync waves and hooks are easy to get wrong.** Ordering resource creation
  (e.g., CRDs before the resources that use them, or DB migrations before
  app pods) requires `sync-wave` annotations and PreSync/PostSync hooks —
  skipping this causes intermittent apply-order failures that are hard to
  reproduce.
- **Large monorepo manifests slow reconciliation.** A single Application
  watching a huge directory of manifests makes diffs slow and noisy in the
  UI. Splitting into smaller, focused Applications (often via app-of-apps)
  keeps sync status legible.
- **Drift from mutating admission webhooks.** Webhooks/operators that
  inject sidecars or mutate specs (e.g., Istio, cert-manager) can make Argo
  CD perpetually show `OutOfSync` because the live manifest never matches
  Git exactly. Use `ignoreDifferences` in the Application spec for fields
  you know are legitimately mutated out-of-band.
