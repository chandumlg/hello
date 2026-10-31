# Argo Rollouts

## What it is and the problem it solves

**Argo Rollouts** is a Kubernetes controller and CRD set that replaces the built-in `Deployment` rolling update with progressive delivery strategies (canary and blue-green), including automated analysis against metrics and automatic rollback when a release looks unhealthy.

## Primary use cases and when to adopt it

- **Canary releases with traffic shaping**: shift 5% → 25% → 50% → 100% of traffic to the new version, pausing between steps. With a traffic router (Istio, Linkerd, NGINX Ingress, Gateway API, AWS ALB, SMI) the split is by weight rather than pod count.
- **Blue-green deployments**: bring up the full new ReplicaSet next to the old one, test it via a preview Service, then promote by flipping the active Service selector.
- **Metric-gated automated rollback**: `AnalysisTemplate`s query Prometheus, Datadog, New Relic, CloudWatch, Kayenta, or arbitrary web endpoints/Jobs. A failed analysis aborts the rollout and scales the stable version back.
- **Experiments**: run short-lived ReplicaSets of alternate versions (A/B/baseline-vs-canary) alongside the stable one for comparison.

**When a team adopts it:** when rolling updates are no longer a sufficient safety net — you serve meaningful traffic, outages from bad releases are costly, and you already have SLO-quality metrics you could gate on. It pairs naturally with Argo CD (GitOps) but works standalone. If you have few services or no reliable metrics, the added CRD complexity is probably not worth it yet.

## Getting started

**1. Install the controller and kubectl plugin**

```bash
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
brew install argoproj/tap/kubectl-argo-rollouts   # or download the binary from the releases page
```

**2. Define a canary `Rollout` (a drop-in replacement for `Deployment`)**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: checkout
spec:
  replicas: 5
  selector:
    matchLabels: { app: checkout }
  template:
    metadata:
      labels: { app: checkout }
    spec:
      containers:
        - name: app
          image: registry.example.com/checkout:v2
          ports: [{ containerPort: 8080 }]
  strategy:
    canary:
      steps:
        - setWeight: 10
        - pause: { duration: 5m }
        - analysis:
            templates: [{ templateName: success-rate }]
        - setWeight: 50
        - pause: {}          # wait for manual promotion
```

**3. Add an automated gate with an `AnalysisTemplate`**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  metrics:
    - name: success-rate
      interval: 1m
      count: 5
      failureLimit: 1
      successCondition: result[0] >= 0.99
      provider:
        prometheus:
          address: http://prometheus.monitoring:9090
          query: |
            sum(rate(http_requests_total{app="checkout",code!~"5.."}[2m]))
            / sum(rate(http_requests_total{app="checkout"}[2m]))
```

**Operate it with the plugin:**

```bash
kubectl argo rollouts get rollout checkout --watch   # live tree view of ReplicaSets, steps, analysis
kubectl argo rollouts set image checkout app=registry.example.com/checkout:v3
kubectl argo rollouts promote checkout               # advance past a manual pause
kubectl argo rollouts abort checkout                 # roll back to stable
kubectl argo rollouts dashboard                      # local web UI
```

## Common pitfalls

- **Without a traffic router, canary weight is approximate.** It is implemented by pod ratios, so 10% of 5 replicas rounds oddly. Use a supported router (and set `trafficRouting`) for exact splits, and size `replicas` accordingly.
- **Migrating from `Deployment`**: the controller does not touch `Deployment`s. Either use `workloadRef` to reference the existing Deployment (and scale it to 0 once the Rollout is healthy), or convert the manifest. Running both at once double-provisions pods.
- **Bad or empty metrics pass silently.** A Prometheus query returning no data yields an inconclusive/error result, not a clear failure — set `inconclusiveLimit`, `failureLimit`, and `consecutiveErrorLimit` deliberately, and test queries against a known-bad release.
- **Analysis windows must be long enough to see traffic.** A 1-minute check at low canary weight on a low-QPS service gives noisy, meaningless statistics.
- **GitOps interaction**: Argo CD may show a Rollout as "Progressing" or fight an in-flight abort. Enable the Rollout health checks/resource customizations and avoid committing a revert while a rollout is paused unless that is intended.
- **Stateful concerns are not solved**: schema migrations and non-backward-compatible API changes still need expand/contract discipline, because old and new versions serve simultaneously.
- **HPA and scaling**: point the HPA at the `Rollout` (not a Deployment) and remember canary pods count toward capacity; scale-down delays (`scaleDownDelaySeconds`) in blue-green keep old pods around longer than you may expect.
