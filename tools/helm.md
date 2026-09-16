# Helm

Helm is the package manager for Kubernetes: it solves the problem of shipping, versioning, and configuring multi-resource Kubernetes applications as a single templated, install/upgrade/rollback-able unit instead of a pile of hand-maintained YAML files.

## Primary use cases

- **Packaging applications for reuse.** Instead of copy-pasting Deployment/Service/Ingress/ConfigMap YAML between environments or teams, you define a "chart" once — a directory of templated manifests plus a `values.yaml` of defaults — and anyone can install it with their own overrides.
- **Environment-specific configuration.** The same chart deploys to dev, staging, and prod by swapping a `values-prod.yaml` file (replica counts, resource limits, image tags, feature flags) without touching the templates themselves.
- **Third-party and internal software distribution.** Most infrastructure software you run on Kubernetes — Prometheus, Grafana, cert-manager, ingress-nginx, Argo CD itself — ships an official Helm chart as its primary install method. Platform teams also publish internal charts (a standard "web service" chart, a "cron job" chart) so product teams don't reinvent Deployment boilerplate.
- **Release lifecycle management.** Helm tracks each install as a "release" with revision history, so `helm upgrade` and `helm rollback` give you atomic, versioned deploys instead of manually diffing and reapplying YAML.

A team typically adopts Helm the moment it has more than one Kubernetes application with near-identical shape (a web service, an API, a worker) that needs to be deployed repeatedly across environments or by multiple teams — the templating pays for itself as soon as copy-paste YAML starts drifting.

## Basic usage

**Install a chart from a public repo:**
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install my-prom prometheus-community/prometheus --namespace monitoring --create-namespace
```

**Scaffold and install your own chart:**
```bash
helm create my-service
# edit my-service/values.yaml and my-service/templates/*.yaml
helm install my-service ./my-service -f values-prod.yaml
```

**Upgrade, inspect, and roll back a release:**
```bash
helm upgrade my-service ./my-service --set image.tag=v1.4.2
helm history my-service
helm rollback my-service 1
```

## Common pitfalls

- **Templates aren't validated against the Kubernetes schema until apply time.** `helm template` renders YAML locally and `helm lint` catches basic mistakes, but a typo in a field name can still slip through and fail (or silently misconfigure) at `helm install`/`upgrade`. Always run `helm template` or `--dry-run` in CI before merging chart changes.
- **`values.yaml` sprawl.** Charts that expose every possible knob as a value end up with deeply nested, hard-to-reason-about config. Keep values scoped to what actually varies between environments; hardcode the rest in templates.
- **Tiller is gone, but old habits linger.** Helm 2's cluster-side Tiller component (a real security liability — cluster-wide RBAC) was removed in Helm 3. If you're following a tutorial or chart written for Helm 2, watch for stale assumptions about a server-side component.
- **Silent upgrade failures and stuck releases.** If a release hangs mid-upgrade (e.g., a pod never becomes ready) and you Ctrl-C or the pipeline times out, Helm can leave the release stuck in `pending-upgrade`. `helm rollback` or `helm upgrade --force` is often needed to unstick it — plan for this in automated deploy pipelines with reasonable `--timeout` values and alerting.
- **Chart dependency drift.** Subcharts pulled in via `dependencies` in `Chart.yaml` are pinned by version but not automatically updated; forgetting to run `helm dependency update` after bumping a version constraint deploys stale nested charts.
- **`--set` vs. values files.** `--set` is handy for one-off overrides but is easy to typo (it silently creates new keys instead of erroring on a misspelled path) and doesn't diff well in code review. Prefer values files checked into version control for anything that isn't a quick manual override.
