# Pulumi

Pulumi is an infrastructure-as-code (IaC) tool that lets you define, deploy, and manage cloud infrastructure using general-purpose programming languages (TypeScript, Python, Go, C#, Java, YAML) instead of a domain-specific configuration language like HCL.

## What problem it solves

Most IaC tools (Terraform being the dominant example) require learning a bespoke declarative language, which limits you to whatever loops, conditionals, and abstractions that language's authors decided to support. Pulumi solves this by treating infrastructure as ordinary code: you get real loops, functions, classes, package managers, IDE autocomplete, static typing, and unit-testing frameworks — while still getting the declarative benefits of a dependency graph, diffing, and state management that IaC tools are known for. Under the hood, Pulumi computes a resource graph and desired-state diff much like Terraform, but the "planning" logic is expressed in code you already know how to write.

## Primary use cases and when a team adopts it

- **Teams that want infra logic to look like application code** — reusing existing engineering practices (code review, unit tests, shared libraries, CI pipelines) for infrastructure instead of maintaining a parallel HCL toolchain.
- **Platform engineering teams building internal infrastructure abstractions** — e.g., a "ComponentResource" that wraps a VPC + EKS cluster + IAM roles into a single reusable class that product teams instantiate with a few parameters, published as an internal package via npm/PyPI.
- **Polyglot organizations** — when infra and application code are maintained by the same engineers who already know TypeScript/Python/Go, avoiding a context switch to HCL.
- **Dynamic/complex provisioning logic** — cases where you need real control flow (conditionally provision resources based on an API response, generate hundreds of near-identical resources from a data structure, etc.) that would be awkward in Terraform's `for_each`/`count` model.
- Teams already invested in Terraform providers don't lose that ecosystem — Pulumi can consume Terraform providers directly, so migration is incremental rather than all-or-nothing.

It's typically adopted alongside or instead of Terraform/CloudFormation/CDK, usually driven by a platform team standardizing how the rest of the org provisions cloud resources.

## Basic usage examples

**1. Install the CLI and create a new project**

```bash
curl -fsSL https://get.pulumi.com | sh
pulumi new aws-typescript   # scaffolds a new stack, prompts for project/stack name
```

**2. Define infrastructure as code** (`index.ts`)

```typescript
import * as aws from "@pulumi/aws";

const bucket = new aws.s3.Bucket("my-bucket", {
    versioning: { enabled: true },
});

export const bucketName = bucket.id;
```

**3. Preview and deploy**

```bash
pulumi preview   # shows the diff, like `terraform plan`
pulumi up        # applies the changes, like `terraform apply`
pulumi destroy   # tears down the stack's resources
```

State is stored in the Pulumi Cloud backend by default (or self-hosted in S3/Azure Blob/GCS/local file with `pulumi login --local`), and each environment (dev/staging/prod) is a separate "stack" managed with `pulumi stack init <name>` / `pulumi stack select <name>`.

## Common pitfalls

- **Secrets handling**: Pulumi encrypts config values marked with `pulumi config set --secret`, but it's easy to accidentally pass a plaintext secret as a resource property that ends up in the state file. Always use `pulumi.secret()` to wrap sensitive outputs.
- **"Output" values are not plain values**: Resource properties (like `bucket.id`) are `Output<T>`, a promise-like wrapper resolved only after deployment — you can't use them directly in plain JS logic (e.g., string concatenation or `if` checks) without `.apply()`/`pulumi.interpolate`. New users often get confused seeing `Calling [toString] on an [Output<T>]` warnings.
- **State backend choice matters at scale**: the default Pulumi Cloud backend is convenient but adds a vendor dependency; self-hosting state (S3, Azure Blob) is straightforward but you lose the built-in policy-as-code and RBAC features unless you pair it with something else.
- **Language runtime overhead**: because your "plan" phase actually executes your program (Node/Python/Go runtime), large stacks with thousands of resources can be slower to preview/diff than an equivalent Terraform plan, and runtime errors (a null reference, an unhandled exception) can crash a deployment in ways a purely declarative language wouldn't allow.
- **Provider version drift**: Pulumi's AWS/Azure/GCP providers are generated from the corresponding Terraform providers; a lag between a new cloud feature shipping and it appearing in Pulumi's provider is common, though `pulumi-aws-native` (based on CloudFormation resource schemas) reduces this for AWS.
- **Component resource sprawl**: teams that build lots of internal reusable components without careful versioning and testing can end up with the same "shared module hell" problem Terraform module consumers face — a breaking internal change quietly propagates to every stack using it.
