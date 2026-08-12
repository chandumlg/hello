# Terraform

Terraform is an infrastructure-as-code (IaC) tool that lets you define cloud
and on-prem infrastructure in a declarative configuration language, then
plan and apply changes to reach that state reproducibly across providers.

## Primary use cases

- **Multi-cloud/provider provisioning**: a single workflow to manage AWS,
  GCP, Azure, Kubernetes, Datadog, GitHub, and hundreds of other providers
  through one consistent CLI and state model.
- **Reproducible environments**: spin up identical dev/staging/prod
  infrastructure from the same configuration, parameterized by variables,
  instead of hand-clicking consoles or writing bespoke scripts per cloud.
- **Change review as code**: `terraform plan` produces a diff of what will
  be created, changed, or destroyed *before* anything happens, so
  infrastructure changes go through the same PR review process as
  application code.
- **Drift detection and state tracking**: Terraform keeps a state file
  mapping your config to real-world resource IDs, so it can tell you when
  actual infrastructure has drifted from what's declared.

A team typically adopts Terraform once manual console changes or ad-hoc
provisioning scripts become a bottleneck — when they need audit trails for
infra changes, want to avoid "snowflake" environments that can't be
recreated, or are managing resources across more than one provider/account
and need a unified workflow.

## Basic usage

**1. Define a resource** (`main.tf`):

```hcl
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_s3_bucket" "logs" {
  bucket = "my-team-app-logs"
}
```

**2. Initialize, plan, apply:**

```bash
terraform init      # downloads providers, sets up backend/state
terraform plan       # shows what will change, without applying it
terraform apply       # applies the plan (prompts for confirmation)
```

**3. Use variables and outputs for reusable modules:**

```hcl
variable "bucket_name" {
  type = string
}

resource "aws_s3_bucket" "logs" {
  bucket = var.bucket_name
}

output "bucket_arn" {
  value = aws_s3_bucket.logs.arn
}
```

```bash
terraform apply -var="bucket_name=prod-app-logs"
```

## Common pitfalls

- **Local state files**: by default state is stored on disk (`terraform.tfstate`).
  For any team usage, configure a remote backend (S3+DynamoDB, Terraform
  Cloud, GCS, etc.) with locking — otherwise concurrent applies corrupt
  state or silently overwrite each other's changes.
- **State drift and manual changes**: if someone edits a resource outside
  Terraform (e.g., via the AWS console), the next `plan`/`apply` may try to
  "correct" it, sometimes destructively. Treat Terraform-managed resources
  as read-only outside of Terraform.
- **Secrets in state**: the state file often contains resource attributes
  in plain text, including secrets (e.g., generated DB passwords). Encrypt
  the backend at rest and restrict access to state.
- **Destructive plans hiding in noise**: a plan showing hundreds of
  "no-op" changes can bury a single unintended `- destroy` on a critical
  resource. Always read the plan summary line and diff carefully, not just
  skim for green vs. red.
- **Provider version drift**: unpinned provider versions can introduce
  breaking changes between runs. Pin versions (`~>` constraints) and commit
  the `.terraform.lock.hcl` file.
- **Large blast radius from a single `apply`**: a single misconfigured
  module can affect many resources at once. Use `-target` sparingly for
  emergencies, and prefer splitting state across smaller, well-scoped
  workspaces/modules for large infrastructures.
