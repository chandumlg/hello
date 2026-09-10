# Crossplane

Crossplane is an open-source Kubernetes extension that lets you provision and manage cloud infrastructure (databases, VPCs, storage buckets, managed Kubernetes clusters, and more) using the same declarative, reconciliation-based model Kubernetes uses for pods and deployments — instead of running a separate tool like Terraform out-of-band.

## Primary use cases

Crossplane is the backbone of most modern internal developer platforms (IDPs). Teams reach for it when they want to:

- **Turn cloud resources into Kubernetes APIs.** Once installed, `kubectl apply -f my-database.yaml` can create a real RDS instance, GCP Cloud SQL database, or S3 bucket — no separate CLI, state file, or pipeline step required.
- **Build self-service platforms.** Platform engineers define higher-level, opinionated "Composite Resources" (XRs) — e.g., a `SQLDatabase` that bundles networking, backups, and IAM policy — and expose them to application teams via simple, narrow YAML manifests or a Backstage template, hiding the underlying cloud complexity.
- **Enforce continuous reconciliation.** Unlike a one-shot `terraform apply`, Crossplane runs controllers that continuously watch and correct drift, the same way a Kubernetes Deployment controller keeps replica counts in sync. If someone manually changes a resource in the cloud console, Crossplane reverts it.
- **Unify infra and app deployment.** Because infrastructure is expressed as Kubernetes objects, it can live in the same GitOps repo and be rolled out by the same tools (Argo CD, Flux) that deploy application workloads — one control plane, one reconciliation loop, one audit trail.

Teams typically adopt Crossplane when they're already Kubernetes-native, want to stop treating "provision infra" and "deploy app" as separate workflows, or are building a platform team's self-service layer and need to abstract cloud primitives behind safer, simpler APIs.

## Basic usage

**1. Install Crossplane into a cluster (via Helm):**

```bash
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace
```

**2. Install a provider and give it cloud credentials:**

```bash
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-aws-s3
spec:
  package: xpkg.upbound.io/upbound/provider-aws-s3:v1
EOF
```

**3. Provision a real cloud resource declaratively:**

```yaml
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
metadata:
  name: my-app-uploads
spec:
  forProvider:
    region: us-east-1
  providerConfigRef:
    name: default
```

```bash
kubectl apply -f bucket.yaml
kubectl get bucket my-app-uploads   # shows READY: True once provisioned in AWS
```

From here, a platform team would typically wrap resources like this into a `Composition` and expose a simplified `XBucket` claim to application developers, so they never touch AWS-specific fields directly.

## Common pitfalls

- **Composition complexity creeps up fast.** Compositions (especially the older Patch-and-Transform style) can become hard to read and debug; many teams now prefer function-based Compositions (KCL, Python, or Go "composition functions") for anything non-trivial.
- **State lives in the cluster, not a state file — but the cluster becomes critical infrastructure.** Losing the management cluster without backups (etcd snapshots, or a GitOps source of truth) means losing track of everything Crossplane provisioned. Treat the control plane itself as tier-0 infrastructure.
- **Provider CRD sprawl.** Each cloud provider package installs large numbers of CRDs; upgrading providers across a big fleet of resources requires care around API version migrations (`v1beta1` → `v1`, etc.) and can require coordinated patches.
- **RBAC and blast radius.** Because any `kubectl apply` can now create real, billable cloud infrastructure, teams need to lock down who can create raw provider resources vs. only the higher-level, guardrailed Composite Resource claims — otherwise you've just made it easier to accidentally spin up expensive or insecure infrastructure.
- **Reconciliation loops can mask slow cloud APIs.** Some managed resources (e.g., large RDS instances) take many minutes to become `READY`; teams new to Crossplane sometimes mistake this normal async reconciliation for a stuck or failed apply.
- **Not a Terraform drop-in replacement for every team.** Organizations with heavy non-Kubernetes infrastructure, or that rely on Terraform's mature module ecosystem and `plan`/preview workflow, often run both side by side rather than migrating everything.
