# Kustomize

Kustomize is a template-free configuration customization tool for Kubernetes that lets you layer patches and overlays on top of plain, valid YAML manifests, so one base definition can be reused across environments without forking or templating it.

## Why teams adopt it

Raw manifests duplicate quickly across dev, staging, and prod, and templating engines (Helm, Jinja, envsubst) turn YAML into a string-substitution problem that is hard to lint, diff, and review. Kustomize keeps the base as real Kubernetes YAML and expresses differences as declarative transformations. It is built into `kubectl` (`kubectl apply -k`) and is the default rendering engine for Argo CD and Flux alongside Helm.

Common adoption triggers:
- Multiple environments or clusters share 90% of their config and differ in replicas, resource limits, image tags, or hostnames.
- A platform team ships a golden base (Deployment, Service, NetworkPolicy) that product teams customize without copying it.
- You consume third-party manifests (an upstream operator's `install.yaml`) and need to patch them without maintaining a fork.
- GitOps workflows where the rendered output must be deterministic and reviewable in PRs.
- You want to post-process Helm output (`helmCharts` generator) with org-specific patches rather than forking a chart.

Core concepts: a **base** (plain manifests plus a `kustomization.yaml`), **overlays** (directories that reference a base and add patches), **transformers** (namePrefix, namespace, labels, images, replicas), **generators** (ConfigMap/Secret from files or literals, with content-hash suffixes that trigger rollouts), and **components** (reusable, optional feature bundles).

## Basic usage

**1. Layout with a base and a production overlay:**
```
app/
  base/
    deployment.yaml
    service.yaml
    kustomization.yaml
  overlays/
    prod/
      kustomization.yaml
      replica-patch.yaml
```
```yaml
# base/kustomization.yaml
resources:
  - deployment.yaml
  - service.yaml
```
```yaml
# overlays/prod/kustomization.yaml
resources:
  - ../../base
namespace: shop-prod
namePrefix: prod-
images:
  - name: shop-api
    newName: registry.example.com/shop-api
    newTag: "1.8.2"
patches:
  - path: replica-patch.yaml
    target:
      kind: Deployment
      name: shop-api
```
```bash
kubectl kustomize overlays/prod        # render to stdout, no cluster needed
kubectl apply -k overlays/prod         # render and apply
```

**2. Inline patches (strategic merge or JSON 6902) and generators:**
```yaml
# overlays/prod/kustomization.yaml (continued)
configMapGenerator:
  - name: app-config
    literals:
      - LOG_LEVEL=warn
patches:
  - target: { kind: Deployment, name: shop-api }
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/0/resources/limits/memory
        value: 1Gi
```
The generated ConfigMap gets a hash suffix (`app-config-7f2k9d`), and references to it are rewritten automatically, so changing config rolls the pods.

**3. Review changes before shipping, and patch a remote base:**
```bash
kustomize build overlays/prod | kubectl diff -f -
kustomize build overlays/prod | kubectl apply --server-side -f -

# Pin an upstream manifest by ref and patch it locally
# resources:
#   - https://github.com/example/operator//config/default?ref=v1.4.0
```

## Pitfalls and things to watch

- **Always pin remote bases** to a tag or commit SHA; an unpinned `?ref=main` makes builds non-reproducible and is a supply-chain risk.
- **Patch target mismatches fail silently or confusingly.** A patch that doesn't match name/kind/namespace after `namePrefix` or `namespace` transforms is either an error or a no-op depending on form; always inspect `kustomize build` output.
- **Transformers apply broadly.** `commonLabels` also rewrites selectors, which are immutable on Deployments and will break upgrades of existing workloads; prefer `labels` with `includeSelectors: false`.
- **Overlay sprawl.** Deep chains (base -> region -> env -> team) become hard to reason about. Keep depth to two or three levels and use components for orthogonal features.
- **Secrets in generators.** `secretGenerator` produces base64, not encryption. Pair with SOPS, External Secrets, or sealed secrets instead of committing values.
- **Not a package manager.** There is no versioning, dependency resolution, or values schema. If you need parameterized, distributable packages with a contract, Helm fits better; many teams use both (Helm for third-party charts, Kustomize for last-mile org patches).
- **Version skew.** The Kustomize bundled in `kubectl` lags standalone releases; pin the standalone binary in CI so local and pipeline renders match.
- **Debugging.** Use `kustomize build --enable-helm` only with Helm installed, and diff rendered output in CI to catch unintended changes.
