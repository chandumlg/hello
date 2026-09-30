# Buf

Buf is a toolchain for Protocol Buffers that replaces ad-hoc `protoc` scripts with linting, breaking-change detection, code generation, and a schema registry, so API contracts stay consistent and safe to evolve.

## Why teams adopt it

Once a company has more than a handful of services talking over gRPC or exchanging protobuf messages (Kafka topics, event buses, mobile clients), `.proto` files become the most important shared contract in the org. Raw `protoc` makes that painful: plugin versions drift between laptops and CI, include paths are fragile, style is inconsistent, and nothing stops someone from renumbering a field and silently corrupting data for every consumer.

Buf fixes this with a declarative workflow:
- **Lint** enforces a style guide (naming, package layout, enum zero values) as code.
- **Breaking-change detection** compares your schema to a previous version (git branch, tag, or registry) and fails CI on wire- or source-incompatible edits.
- **Managed code generation** (`buf generate`) pins plugins and inputs in config, so every developer and CI job produces identical stubs.
- **Buf Schema Registry (BSR)** hosts modules, resolves dependencies (like `googleapis`), and can generate SDKs remotely.

Typical adopters: platform teams owning a shared API/schema repo, AI/data engineers using protobuf for event streams, and staff engineers introducing API governance without a heavy review bureaucracy.

## Basic usage

**1. Initialize a module and lint it:**
```bash
buf config init        # creates buf.yaml (older versions: buf mod init)
buf lint               # checks all .proto files against the configured rules
```
A minimal `buf.yaml`:
```yaml
version: v2
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

**2. Catch breaking changes against main before merging:**
```bash
buf breaking --against '.git#branch=main'
```
Deleting a field, changing a field's type, or reusing a field number will fail with a precise file/line message. Use `WIRE_JSON` or `WIRE` rulesets if you only care about on-the-wire compatibility, not generated-code compatibility.

**3. Generate code reproducibly:**
```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
```
```bash
buf generate
```
Because plugins are referenced by version (remote plugins or pinned local ones), output is deterministic across machines.

## Pitfalls to watch for

- **Choosing the wrong breaking ruleset.** `FILE` (strictest) flags moving a message between files, which is harmless on the wire; `WIRE` allows it. Pick the level matching your compatibility promise, including whether generated-code consumers exist.
- **Comparing against the wrong baseline.** `--against` main only works if CI fetches full history/the branch; shallow clones cause confusing failures. For released APIs, compare against the last tag or the BSR instead.
- **Adopting STANDARD lint on a legacy repo.** It will produce hundreds of findings. Start with `MINIMAL`/`BASIC`, or use `buf config ls-lint-rules`-guided `except`/ignore lists (or `buf lint --error-format=config-ignore-yaml`) to baseline, then tighten gradually.
- **Reserved fields.** Buf can't know you intended to retire a field number; always use `reserved` for removed fields/names, and enable the `FIELD_NO_DELETE_UNLESS_*_RESERVED` rules.
- **Remote plugin dependency.** Remote plugins/BSR require network access and add an external dependency to builds; vendor or pin local plugins for air-gapped or hermetic (e.g. Bazel) environments.
- **v1 vs v2 config.** `buf.yaml` v2 changed layout (workspaces defined in one root file); mixing versions across a monorepo or following older tutorials leads to confusing errors.
- **Lint and breaking are not semantic review.** Buf won't catch a field whose *meaning* changed while keeping its type; pair it with human review and documentation.
