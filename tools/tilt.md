# Tilt

Tilt is a local development orchestrator that watches your code, rebuilds and live-updates your services in a Kubernetes (or Docker Compose) environment, and shows build/runtime status in one dashboard — so the inner dev loop for multi-service systems stays fast.

## Why teams adopt it

Once an application is a dozen services plus databases, queues, and sidecars, "run it locally" turns into a pile of `docker build`, `kubectl apply`, `port-forward`, and log-tailing terminals. The classic loop of edit → build image → push → redeploy → wait can take minutes, and teams respond by either mocking everything (and missing integration bugs) or testing only in shared staging (and stepping on each other).

Tilt keeps your real manifests in the loop but shortens the cycle: it syncs changed files straight into running containers or does incremental image builds, restarts only the affected process, and surfaces errors in a web UI.

Teams typically reach for it when:
- Platform engineers want a standard, versioned "one command to get a working dev environment" that uses the same Helm charts/Kustomize/manifests as production.
- Microservice teams find that `docker compose` diverges from how things actually run on Kubernetes.
- Onboarding takes days because the setup lives in a wiki rather than in code.
- Developers work against a shared remote dev cluster, or a local one (kind, minikube, Docker Desktop, k3d), and want fast feedback either way.

It is a dev-loop tool, not a deployment tool: it should not replace Argo CD/Flux or your CI pipeline.

## Basic usage

**1. Create a local cluster and a minimal `Tiltfile`** (Tiltfiles are written in Starlark, a Python dialect):
```bash
brew install tilt          # or: curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
ctlptl create cluster kind --registry=ctlptl-registry   # local kind cluster + registry
```
```python
# Tiltfile
docker_build('myorg/api', './api')
k8s_yaml('deploy/api.yaml')
k8s_resource('api', port_forwards=8080, labels=['backend'])
```
Run `tilt up`; it builds the image, applies the manifest, forwards the port, and opens the dashboard at http://localhost:10350.

**2. Live-update instead of rebuilding images:**
```python
docker_build(
  'myorg/api', './api',
  live_update=[
    sync('./api/src', '/app/src'),                       # copy changed files into the container
    run('pip install -r requirements.txt',               # only when deps change
        trigger=['./api/requirements.txt']),
  ],
)
```
Saving a source file now updates the running container in about a second, with no image rebuild or pod restart (for interpreted languages, or compiled ones via a restart wrapper).

**3. Bring in Helm, local commands, and dependencies:**
```python
k8s_yaml(helm('charts/postgres', name='db', values=['dev-values.yaml']))
local_resource('migrate', cmd='./scripts/migrate.sh',
               resource_deps=['db'], labels=['setup'])
k8s_resource('api', resource_deps=['migrate'])
```
`resource_deps` gives you start-up ordering, and `local_resource` runs host commands (codegen, tests, linters) as first-class, observable resources. In CI, `tilt ci` brings the environment up, waits until everything is healthy, and exits non-zero on failure.

## Common pitfalls

- **Pointing at the wrong cluster.** Tilt refuses to deploy to contexts it doesn't recognise as local unless you opt in with `allow_k8s_contexts(...)`. Don't blanket-allow production contexts; a misconfigured kubeconfig plus `tilt up` can apply manifests where you didn't intend.
- **Tiltfile sprawl.** It is real code. Without shared helper functions (`load('ext://...')` or an internal library) and review, each team's Tiltfile diverges and becomes unmaintainable.
- **Live update fragility.** `sync` plus `run` steps can leave containers in a state that differs from a clean image build. Periodically verify with a full rebuild, and make sure the `Dockerfile` path stays what CI uses.
- **Resource consumption.** Running the whole stack locally is heavy on laptops. Use `tilt up -- <args>` or `config.define_string_list` to start only the services a developer needs, and point the rest at a shared environment.
- **Overlapping file watches.** Large monorepos or `node_modules` can slow file watching; use `.tiltignore` and `.dockerignore` aggressively to stop spurious rebuild loops (e.g. generated files written into watched directories).
- **Drift from production.** Dev-only overrides (relaxed resource limits, disabled auth) accumulate in the Tiltfile. Keep them in clearly marked overlays so the base manifests stay identical to what ships.
