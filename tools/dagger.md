# Dagger

Dagger is a programmable CI/CD engine that lets you define build, test, and deploy pipelines as real code (Go, Python, TypeScript, and others) running in containers via BuildKit, so the exact same pipeline executes identically on a laptop and in any CI system instead of living in vendor-specific YAML.

## Primary use cases

- **Portable pipelines**: write the pipeline once as typed functions and run it locally (`dagger call ...`) or from GitHub Actions, GitLab CI, or Jenkins with a one-line invocation. This ends the "push and pray" loop of debugging YAML in CI.
- **Reproducible, cached builds**: every step runs in a container, and results are content-addressed and cached. Unchanged steps are skipped, locally and (with a shared cache) across CI runners.
- **Reusable platform building blocks**: platform teams publish Dagger modules (e.g., "build and scan a Go service") that product teams call from their own repos. The pipeline logic is versioned and testable like any library.
- **Complex, polyglot workflows**: fan-out test matrices, multi-service integration tests with ephemeral service containers, and monorepo builds are expressed with loops and functions instead of templated YAML.
- **AI/agent workflows**: Dagger's container-and-function model is also used to give LLM agents sandboxed environments and tools.

A team typically adopts Dagger when CI config has become a sprawl of copy-pasted YAML that nobody can run locally, when they're migrating between CI vendors, or when a platform team wants to ship pipelines as a versioned internal product.

## Basic usage

**1. Install and initialize a module**

```bash
curl -fsSL https://dl.dagger.io/dagger/install.sh | sh   # or: brew install dagger/tap/dagger
# Requires a container runtime (Docker, Podman, etc.) to host the engine

cd my-service
dagger init --sdk=python --name=ci     # scaffolds .dagger/ with a module
dagger functions                       # list callable functions
```

**2. Write a function** (`.dagger/src/ci/main.py`)

```python
import dagger
from dagger import dag, function, object_type

@object_type
class Ci:
    @function
    async def test(self, source: dagger.Directory) -> str:
        return await (
            dag.container()
            .from_("python:3.12-slim")
            .with_directory("/app", source)
            .with_workdir("/app")
            .with_mounted_cache("/root/.cache/pip", dag.cache_volume("pip"))
            .with_exec(["pip", "install", "-r", "requirements.txt", "pytest"])
            .with_exec(["pytest", "-q"])
            .stdout()
        )
```

**3. Run it locally, then in CI**

```bash
dagger call test --source=.            # runs in containers, caches layers + pip

# In GitHub Actions, the same call:
#   - uses: dagger/dagger-for-github@v7
#     with:
#       call: test --source=.
```

**4. Reuse a community module**

```bash
dagger -m github.com/<org>/<module> functions
dagger call -m github.com/<org>/<module> <function> --help
```

## Common pitfalls

- **It needs a container runtime and engine**: CI runners must allow Docker (or equivalent). Locked-down or rootless environments need extra setup, and the engine container is privileged.
- **Cache is the performance story, and it's ephemeral by default**: on fresh, stateless CI runners you get cold builds unless you provide a persistent engine, a shared cache backend, or Dagger Cloud. Without it, expect no speedup over plain Docker builds.
- **Only explicitly passed inputs exist**: functions run in isolated containers, so secrets, env vars, and host directories must be passed as typed arguments (`dagger.Secret`, `dagger.Directory`). Use `Secret` types rather than plain strings so values aren't leaked into logs or cache keys.
- **Lazy evaluation surprises**: pipeline steps are a DAG that executes only when something is awaited or its output is requested (e.g., `.stdout()`, `.sync()`). A step with no consumer silently never runs.
- **Large directory uploads**: passing `--source=.` uploads the host directory to the engine. Use `ignore` filters (`.dockerignore`-style annotations) to avoid shipping `node_modules` or `.git` every run.
- **Younger ecosystem and API churn**: the module system and SDKs have changed significantly over versions. Pin the Dagger version in CI and expect to read release notes on upgrades.
- **Overkill for tiny projects**: if your CI is three `make` targets, a plain workflow file is simpler. Dagger pays off with complexity, reuse, or a need for local reproducibility.
