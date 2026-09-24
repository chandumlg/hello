# Chaos Mesh

**What it is:** Chaos Mesh is a cloud-native chaos engineering platform for Kubernetes that lets you deliberately inject failures — pod crashes, network latency, disk I/O errors, clock skew, and more — into running workloads to verify your system actually survives the failures you assume it handles.

## Why teams adopt it

Most outages come from failure modes nobody tested: a dependency that's slow instead of down, a node that gets network-partitioned, a disk that fills up. Chaos Mesh turns "what if X fails?" from a hallway debate into a repeatable, version-controlled experiment defined as Kubernetes custom resources (CRDs), so chaos experiments live in git next to the manifests they test.

Typical adopters are platform and SRE teams running production or pre-prod workloads on Kubernetes who want to:
- Validate that retries, circuit breakers, and failover actually trigger under real fault conditions, not just in unit tests.
- Run scheduled "game days" or continuous chaos in staging as part of CI/CD, catching regressions in resilience before they ship.
- Build confidence ahead of a migration (new CNI, new storage class, multi-AZ rollout) by simulating the failure modes the migration could introduce.

It's overkill for a team that hasn't yet nailed basic observability and alerting — you want to be able to *see* the blast radius of an experiment before you start causing them deliberately.

## Basic usage

Install via Helm into its own namespace:

```bash
helm repo add chaos-mesh https://charts.chaos-mesh.org
helm install chaos-mesh chaos-mesh/chaos-mesh -n chaos-mesh --create-namespace \
  --set chaosDaemon.runtime=containerd \
  --set chaosDaemon.socketPath=/run/containerd/containerd.sock
```

Kill a random pod matching a label selector every experiment run, to test that your deployment tolerates pod churn:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: kill-order-service-pod
  namespace: chaos-mesh
spec:
  action: pod-kill
  mode: one
  selector:
    namespaces: [production]
    labelSelectors:
      app: order-service
  scheduler:
    cron: "@every 10m"
```

Inject 200ms of latency plus jitter between a service and its database to test timeout and retry logic:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: db-latency
spec:
  action: delay
  mode: all
  selector:
    labelSelectors:
      app: order-service
  delay:
    latency: "200ms"
    jitter: "50ms"
  direction: to
  target:
    selector:
      labelSelectors:
        app: postgres
    mode: all
  duration: "5m"
```

Apply either with `kubectl apply -f experiment.yaml`, then watch effects with `kubectl get podchaos,networkchaos -n chaos-mesh` and your normal dashboards. Chaos Mesh also ships a web dashboard for building experiments visually and viewing real-time status.

## Pitfalls to watch for

- **Blast radius creep.** A loose label selector or `mode: all` instead of `mode: one`/`fixed-percent` can take out far more than intended — always scope selectors tightly and dry-run against a non-critical namespace first.
- **No automatic rollback on cluster failure.** If the Chaos Mesh controller pod itself gets killed or the cluster has an unrelated outage mid-experiment, some fault injections (especially `NetworkChaos` iptables/tc rules) can persist longer than the stated duration. Always have a way to force-delete the chaos CR and verify the underlying rule is actually cleared.
- **Requires elevated daemon privileges.** The `chaos-daemon` component runs privileged (to manipulate network namespaces, mount namespaces, etc.) on every node, which is a meaningful addition to your cluster's attack surface — restrict who can create Chaos Mesh CRDs via RBAC, since a PodChaos or NetworkChaos object is effectively a remote kill switch.
- **Running chaos in production before staging is stable.** If your staging environment can't survive basic pod-kill and network-delay experiments, running the same experiments in prod will just produce incidents, not insight. Start in staging/game-days, graduate to prod once responses (alerts, runbooks, auto-remediation) are proven.
- **Treating an experiment as one-and-done.** A resilience property validated once can regress silently after a refactor or config change. The value compounds when chaos experiments are scheduled recurring or wired into CI, not run manually and forgotten.
