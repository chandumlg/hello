# Cilium

Cilium is an open-source networking, observability, and security layer for Kubernetes (and other container platforms) that uses eBPF — small sandboxed programs that run inside the Linux kernel — instead of iptables to route, filter, and inspect traffic between pods.

## Why it exists

Kubernetes needs something to actually move packets between pods across nodes (a CNI plugin), and by default most clusters do this with iptables-based kube-proxy rules and a basic CNI like Flannel or Calico's legacy mode. That works, but iptables rule chains scale linearly with the number of services and pods, so on large clusters (thousands of services) packet processing latency and CPU overhead climb noticeably, rule updates get slow, and you get very little visibility into what's actually happening at L3/L4/L7 without bolting on a service mesh. eBPF lets Cilium hook directly into the kernel's networking path, replace iptables with O(1) hash-table lookups, and — because it can see every packet — also give you fine-grained network policy, deep flow-level observability, and even service-mesh-like L7 traffic management without needing sidecar proxies for every pod.

## Primary use cases and when to adopt it

- **CNI replacement at scale**: teams running large multi-tenant clusters (hundreds+ nodes) hit iptables performance walls and switch to Cilium for its eBPF datapath, which scales far better and eliminates kube-proxy entirely (`kube-proxy replacement` mode).
- **Kubernetes network policy done properly**: Cilium's `CiliumNetworkPolicy` CRD supports L3/L4 rules like standard `NetworkPolicy`, but also L7-aware rules (e.g., "this pod may only issue HTTP GET to `/api/orders`" or "may only run specific SQL statements against this Postgres"), and identity-based policy that doesn't break when pod IPs churn.
- **Sidecar-free service mesh (Cilium Mesh / "sidecar-less" mesh)**: using eBPF plus an optional per-node Envoy, Cilium can provide mTLS, L7 routing, and traffic management without injecting a sidecar into every pod — attractive to platform teams who found Istio's sidecar overhead too costly.
- **Deep network observability via Hubble**: Cilium ships with Hubble, which gives you a live, queryable flow log of every connection in the cluster (who talked to whom, over what protocol, was it allowed or dropped) — useful for debugging connectivity issues and for security/compliance audits, without needing to instrument application code.
- **Multi-cluster / multi-cloud networking**: Cilium Cluster Mesh connects the pod networks of multiple Kubernetes clusters so services can discover and call each other across clusters as if local.

A team typically adopts Cilium when: they're outgrowing kube-proxy/iptables performance, they need real network policy enforcement (not just permissive defaults), they want mesh-like features without the operational cost of sidecars everywhere, or they need cluster-wide traffic visibility for debugging or security review.

## Basic usage

**1. Install Cilium into a cluster (via the CLI, on top of an existing kubeconfig context):**
```bash
cilium install --version 1.16.0
cilium status --wait
```

**2. Check connectivity and run the built-in test suite:**
```bash
cilium connectivity test
```

**3. Write a network policy restricting traffic to a specific pod, allowing only GET requests to `/public` on port 80 from pods labeled `role=frontend`:**
```yaml
apiVersion: "cilium.io/v2"
kind: CiliumNetworkPolicy
metadata:
  name: allow-frontend-get-public
spec:
  endpointSelector:
    matchLabels:
      app: backend
  ingress:
    - fromEndpoints:
        - matchLabels:
            role: frontend
      toPorts:
        - ports:
            - port: "80"
              protocol: TCP
          rules:
            http:
              - method: "GET"
                path: "/public"
```
Apply it with `kubectl apply -f policy.yaml`.

**4. Observe live traffic flows with Hubble (after enabling it: `cilium hubble enable`):**
```bash
hubble observe --namespace default --verdict DROPPED
```
This shows every dropped packet in the namespace in real time, with source/destination identity and the policy that caused the drop — invaluable when a `NetworkPolicy` change unexpectedly breaks something.

## Common pitfalls

- **Kernel version matters.** eBPF features Cilium relies on (especially kube-proxy replacement and L7 policy) require a reasonably modern Linux kernel (5.x+ recommended for full functionality). Older kernels on managed node images can silently disable features or force fallback modes — check `cilium status` for warnings, don't assume everything installed cleanly.
- **Migrating an existing cluster off another CNI is disruptive.** Swapping CNIs generally requires rolling every node (pod networking is node-level state), so this is a maintenance-window operation, not a rolling no-downtime change, unless you carefully follow Cilium's documented migration path.
- **L7 policies add a proxy in the path.** L7-aware `CiliumNetworkPolicy` rules route matching traffic through an embedded Envoy instance, which adds latency and a new failure mode compared to pure L3/L4 eBPF enforcement — don't reach for L7 policy for cases plain port/protocol rules would cover.
- **Default-deny surprises.** Once you apply your first `CiliumNetworkPolicy` (or `NetworkPolicy`) selecting a pod, all traffic not explicitly allowed is dropped for that pod — teams that test policies against a couple of known flows often get surprised later by an unrelated dependency (DNS, a metrics scraper, a health check) getting silently blocked. Always allow DNS (`kube-dns`/CoreDNS) explicitly, and use Hubble to check for unexpected drops after any policy rollout.
- **Cluster Mesh and kube-proxy replacement both require careful MTU/routing planning** (especially with overlay vs. native routing modes and cloud provider networking); getting this wrong shows up as intermittent connectivity issues that are hard to diagnose without Hubble.
- **Upgrades need attention.** Because Cilium replaces core networking, minor version upgrades should be tested in a non-critical cluster first and done via the documented rolling upgrade procedure — skipping versions or upgrading carelessly can cause connectivity gaps.
