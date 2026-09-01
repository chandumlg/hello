# Bazel

Bazel is a build and test tool from Google that solves the problem of slow,
inconsistent builds in large codebases by making builds hermetic,
incremental, and cacheable down to the individual file.

## What problem it solves

Once a codebase grows past a certain size — many services, many languages,
many internal libraries depending on each other — tools like `make`,
per-language build systems (`npm`, `mvn`, `go build`), or shell scripts
start to break down. Builds get slow because they rebuild things that
haven't changed, "works on my machine" bugs creep in because the build
implicitly depends on whatever happens to be installed locally, and CI
takes longer every quarter as the repo grows. Bazel fixes this by requiring
every build and test target to declare its inputs and outputs explicitly,
then using that dependency graph to build only what changed, in parallel,
and to cache results (locally and remotely) so that identical inputs never
get rebuilt twice — even across machines and CI runs.

## Primary use cases and when a team adopts it

- **Monorepos with multiple languages.** Bazel treats Go, Java, Python,
  C++, JS/TS, Rust, and more as first-class citizens in one dependency
  graph, so a change to a shared library only rebuilds the services that
  actually depend on it.
- **Reproducible, hermetic builds.** Because Bazel sandboxes each build
  step and pins toolchains, "it built on CI but not locally" mostly stops
  happening — this matters a lot once you have dozens of engineers and
  multiple CI runners.
- **Remote caching and remote execution.** Teams with large build farms
  (or just a big CI bill) use Bazel's remote cache so that if one engineer
  or one CI job has already built a given target, nobody else has to
  rebuild it — this is often the single biggest lever for cutting CI time.
- **When to adopt it:** Bazel pays off once build times or build
  flakiness become a real tax on the team — typically past a few dozen
  services or a codebase spanning multiple languages. For a single small
  service, the standard language tooling is simpler and Bazel's setup cost
  isn't worth it.

## Basic usage

Install via Bazelisk (a version-manager wrapper that reads a
`.bazelversion` file, so every engineer and CI run uses the exact same
Bazel version):

```bash
# macOS
brew install bazelisk

# or download the binary directly and put it on PATH as `bazel`
```

A minimal `BUILD` file (Bazel's per-directory manifest) for a Go binary:

```python
# BUILD
load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "hello_lib",
    srcs = ["main.go"],
    importpath = "example.com/hello",
    deps = ["//pkg/greeting"],
)

go_binary(
    name = "hello",
    embed = [":hello_lib"],
)
```

Build and run a target by its label (`//path/to/package:target`):

```bash
bazel build //cmd/hello:hello
bazel run //cmd/hello:hello
bazel test //pkg/greeting:greeting_test
```

Query the dependency graph — e.g. find everything that would be affected
by a change to one package, which is how CI systems compute "only test
what changed":

```bash
bazel query 'rdeps(//..., //pkg/greeting:greeting)'
```

## Common pitfalls

- **The learning curve is real.** `BUILD` files, `WORKSPACE`/`MODULE.bazel`
  dependency management, and Starlark (Bazel's Python-like config
  language) are all new concepts for a team coming from `npm` or `go
  build`. Budget real ramp-up time, and consider `gazelle` or similar
  BUILD-file generators to avoid hand-writing every target.
- **Implicit dependencies break hermeticity.** If a build step reaches out
  to the network, reads an undeclared file, or depends on a locally
  installed tool Bazel doesn't know about, it'll work for you and fail (or
  silently cache wrong) for someone else. Sandbox violations are the most
  common source of "works here, not there" bugs in a Bazel repo.
- **Remote cache correctness.** A remote cache is only safe if build
  actions are truly deterministic — non-deterministic outputs (embedded
  timestamps, unordered map iteration in codegen, etc.) can poison the
  cache and produce hard-to-debug "stale artifact" bugs across the whole
  team.
- **Migrating an existing large repo is a project, not a weekend task.**
  Most teams migrate incrementally, language by language or service by
  service, rather than converting everything at once.
- **`WORKSPACE` is being replaced by `MODULE.bazel`/Bzlmod.** New
  projects should start on Bzlmod; older tutorials and examples online
  often still show the legacy `WORKSPACE` syntax, which can be confusing
  when mixed together.
