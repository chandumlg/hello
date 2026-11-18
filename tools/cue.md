# CUE

CUE (Configure, Unify, Execute) is a configuration and validation language that merges schemas, constraints and data into one model, so you can define what valid configuration looks like, check YAML/JSON against it, and generate the final YAML/JSON from a single source of truth.

## Why teams adopt it

Most configuration at scale is YAML or JSON with no type system: Kubernetes manifests, Helm values, CI pipelines, feature flag files, service catalogs. The usual responses each have a cost. Templating (Helm, Jinja, `envsubst`) treats config as text, so mistakes surface as broken YAML or as wrong values that deploy fine. JSON Schema validates but cannot generate or fill in config. General-purpose languages (Pulumi, CDK, Jsonnet scripts) can do anything, which means reviewers must read a program to know what the output is.

CUE takes a different position: **types and values are the same thing**. `int`, `>=1 & <=65535`, `"prod" | "staging"` and `8080` are all values, ordered from less specific to more specific. Combining two values (called *unification*, written `&`) produces the most specific value that satisfies both, or an error if they conflict. There is no override and no inheritance. A value, once set, cannot be changed somewhere else, only further constrained. This property makes CUE configs safe to split across many files and teams, because the result does not depend on file order or on which layer "wins".

The core concepts:
- **Unification (`&`)**: merge constraints and data. `port: int & >=1024` and `port: 8080` unify to `8080`. `port: 8080` and `port: 9090` is a conflict error, not a silent override.
- **Disjunctions and defaults (`|`, `*`)**: `env: *"dev" | "staging" | "prod"` means one of three values, defaulting to `"dev"` if nothing more specific is given.
- **Definitions (`#Name`)**: schemas. Definitions are *closed* by default, so a field not listed in the schema (a typo like `replcas`) is an error.
- **Packages**: every `.cue` file in a directory that declares the same `package` is unified together. You can put the schema in one file, org-wide policy in another and per-service values in a third.
- **Comprehensions**: `for` and `if` clauses that generate fields, used to stamp out per-environment or per-service config from a table of inputs.
- **Concreteness**: `cue export` only succeeds if every field resolves to a concrete value (`8080`, not `int`). This is how CUE says that the config is complete.
- **Modules and registry**: `cue mod init` creates a module, and dependencies (including shared schemas) can be published to and fetched from an OCI registry, so a platform team can version its schemas like a library.

Typical adoption triggers:
- A platform team ships "golden path" configuration (Kubernetes workloads, CI templates, service definitions) and wants to enforce org policy (resource limits set, approved registries only, required labels) in CI instead of in code review.
- Helm charts have grown hundreds of lines of `{{ if }}` logic and nobody trusts a change to `values.yaml` without rendering every environment.
- Several tools consume the same facts (service name, ports, owners, SLOs), and those facts are copied into many YAML files that drift.
- You want to validate existing YAML or JSON (OpenAPI specs, GitHub Actions workflows, data fixtures) against a schema without rewriting it. CUE can do validation alone, with no other changes to the stack.
- You need typed config generation from Go structs, JSON Schema or OpenAPI. CUE can import all three.

CUE is used in Dagger's earlier releases, in KubeVela, in Grafana's schema work (Thema/Kinds) and in many internal platform tools. The reference implementation and Go API live at `cuelang.org/go`, so Go services can embed it.

## Basic usage

**1. Validate existing YAML against a schema**

You do not need to convert anything to CUE to get value from it. Write a schema and check the YAML you already have.

```cue
// schema.cue
package deploy

#Service: {
	name:     string & =~"^[a-z][a-z0-9-]{1,40}$"
	team:     string
	env:      "dev" | "staging" | "prod"
	replicas: int & >=1 & <=50
	port:     int & >=1024 & <=65535
	image:    string & =~"^registry\\.internal\\.example\\.com/"

	// Production must run at least 3 replicas
	if env == "prod" {
		replicas: >=3
	}
}
```

```yaml
# services/payments.yaml
name: payments
team: billing
env: prod
replicas: 2
port: 8443
image: docker.io/library/payments:1.4.2
```

```bash
# Install (Go toolchain, Homebrew, or a release binary)
go install cuelang.org/go/cmd/cue@latest   # or: brew install cue-lang/tap/cue

# Validate every YAML file against #Service
cue vet -c schema.cue services/*.yaml -d '#Service'
```

Output (abridged):

```
replicas: invalid value 2 (out of bound >=3)
image: invalid value "docker.io/library/payments:1.4.2" (out of bound =~"^registry\\.internal\\.example\\.com/")
```

`-d '#Service'` applies the definition to each data file. `-c` requires every field to be concrete, so a missing required field also fails. Run this in CI and you have a policy gate with no custom code.

**2. Generate config from a small amount of input**

Here CUE is the source, and the YAML is a build output.

```cue
// services.cue
package deploy

// Per-service inputs: the only part most engineers edit
_services: {
	payments: {team: "billing", port: 8443, replicas: 3}
	ledger:   {team: "billing", port: 8080}
	search:   {team: "discovery", port: 9200, replicas: 4}
}

// Org-wide defaults, expressed as constraints with defaults
#Defaults: {
	replicas: *2 | int
	env:      *"prod" | string
	...  // allow the per-service fields (team, port) through the closed definition
}

// Generate one Kubernetes Deployment per service
deployments: {
	for name, svc in _services {
		(name): {
			apiVersion: "apps/v1"
			kind:       "Deployment"
			metadata: {
				"name": name
				labels: {app: name, team: svc.team}
			}
			let cfg = #Defaults & svc
			spec: {
				replicas: cfg.replicas
				selector: matchLabels: app: name
				template: {
					metadata: labels: app: name
					spec: containers: [{
						"name": name
						image:  "registry.internal.example.com/\(name):latest"
						ports: [{containerPort: svc.port}]
					}]
				}
			}
		}
	}
}
```

```bash
# Check the whole package for errors and concreteness
cue vet -c ./...

# Render just the deployments as a YAML stream for kubectl
cue export . -e 'deployments' --out yaml

# Or one service
cue export . -e 'deployments.ledger' --out json
```

Fields beginning with `_` are hidden and never exported, so `_services` is input only. `ledger` gets `replicas: 2` from the default. Setting `replicas: 0` would still be accepted by `int`; add `& >=1` to `#Defaults` and CUE rejects it before anything reaches the cluster.

**3. Turn existing types into schemas and inject per-environment values**

```bash
# Start a module so imports and dependencies resolve
cue mod init example.com/platform

# Import Kubernetes API types from Go as CUE definitions
go mod init example.com/platform && go get k8s.io/api/apps/v1
cue get go k8s.io/api/apps/v1

# Convert an existing YAML or JSON Schema file into CUE to start from
cue import -p deploy services/payments.yaml
cue import -p deploy jsonschema: schemas/service.schema.json
```

Then constrain your generated objects with the upstream types:

```cue
import appsv1 "k8s.io/api/apps/v1"

deployments: [string]: appsv1.#Deployment
```

Now a misspelled field anywhere in a generated Deployment (`replica:` instead of `replicas:`) fails `cue vet`, because the imported definition is closed.

Values can be injected at the command line with tags, which is the usual way to choose an environment without templating:

```cue
env: *"dev" | "staging" | "prod" @tag(env)
```

```bash
cue export . -t env=prod -e deployments --out yaml
```

## Common pitfalls

- **There is no override.** Engineers coming from Helm or Kustomize expect a later layer to replace an earlier value. In CUE, `replicas: 2` in one file and `replicas: 3` in another is a conflict. Model the changeable value as a default (`replicas: *2 | int`) in the base layer, and set the concrete value exactly once. Most "CUE is fighting me" complaints come from this design choice.
- **Open structs hide typos.** A plain struct (`{...}`, or a struct not defined with `#`) accepts any extra field, so `replcas: 3` passes silently. Use definitions (`#Service`) for anything that should be checked, and use `...` only where extra fields are intended.
- **`cue eval` is not `cue export`.** `cue eval` happily prints incomplete values like `port: int`. Only `cue export` and `cue vet -c` prove that the config is concrete. Use those in CI.
- **Package-wide unification surprises.** Every `.cue` file in a directory with the same package name is merged. A stray test or scratch file in that directory changes the output. Keep packages small and deliberate, and use `cue export file.cue` when you want a single file.
- **`int` and `float` are distinct.** `1` is an `int` and `1.0` is a `float`. A field typed `int` rejects `1.0`, which is a common failure when validating JSON generated by tools that write all numbers as floats. Use `number` where either is acceptable.
- **Default resolution can be ambiguous.** Two defaults that unify into different results (`*1 | int` and `*2 | int`) leave the field without a default, and export fails as "incomplete". Keep each default in one place.
- **Performance with large disjunctions and comprehensions.** Deeply nested disjunctions over big structures (for example, unioning every Kubernetes kind) can be slow. Recent releases ship a rewritten evaluator that fixes many of these cases, so stay on a current version and run `cue vet` timing checks in CI on large configs.
- **The language is pre-1.0.** CUE is stable in practice, but minor versions have changed behavior (the module system, the evaluator and some built-ins). Pin the `cue` version in CI and in `cue.mod/module.cue` (`language: version:`), and read the release notes before upgrading.
- **The learning curve falls on the whole team.** CUE's lattice model is simple once learned, but its error messages reference unification paths that can be confusing at first. A common pattern is for the platform team to own the schemas and generators while service teams only edit small, flat input files (the `_services` block above). Most engineers never need to learn more than that.
- **Choose the integration point.** CUE can be only a validator (`cue vet` over existing YAML), a generator (`cue export` into Git or a pipeline), or embedded through the Go API. Starting with validation is low-risk and builds trust before you move any source of truth into CUE.

## Further reading

- Language tour and docs: https://cuelang.org/docs/
- CUE playground (try snippets in the browser): https://cuelang.org/play/
- Go API: https://pkg.go.dev/cuelang.org/go/cue
