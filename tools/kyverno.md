# Kyverno

Kyverno is a Kubernetes-native policy engine that validates, mutates, generates, and verifies resources using policies written as plain Kubernetes YAML — no new language to learn.

## Why teams adopt it

Platform and security engineers adopt Kyverno when "please follow our conventions" in a wiki stops scaling across dozens of teams and clusters. It runs as a dynamic admission controller, so every create/update request to the API server is checked against policy before it is persisted. Where OPA/Gatekeeper requires Rego, Kyverno policies are CRDs (`ClusterPolicy` / `Policy`) that look like the resources they govern, which lowers the barrier for teams who don't want to become policy-language experts.

Common adoption triggers:
- **Guardrails (validate):** block privileged containers, `:latest` tags, missing resource limits, or images from unapproved registries.
- **Defaults (mutate):** automatically inject labels, `securityContext`, node selectors, or sidecar config so developers don't have to remember.
- **Self-service (generate):** create a `NetworkPolicy`, `ResourceQuota`, or `RoleBinding` automatically whenever a new Namespace appears.
- **Supply chain (verifyImages):** require container images to be signed with Cosign/Sigstore and carry attestations (SBOM, provenance) before admission.
- **Audit-first rollout:** run policies in `Audit` mode to get PolicyReports on existing workloads, then flip to `Enforce`.

## Basic usage

**1. Install with Helm:**
```bash
helm repo add kyverno https://kyverno.github.io/kyverno/
helm install kyverno kyverno/kyverno -n kyverno --create-namespace
# For production, use the HA chart values (3 admission-controller replicas)
```

**2. A validate policy: require resource limits and ban `:latest`:**
```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-limits-and-pinned-tags
spec:
  validationFailureAction: Audit   # switch to Enforce once reports are clean
  background: true                 # also scan existing resources
  rules:
    - name: no-latest-tag
      match:
        any:
          - resources: { kinds: [Pod] }
      validate:
        message: "Images must use a pinned tag, not ':latest'."
        pattern:
          spec:
            containers:
              - image: "!*:latest"
    - name: require-memory-limit
      match:
        any:
          - resources: { kinds: [Pod] }
      validate:
        message: "Containers must set a memory limit."
        pattern:
          spec:
            containers:
              - resources:
                  limits:
                    memory: "?*"
```

**3. Mutate and generate, then test offline in CI:**
```yaml
# Generate a default-deny NetworkPolicy for every new namespace
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: default-deny-netpol
spec:
  rules:
    - name: gen-deny-all
      match:
        any:
          - resources: { kinds: [Namespace] }
      generate:
        apiVersion: networking.k8s.io/v1
        kind: NetworkPolicy
        name: default-deny
        namespace: "{{request.object.metadata.name}}"
        synchronize: true   # recreate if someone deletes it
        data:
          spec:
            podSelector: {}
            policyTypes: [Ingress, Egress]
```
```bash
# Test policies against manifests without a cluster (works in CI)
kyverno apply policy.yaml --resource deployment.yaml
kyverno test ./policies        # runs declarative test cases (kyverno-test.yaml)

# Inspect results on a live cluster
kubectl get policyreport -A
kubectl get clusterpolicyreport
```

## Pitfalls and things to watch

- **Start in Audit, not Enforce.** Enforcing on day one will block deployments (including system components and operators) you didn't know existed. Use PolicyReports to find violators first.
- **Webhook availability is cluster availability.** If the admission webhook is down and `failurePolicy: Fail`, resource creation can stall. Run HA replicas, set sensible timeouts, and exclude `kube-system` and Kyverno's own namespace.
- **Pods vs. controllers.** Matching `Pod` alone is fine, but validation failures then surface at ReplicaSet/Pod creation, far from the Deployment the user applied. Kyverno auto-generates rules for Pod controllers (Deployment, StatefulSet, Job…) by default; verify this behaves as expected for your CRDs, or users get confusing errors.
- **Mutation ordering and idempotency.** Mutate policies run before validate; multiple mutations can conflict. Keep them idempotent, and remember mutating live objects can cause GitOps drift (Argo CD / Flux may show perpetual diffs — configure ignore rules).
- **Generate + synchronize is powerful and sharp.** `synchronize: true` overwrites manual changes to generated resources; deleting the policy can delete the generated objects (depending on `orphanDownstreamOnPolicyDelete`).
- **API-call/context lookups add latency.** Policies that call the API server or external registries (e.g., `verifyImages`) sit in the admission path; cache aggressively and watch webhook latency metrics.
- **Resource footprint at scale.** Background scans and large PolicyReports can pressure etcd and the reports controller on big clusters; tune scan intervals and report retention.
- **Know when it's not the right tool.** For policy beyond Kubernetes (Terraform plans, API authz, CI gates), OPA is more general; Kyverno is best when Kubernetes YAML is the whole domain.
