# Infracost

**Infracost reads your infrastructure-as-code (Terraform, OpenTofu, Terragrunt) and tells you what it will cost every month, and what a given change will add to or remove from that bill, before you apply it.**

## Primary use cases and when to adopt it

- **Cost diffs in pull requests:** This is the flagship workflow. CI runs Infracost on the base branch and the PR branch and posts a comment like "+$1,240/month (+18%)" that breaks the change down by resource. The reviewer sees that `db.r6g.4xlarge` is a $1,900/month line item while the PR is still open, not when the invoice arrives.
- **Shifting FinOps left:** Most cost tooling (AWS Cost Explorer, Kubecost, CloudHealth) tells you what you *already* spent. Infracost looks at intent: what you are *about* to spend. That moves the conversation from the finance team after the fact to the engineer at review time.
- **Estimating new architectures:** You can price a design without deploying it. Write the Terraform for "three NAT gateways vs. one", "gp3 vs. io2", or "m7g vs. c7i", then run `infracost breakdown` to compare options during design review.
- **Guardrails and policies:** Fail or flag a PR when the monthly delta passes a threshold. You can also enforce rules such as "no previous-generation instance types" or "all resources must carry a `cost-center` tag". Infracost Cloud (the SaaS layer) has built-in FinOps and tagging policies. With the open-source CLI alone, you write the checks yourself against the JSON output, using OPA/Conftest or a script.
- **Platform-team visibility:** Infracost Cloud aggregates estimates across repos into a dashboard of which teams and projects drive cost changes. It can also apply your negotiated discounts and custom price books, which the CLI cannot.

**Adopt it when** your infrastructure is mostly defined in Terraform or OpenTofu, many engineers can open infra PRs, and surprise bills or "who launched this?" investigations keep happening. It is especially useful for platform teams running self-service modules, since one innocuous-looking variable change (`instance_count = 30`) can be the most expensive line in the PR. It supports AWS, Azure and Google Cloud, with over a thousand priced resource types. The CLI is open source (Apache 2.0) and free. It queries Infracost's hosted Cloud Pricing API, which requires a free API key, and you can self-host that pricing API if you can't send anything out.

## Getting started

### 1. Install, authenticate, and price a Terraform directory

```bash
# macOS (or: curl -fsSL https://raw.githubusercontent.com/infracost/infracost/master/scripts/install.sh | sh)
brew install infracost

# Get a free API key for the Cloud Pricing API (opens a browser)
infracost auth login
# Headless/CI alternative: export INFRACOST_API_KEY=ico-...

# Price everything Infracost can find under a Terraform directory
cd infra/prod
infracost breakdown --path .
```

Example output (abridged):

```
 Name                                          Monthly Qty  Unit        Monthly Cost
 aws_instance.api
 ├─ Instance usage (Linux/UNIX, on-demand, m7i.xlarge)  730  hours        $147.17
 └─ root_block_device
    └─ Storage (general purpose SSD, gp3)            100  GB             $8.00
 aws_db_instance.main
 ├─ Database instance (on-demand, Multi-AZ, db.r6g.2xlarge) 730 hours  $1,516.21
 └─ Storage (general purpose SSD, gp3)               500  GB           $115.00
 aws_nat_gateway.egress
 ├─ NAT gateway                                      730  hours         $32.85
 └─ Data processed                     Monthly cost depends on usage: $0.045 per GB

 OVERALL TOTAL                                                        $1,819.23
```

By default Infracost parses the HCL directly, so it doesn't need cloud credentials or a `terraform init`. If you want maximum accuracy (fully resolved variables, `count`/`for_each`, data sources), feed it a plan instead:

```bash
terraform init && terraform plan -out tfplan.binary
terraform show -json tfplan.binary > plan.json
infracost breakdown --path plan.json
```

### 2. Get a cost *diff* between two states of the code

```bash
# On main: snapshot the baseline
git checkout main
infracost breakdown --path . --format json --out-file /tmp/infracost-base.json

# On your feature branch: compare against it
git checkout feature/bigger-db
infracost diff --path . --compare-to /tmp/infracost-base.json
```

```
~ aws_db_instance.main
  +$758 ($1,631 → $2,389)
    ~ Database instance (on-demand, Multi-AZ, db.r6g.2xlarge → db.r6g.4xlarge)
      +$758 ($1,516 → $2,274)

Monthly cost change for infra/prod
Amount:  +$758 ($1,819 → $2,577)
Percent: +42%
```

For monorepos with many stacks, add an `infracost.yml` config file at the repo root that lists each project (`path`, `terraform_var_files`, `name`). Then use `--config-file infracost.yml` instead of `--path`, and Infracost prices every project in one run.

### 3. Post cost comments on GitHub pull requests

```yaml
# .github/workflows/infracost.yml
name: infracost
on: [pull_request]

permissions:
  contents: read
  pull-requests: write

jobs:
  infracost:
    runs-on: ubuntu-latest
    steps:
      - uses: infracost/actions/setup@v3
        with:
          api-key: ${{ secrets.INFRACOST_API_KEY }}

      - name: Baseline from the base branch
        uses: actions/checkout@v4
        with:
          ref: ${{ github.event.pull_request.base.ref }}
      - run: |
          infracost breakdown --path=infra/ \
            --format=json --out-file=/tmp/infracost-base.json

      - name: Diff from the PR branch
        uses: actions/checkout@v4
      - run: |
          infracost diff --path=infra/ \
            --compare-to=/tmp/infracost-base.json \
            --format=json --out-file=/tmp/infracost.json

      - name: Comment on the PR
        run: |
          infracost comment github --path=/tmp/infracost.json \
            --repo=$GITHUB_REPOSITORY \
            --github-token=${{ github.token }} \
            --pull-request=${{ github.event.pull_request.number }} \
            --behavior=update
```

`--behavior=update` edits a single comment in place on each push instead of adding a new one every time. Other options are `new`, `hide-and-new` and `delete-and-new`. There are equivalent `infracost comment` subcommands for GitLab, Azure Repos and Bitbucket. Infracost also ships a GitHub App and a GitLab app, which do all of the above without any workflow YAML if you'd rather not maintain it.

To gate merges on cost, parse `/tmp/infracost.json` (for example `jq '.diffTotalMonthlyCost | tonumber'`) and exit non-zero above your threshold. You can also run Conftest/OPA policies against the same JSON.

## Common pitfalls and things to watch out for

- **Usage-based costs show up as $0 unless you say otherwise.** Lambda invocations, S3 requests and storage, NAT gateway data processing, CloudWatch log ingestion and data transfer often make up a large share of a real bill. Infracost can't infer them from code. Generate a usage file with `infracost breakdown --path . --sync-usage-file --usage-file infracost-usage.yml`, fill in realistic monthly values (e.g. `monthly_data_processed_gb: 5000`), and pass `--usage-file` in CI. If you skip this, serverless-heavy stacks will look deceptively cheap. Infracost Cloud can pull actual usage from your cloud account to fill this in automatically.
- **These are list prices, not your bill.** The open-source CLI prices everything on-demand at public rates. It does not know about Savings Plans, Reserved Instances, committed-use discounts, EDP or negotiated enterprise discounts, credits, or tax. Treat the numbers as a consistent relative signal for comparing changes, not a forecast to reconcile against the invoice. Custom price books (applying your discounts) are an Infracost Cloud feature.
- **The scope is IaC only.** Anything not declared in Terraform is invisible: click-ops resources, autoscaling that happens at runtime, and workloads scheduled onto a Kubernetes cluster (Infracost prices the node groups, not the pods). Pair it with a runtime cost tool (Kubecost/OpenCost, the cloud provider's cost explorer) for the actual spend.
- **HCL parsing has limits.** Without a plan, Infracost evaluates the HCL itself. Values that come from remote state, `data` sources, or provider-computed attributes may not resolve, and Infracost falls back to defaults or skips those values. If counts or instance sizes look wrong, run from a `plan.json`, or pass the right `--terraform-var-file` and `--terraform-var` values. Make sure CI uses the same tfvars as your real deploy (e.g. `prod.tfvars`, not the defaults).
- **Private modules need credentials.** For modules from a private Git repo or Terraform Cloud/Enterprise registry, give CI access via `GIT_SSH_KEY` or `INFRACOST_TERRAFORM_CLOUD_TOKEN` and related env vars. Otherwise those modules are silently skipped and their resources go unpriced. Look for "unsupported" or "skipped" warnings in the output, and in the comment's footer.
- **Check what is unsupported.** The output ends with a count of resources that are free (IAM, security groups) and resources that Infracost doesn't support yet. A new or niche resource type may not be priced at all. Don't read "no cost change" as "no cost" without checking that list.
- **Data egress and privacy.** The CLI sends resource types and price-relevant attributes (region, instance type, storage size) to the pricing API to look up prices. According to Infracost it does not send secrets, Terraform state, or full source code. Have your security team review it anyway. If no outbound calls are allowed, self-host the Cloud Pricing API (it's open source and syncs a price database you host).
- **Comment fatigue.** If every PR in a monorepo gets a cost comment, even $0.00 ones, people stop reading them. Scope the workflow with `paths:` filters on infra directories, use `--behavior=update`, and consider only commenting or alerting above a minimum delta.
- **Pin versions.** Output format and pricing logic change between releases. Pin the CLI version in CI (the `version` input on `infracost/actions/setup`) so cost diffs don't shift because the tool changed rather than your code.

**Bottom line:** Infracost won't replace your cost explorer. Its value is that the cost of an infrastructure change shows up in code review, next to the diff that causes it, while it is still cheap to change course. Set it up in CI, add a usage file for your usage-heavy services, and treat the numbers as a directional signal rather than an invoice.
