# Open Policy Agent (OPA)

Open Policy Agent (OPA, pronounced "oh-pa") is an open-source, general-purpose policy engine that lets you write authorization and validation rules as code in a dedicated language called Rego, then enforce those rules consistently across services, Kubernetes clusters, CI pipelines, and infrastructure-as-code — instead of scattering hand-rolled `if` statements across every system that needs to make an access-control or compliance decision.

## Why teams adopt it

Staff, platform, and security engineers reach for OPA when the same class of question — "is this action allowed?" — keeps getting answered differently (and inconsistently) across a growing number of systems: one team hardcodes RBAC checks in a microservice, another writes bash scripts to lint Terraform, another manually reviews Kubernetes manifests for missing resource limits. OPA decouples the *decision* ("can user X do Y to resource Z?") from the *enforcement point* (the app, the API gateway, the admission controller), so you write the policy once in Rego and query it from anywhere over a simple API. Because it's declarative and data-driven, policies can be version-controlled, unit-tested, and reviewed like any other code — turning tribal knowledge about "what's allowed" into an auditable artifact.

Common adoption triggers:
- **Kubernetes admission control** — blocking pods that run as root, lack resource limits, use `:latest` image tags, or violate naming/labeling conventions, typically via Gatekeeper or Kyverno-style integration.
- **Microservice authorization** — centralizing fine-grained access control (who can call which API, with which parameters) instead of duplicating auth logic in every service.
- **Infrastructure-as-code guardrails** — evaluating Terraform plans in CI to reject changes that open security groups to the world, disable encryption, or exceed cost/resource thresholds before they're applied.
- **Compliance-as-code** — encoding regulatory or internal security requirements (e.g., "all S3 buckets must have encryption enabled") as testable policies that run automatically in pipelines rather than relying on periodic manual audits.

## Basic usage

**1. Write a policy in Rego** that denies Kubernetes pods without resource limits:
```rego
package kubernetes.admission

deny[msg] {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    not container.resources.limits
    msg := sprintf("container '%v' must specify resource limits", [container.name])
}
```

**2. Evaluate the policy locally with the `opa` CLI** against a sample input:
```bash
# eval.sh — load the policy and test it against a JSON input document
opa eval --data policy.rego --input pod.json "data.kubernetes.admission.deny"
```

**3. Run OPA as a server and query it over HTTP** (how most production integrations work — e.g., an API gateway calling out on every request):
```bash
# Start OPA as a daemon, loading policies from the ./policies directory
opa run --server --addr :8181 ./policies

# From your application, POST the request context and get an allow/deny decision
curl -X POST localhost:8181/v1/data/kubernetes/admission/deny \
  -d '{"input": {"request": {"kind": {"kind": "Pod"}, "object": {...}}}}'
```

## Pitfalls to watch out for

- **Rego has a real learning curve.** It's a declarative, Datalog-derived logic language, not an imperative scripting language — engineers used to writing `if/else` chains often fight the "rules that all must be satisfied" mental model at first. Budget time for the team to get comfortable before rolling it out broadly.
- **Policy sprawl without testing becomes its own liability.** Rego supports unit tests (`opa test`) and they're easy to skip under deadline pressure — untested policies that silently deny (or worse, silently allow) traffic are hard to debug in production. Treat policies as code requiring the same review/test discipline as application logic.
- **Performance depends on how you query it.** Evaluating large, deeply nested `input`/`data` documents on every request can add latency; for high-throughput authorization paths, use partial evaluation, bundle caching, or a sidecar deployment pattern rather than a network hop to a shared OPA service.
- **It's a decision engine, not an enforcement mechanism.** OPA tells you "allow" or "deny" — it doesn't stop anything by itself. You must correctly wire it into the actual enforcement point (admission webhook, gateway middleware, CI gate); a misconfigured integration point means the policy is evaluated but never actually acted on.
- **Data staleness in distributed deployments.** When OPA instances pull policy/data bundles periodically rather than synchronously, there's a window where different instances can enforce slightly different rules — factor this into how frequently you push bundle updates for security-sensitive policies.
