# uv

uv is an extremely fast Python package and project manager (written in Rust, from Astral) that replaces pip, pip-tools, virtualenv, pipx, and pyenv with a single tool and a universal lockfile, so dependency resolution, environment creation, and Python version management take milliseconds instead of minutes.

## Why teams adopt it

Python environments are a perennial source of "works on my machine" pain, slow CI, and bloated Docker builds. uv gives platform and AI/ML engineers a reproducible, cross-platform workflow (`pyproject.toml` + `uv.lock`) that is typically 10-100x faster than pip. It also manages interpreter installs, runs one-off tools, and supports monorepo workspaces.

Common adoption triggers:
- CI or Docker builds spend minutes on `pip install`, and you want cacheable, deterministic installs.
- ML repos with heavy, fragile dependency trees (torch, CUDA wheels) need a lockfile that is reproducible across macOS/Linux and across teammates.
- You want one tool for "install Python 3.12, create venv, add dependency, run script" instead of pyenv + venv + pip + pipx.
- You maintain a monorepo of several Python packages and want shared locking via workspaces.
- You need self-contained scripts with inline dependencies for ops/tooling.

## Basic usage

**1. Start a project and manage dependencies (lockfile is created automatically):**
```bash
uv init myservice && cd myservice
uv python pin 3.12          # uv downloads the interpreter if missing
uv add fastapi "httpx>=0.27"
uv add --dev pytest ruff
uv run pytest               # syncs .venv to uv.lock, then runs
```

**2. Reproducible installs in CI / Docker:**
```bash
uv sync --locked            # fail if uv.lock is out of date with pyproject.toml
uv sync --frozen --no-dev   # install exactly what's locked, skip dev deps
```
```dockerfile
COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-install-project --no-dev   # cached dependency layer
```

**3. Run tools and single-file scripts without polluting your environment:**
```bash
uvx ruff check .                      # like pipx run: ephemeral isolated tool env
uv tool install httpie                # persistent global CLI tool
uv run --with rich script.py          # add an ad-hoc dependency for one run
```
Inline script metadata (PEP 723) lets a script declare its own deps:
```python
# /// script
# requires-python = ">=3.11"
# dependencies = ["requests"]
# ///
```
then `uv run script.py`.

## Pitfalls and things to watch

- **Pin the uv version** in CI and Docker (use a tag rather than `latest`); the tool evolves quickly and lockfile format or behavior can shift between releases.
- **Commit `uv.lock`** and use `--locked`/`--frozen` in CI. Plain `uv sync` may silently re-lock.
- **`uv pip` is not identical to pip**: it is a compatible interface, but resolution semantics differ (e.g., index handling), so migrate deliberately.
- **Multiple indexes and dependency confusion**: uv's default strategy is `first-index`, not "best match across all indexes"; configure `[[tool.uv.index]]` explicitly (and `explicit = true`) for private or PyTorch CUDA indexes.
- **Universal resolution** locks for all platforms, which can fail if a package lacks wheels for some platform; restrict with `tool.uv.environments`.
- **Build isolation**: packages needing build-time deps (older C extensions) may need `--no-build-isolation` or `extra-build-dependencies` configuration.
- **System Python vs managed Python**: uv prefers its own downloaded interpreters (python-build-standalone) which can differ subtly from distro Pythons; use `--python-preference system` if required.
- **Cache location and hardlinks**: the global cache uses hardlinks/reflinks; cross-filesystem setups (Docker mounts) fall back to copying and warn. Set `UV_LINK_MODE=copy` to silence it.
