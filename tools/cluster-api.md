# Cluster API

Cluster API (CAPI) is a Kubernetes SIG subproject that lets you create, upgrade, and delete entire Kubernetes clusters by declaring them as Kubernetes resources, so cluster lifecycle management becomes the same reconcile-loop, GitOps-friendly workflow you already use for applications.

## Why teams adopt it

Most organizations that outgrow a single cluster end up with a pile of bespoke Terraform modules, shell scripts, and vendor consoles (EKS here, kubeadm there, something hand-rolled on-prem). Upgrades differ per environment, and nobody is sure what "cluster v1.31 with our standard add-ons" actually means. Cluster API gives you one declarative, provider-agnostic API: a *management cluster* runs controllers that watch `Cluster`, `Machine`, `MachineDeployment`, and control-plane objects, and use *infrastructure providers* (AWS, Azure, GCP, vSphere, Metal3, Docker, and others) to make reality match the spec.

Teams typically adopt it when:
- They operate tens to thousands of clusters (per-team, per-region, per-customer) and need a consistent, self-service way to create them.
- They run on-prem or hybrid and want cloud-like machine provisioning, rolling node replacement, and automatic remediation of unhealthy nodes.
- They want cluster upgrades to be a one-line version bump that is rolled out safely (control plane first, then workers via surge/unavailable settings), instead of a runbook.
- They practice GitOps (Argo CD / Flux) and want cluster definitions reviewed and versioned like any other manifest.
- They're building an internal platform or managed-Kubernetes offering and need a stable API to build on top of (it is what several commercial Kubernetes platforms use under the hood).

Core concepts worth knowing up front:
- **Management cluster vs. workload clusters** — the former hosts CAPI controllers; the latter are the clusters it creates and manages.
- **Providers** — infrastructure (e.g. CAPA for AWS, CAPZ for Azure, CAPV for vSphere, CAPD for Docker), bootstrap (usually kubeadm), and control plane (e.g. KubeadmControlPlane).
- **MachineDeployment / MachineSet / Machine** — the Deployment/ReplicaSet/Pod analogy, but for nodes.
- **ClusterClass + topology** — a reusable cluster template; individual clusters then just reference a class and set variables and a Kubernetes version.
- **MachineHealthCheck** — declaratively replaces nodes that fail health conditions.

## Basic usage

**1. Bootstrap a local management cluster and initialize providers.** Using `kind` and the Docker provider is the quickest way to experiment (don't use CAPD in production):
```bash
# Install clusterctl (macOS example; binaries are on the GitHub releases page)
brew install clusterctl

# Create a throwaway management cluster
kind create cluster --name capi-mgmt

# Turn it into a CAPI management cluster with the Docker infrastructure provider
export CLUSTER_TOPOLOGY=true   # enables ClusterClass support in the Docker quickstart
clusterctl init --infrastructure docker

kubectl get pods -A | grep -E 'capi|capd'
```
For real clouds, swap in `--infrastructure aws` (after exporting credentials such as `AWS_B64ENCODED_CREDENTIALS`) or the provider of your choice.

**2. Generate and apply a workload cluster.**
```bash
clusterctl generate cluster dev-1 \
  --flavor development \
  --kubernetes-version v1.31.0 \
  --control-plane-machine-count 1 \
  --worker-machine-count 2 > dev-1.yaml

kubectl apply -f dev-1.yaml

# Watch it come up
clusterctl describe cluster dev-1
kubectl get machines

# Fetch a kubeconfig for the new cluster
clusterctl get kubeconfig dev-1 > dev-1.kubeconfig
kubectl --kubeconfig dev-1.kubeconfig get nodes
```
Note that the nodes will stay `NotReady` until you install a CNI (e.g. Calico or Cilium) into the workload cluster. CAPI deliberately doesn't pick one for you (or use `ClusterResourceSet` / Cluster API Add-on Provider for Helm to automate it).

**3. Scale, upgrade, and self-heal declaratively.**
```bash
# Scale workers
kubectl scale machinedeployment dev-1-md-0 --replicas=5

# Upgrade Kubernetes: with a topology-managed (ClusterClass) cluster, bump one field
kubectl patch cluster dev-1 --type merge \
  -p '{"spec":{"topology":{"version":"v1.31.2"}}}'
```
```yaml
# Auto-remediate nodes that stay NotReady for 5 minutes
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineHealthCheck
metadata:
  name: dev-1-md-0-unhealthy
spec:
  clusterName: dev-1
  maxUnhealthy: 40%
  selector:
    matchLabels:
      cluster.x-k8s.io/deployment-name: dev-1-md-0
  unhealthyConditions:
    - type: Ready
      status: Unknown
      timeout: 300s
    - type: Ready
      status: "False"
      timeout: 300s
```
(Check the API version against your installed CAPI release; the `v1beta1` API has been the long-standing one, but newer releases are introducing newer versions.)

## Pitfalls and things to watch out for

- **The management cluster is a critical, stateful dependency.** If it's lost without backups, you lose the desired-state records for every workload cluster (the workloads keep running, but you can't manage them). Run it highly available, back up its etcd/resources, and use `clusterctl move` to pivot from a bootstrap cluster to a permanent one.
- **Never run the management cluster as a workload-sharing cluster.** Keep it small, locked down, and separate; it holds cloud credentials with power to create and destroy infrastructure.
- **Provider versions and contract compatibility matter.** Core CAPI, bootstrap, control-plane, and infrastructure providers must be compatible with each other. Upgrade with `clusterctl upgrade plan` / `apply` and read release notes first, especially across minor versions.
- **Deleting a `Cluster` deletes real infrastructure.** Deletion cascades to machines, load balancers, and volumes. Protect against accidental deletes with RBAC, admission policies (Kyverno/OPA), and GitOps prune settings. Stuck finalizers usually mean cloud resources the provider can't remove (leftover ENIs, security group dependencies).
- **Add-ons are your problem.** CNI, CSI, cloud-controller-manager, and CoreDNS config aren't solved by core CAPI; plan for ClusterResourceSets, the Helm add-on provider, or Argo CD/Flux bootstrapping per cluster.
- **Managed-service variants differ.** For EKS/AKS/GKE, provider support (e.g. `AWSManagedControlPlane`) and its feature coverage vary by provider; verify the feature you need exists before committing.
- **Learning curve.** The object graph (Cluster → InfrastructureCluster, ControlPlane, MachineDeployments → Machines → InfraMachines + Bootstrap configs) is deep. Start with `clusterctl describe cluster --show-conditions all` when debugging, and adopt ClusterClass early rather than hand-maintaining large YAML per cluster.
- **Don't use it for a single cluster.** If you only run one or two clusters, EKS/GKE plus Terraform is simpler; CAPI pays off with scale and consistency requirements.
