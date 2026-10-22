# Linkerd

Linkerd is an ultralight, security-focused Kubernetes service mesh that transparently adds mutual TLS, golden-signal metrics, retries, timeouts, and traffic splitting to your services using a purpose-built Rust sidecar proxy (`linkerd2-proxy`) — without changing application code.

## Why teams adopt it

Once a cluster runs dozens of services, questions like "is service-to-service traffic encrypted?", "which hop is adding latency?", and "can we shift 5% of traffic to the new version?" stop being answerable by individual teams. Linkerd answers them at the platform layer. Compared with Istio (see `istio.md`), it deliberately trades breadth of features for operational simplicity, a small resource footprint, and sane defaults.

Common adoption triggers:
- Compliance or zero-trust requirements demand mTLS between all pods, with automatic certificate rotation and workload identity.
- You need consistent request success rate, latency percentiles (p50/p95/p99), and throughput per route without instrumenting every service.
- You want progressive delivery (canaries, blue/green) or per-route retries/timeouts as declarative config, not library code.
- Platform teams want multi-cluster service communication with a smaller operational burden than a full-featured mesh.
- Sidecar overhead matters: the Rust proxy typically uses far less CPU/memory than Envoy-based sidecars.

## Basic usage

**1. Install the CLI, validate the cluster, and install the control plane:**
```bash
curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install-edge | sh
linkerd check --pre
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -
linkerd check
```
(Production setups usually use the Helm charts instead and bring their own trust anchor / issuer certificates from cert-manager — see `cert-manager.md`.)

**2. Mesh a workload by annotating its namespace, then inspect traffic:**
```bash
kubectl annotate namespace shop linkerd.io/inject=enabled
kubectl rollout restart deploy -n shop

linkerd viz install | kubectl apply -f -   # optional metrics dashboard
linkerd viz stat deploy -n shop            # success rate, RPS, latency
linkerd viz tap deploy/checkout -n shop    # live request stream
linkerd viz edges deployment -n shop       # confirms mTLS between peers
```

**3. Restrict access with policy (only `checkout` may call `payments`):**
```yaml
apiVersion: policy.linkerd.io/v1beta1
kind: Server
metadata: { name: payments-grpc, namespace: shop }
spec:
  podSelector: { matchLabels: { app: payments } }
  port: grpc
  proxyProtocol: gRPC
---
apiVersion: policy.linkerd.io/v1alpha1
kind: AuthorizationPolicy
metadata: { name: payments-from-checkout, namespace: shop }
spec:
  targetRef: { group: policy.linkerd.io, kind: Server, name: payments-grpc }
  requiredAuthenticationRefs:
    - kind: ServiceAccount
      name: checkout
      namespace: shop
      group: ""
```
Note that once a `Server` exists, traffic not matched by a policy is denied, so apply policies deliberately and test in a non-production namespace first.

## Pitfalls and things to watch

- **Licensing and distribution changed.** Since early 2024, the CNCF-hosted project no longer publishes stable release artifacts; stable builds come from Buoyant (free for small orgs, paid beyond a company-size threshold), while open source "edge" releases remain available. Check current terms before standardizing.
- **Trust anchor expiry is the classic outage.** Linkerd's default generated identity certificates are short-lived for the issuer, and an expired trust anchor/issuer breaks mTLS cluster-wide. Automate rotation (cert-manager) and alert on expiry.
- **Opaque and non-HTTP ports.** Protocol detection can stall on server-speaks-first protocols (MySQL, SMTP, Postgres in some setups). Mark such ports as opaque or skip them via annotations (`config.linkerd.io/opaque-ports`, `skip-outbound-ports`).
- **Sidecar lifecycle issues.** Jobs/CronJobs may hang because the proxy keeps the pod alive; use native sidecar support (Kubernetes 1.29+) or the `linkerd-await` pattern. Startup races can also cause early connection failures without proper ordering.
- **Load balancing is per-request, not per-connection.** That's a benefit for gRPC/HTTP2, but clients that pin long-lived connections outside the mesh won't see the effect; make sure workloads are actually meshed on both ends.
- **Not a full API gateway.** Linkerd handles east-west traffic; ingress, rate limiting, and WAF concerns still need a separate gateway (it integrates with Gateway API for routing).
- **Resource requests for the control plane and proxies** should be set explicitly; defaults in dev clusters rarely match production load.
