# Trivy

Trivy is an open-source, all-in-one scanner from Aqua Security that finds known vulnerabilities, misconfigurations, exposed secrets, and license issues across container images, filesystems, Git repositories, and infrastructure-as-code — in one binary, without needing a running daemon or a backend service.

## Why teams adopt it

Platform and security engineers reach for Trivy when they need a fast "is this thing safe to ship?" check that plugs into CI without standing up heavyweight scanning infrastructure. Where Semgrep looks at your own source code for bug patterns, and Vault manages secrets at runtime, Trivy answers a different question: does this artifact — a Docker image, a Terraform module, a `package.json`, a Kubernetes manifest — contain known-vulnerable dependencies (CVEs), dangerous default configurations, or hardcoded credentials that were accidentally committed?

Common adoption triggers:
- Needing to block a container image from being pushed to a registry or deployed to production if it bundles OS packages or language dependencies with critical CVEs.
- Scanning Terraform/CloudFormation/Kubernetes YAML for misconfigurations (e.g., a public S3 bucket, a security group open to `0.0.0.0/0`, a container running as root) before `apply` or `kubectl apply`.
- Wanting a single tool to replace several point solutions — image scanning, IaC linting for security, secret detection, and SBOM (software bill of materials) generation — with one consistent CLI and output format.
- Enforcing supply-chain policy in CI: fail the build if any dependency has a fix available for a HIGH/CRITICAL vulnerability, so stale base images and outdated libraries don't silently accumulate.
- Needing SBOMs for compliance (e.g., generating a CycloneDX/SPDX document per release) without a separate SBOM tool.

## Basic usage

**1. Scan a container image for vulnerabilities:**
```bash
# Install (brew, apt/yum repo, or run via Docker)
brew install trivy

# Scan a built image, only show HIGH/CRITICAL findings
trivy image --severity HIGH,CRITICAL myregistry.io/myapp:1.4.0
```

**2. Scan Infrastructure-as-Code for misconfigurations:**
```bash
# Scan a directory of Terraform/Kubernetes/Dockerfile configs
trivy config ./infra

# Example finding: "Security group rule allows ingress from 0.0.0.0/0"
```

**3. Fail CI on fixable critical vulnerabilities and emit an SBOM:**
```bash
# Exit non-zero only when a CRITICAL vuln has a known fix
trivy image --severity CRITICAL --ignore-unfixed --exit-code 1 myapp:1.4.0

# Generate an SBOM alongside the scan
trivy image --format cyclonedx --output sbom.json myapp:1.4.0
```

## Pitfalls to watch out for

- **Vulnerability databases need connectivity (and caching) to stay current.** Trivy pulls its vuln DB and Java/npm advisory data from the network on first run; in air-gapped or heavily rate-limited CI environments, you need to pre-cache the DB or run a local Trivy server, or scans will fail or use a stale database.
- **Noise from unfixed or low-severity CVEs.** A default scan on a full OS base image can return hundreds of findings, many with no available fix; use `--ignore-unfixed` and severity filters, and maintain a `.trivyignore` for accepted-risk CVEs, or teams stop reading the output.
- **Base image choice dominates the vulnerability count.** Scanning is only as good as what it's scanning — switching from a full Debian/Ubuntu base to a slim or distroless image often removes more findings than any amount of patching, so treat scan results as a signal to reconsider the base image, not just to bump package versions.
- **IaC misconfiguration rules can be opinionated.** `trivy config` ships with a broad default policy set (via its bundled Rego checks); some findings will be intentional design choices for your environment — use inline ignores (`#trivy:ignore:<check-id>`) deliberately rather than suppressing whole categories.
- **It's a point-in-time scan, not runtime protection.** A clean scan at build time doesn't mean the running container stays clean — new CVEs are published constantly, so re-scan on a schedule (not just at build) to catch vulnerabilities disclosed after an image shipped.
