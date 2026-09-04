# Istio

Istio is a service mesh for Kubernetes that adds traffic control, mutual TLS, and deep observability to service-to-service calls without changing application code.

## What problem it solves

Once a system grows past a handful of services, the questions that matter stop being "does my code work" and start being "can service A reach service B, is that call encrypted, what happens if B is slow, and which version of C is actually receiving traffic right now." Handling that in application code means every team reimplements retries, timeouts, circuit breaking, and TLS setup in whatever language they happen to use, inconsistently and usually badly. Istio moves all of that into the network layer: it injects a small proxy (Envoy) alongside every pod, and every request between services flows through that proxy instead of going directly service-to-service. Because the proxies see every request, Istio can enforce policy, encrypt traffic, and collect metrics uniformly across the whole fleet — regardless of what language or framework each service is written in.

## Primary use cases and when a team adopts it

- **mTLS everywhere with zero app changes**: Istio can automatically encrypt and authenticate all pod-to-pod traffic in a cluster, which is the usual first reason platform teams reach for it — satisfying a security/compliance requirement ("all internal traffic must be encrypted") without touching every service.
- **Traffic shaping for progressive delivery**: canary rollouts, blue/green deploys, and A/B testing by splitting traffic by percentage or by header (e.g. send 5% of traffic, or all requests with `x-canary: true`, to the new version of a service) — used heavily alongside tools like Argo Rollouts.
- **Resilience without library sprawl**: retries, timeouts, circuit breaking, and outlier detection (automatically ejecting an unhealthy pod from the load-balancing pool) configured declaratively per-service instead of coded into every client library.
- **Uniform observability**: golden-signal metrics (request rate, error rate, latency) and distributed trace propagation for every service in the mesh, even legacy or third-party services that were never instrumented — because the sidecar generates the telemetry, not the app.
- **Zero-trust authorization**: `AuthorizationPolicy` resources that say "service X may call service Y's `/admin` endpoint only," enforced at the proxy regardless of what the app itself would allow.

A team typically adopts Istio once they have enough services that inconsistent client-side retry/timeout/TLS logic has caused real incidents, or once a security/compliance mandate requires encrypted internal traffic and audited service-to-service access — and once they've accepted the operational cost of running a sidecar proxy next to every pod (or, more recently, Istio's sidecar-less "ambient mesh" mode, which trades some per-pod isolation for much lower resource overhead).

## Basic usage

**1. Install Istio into a cluster and enable sidecar injection for a namespace:**
```bash
istioctl install --set profile=demo -y
kubectl label namespace default istio-injection=enabled
# any pod created in `default` from now on gets an Envoy sidecar automatically
kubectl rollout restart deployment -n default
```

**2. Split traffic between two versions of a service (a 90/10 canary) using `VirtualService` and `DestinationRule`:**
```yaml
apiVersion: networking.istio.io/v1
kind: DestinationRule
metadata:
  name: reviews
spec:
  host: reviews
  subsets:
    - name: v1
      labels: { version: v1 }
    - name: v2
      labels: { version: v2 }
---
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: reviews
spec:
  hosts: [reviews]
  http:
    - route:
        - destination: { host: reviews, subset: v1 }
          weight: 90
        - destination: { host: reviews, subset: v2 }
          weight: 10
```

**3. Enforce strict mTLS for a namespace and restrict which service can call an endpoint:**
```yaml
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: payments
spec:
  mtls:
    mode: STRICT
---
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: allow-checkout-only
  namespace: payments
spec:
  selector:
    matchLabels: { app: payments-api }
  rules:
    - from:
        - source: { principals: ["cluster.local/ns/checkout/sa/checkout-sa"] }
```

**4. Inspect mesh traffic and config with `istioctl`:**
```bash
istioctl proxy-status                 # which proxies are in sync with the control plane
istioctl analyze                      # lint mesh config for misconfigurations
istioctl proxy-config routes <pod>    # see the routing rules a specific sidecar is using
```

## Common pitfalls

- **Sidecar resource overhead and startup ordering**: every pod now runs an extra Envoy container, which adds real CPU/memory cost across a large cluster and can cause "connection refused" errors right after pod startup if the app container starts talking to the network before the sidecar is ready (mitigated by Istio's `holdApplicationUntilProxyStarts` setting, or by moving to ambient mode).
- **Debugging becomes a two-layer problem**: a failing request might be an application bug or an Istio routing/mTLS/authorization misconfiguration, and it's easy to burn time in the wrong layer — `istioctl analyze` and `istioctl proxy-config` are the first things to check, not app logs.
- **STRICT mTLS breaks non-mesh callers**: turning on strict mTLS in a namespace will silently reject traffic from anything not in the mesh (a cron job outside the cluster, a health check from an external load balancer) unless it's explicitly exempted.
- **VirtualService/DestinationRule conflicts are easy to create by accident**: multiple teams editing routing rules for the same host without coordination leads to confusing "why is my traffic going to the wrong version" bugs — `istioctl analyze` catches most of these before they hit production.
- **Upgrades are a real operational task**: the control plane and every sidecar's Envoy version need to stay compatible, and skipping minor versions during upgrade is unsupported — treat Istio upgrades like a database migration, not a routine `helm upgrade`.
- **It's not a replacement for API gateway or ingress design**: Istio manages traffic *inside* the mesh; teams sometimes over-scope it to solve north-south (external) traffic problems that a simpler ingress controller would handle more simply.
