# Karpenter

Karpenter is a Kubernetes node-autoscaling controller that provisions right-sized compute directly from your cloud provider in response to unschedulable pods, replacing the slower, instance-type-locked scaling of the Cluster Autoscaler.

## Primary use cases and when to adopt it

Karpenter's core job is to watch for pods that can't be scheduled (insufficient CPU/memory/GPU on existing nodes) and immediately launch new nodes that fit, rather than scaling a pre-defined, fixed-shape node group. It picks the actual instance type, zone, and capacity type (on-demand vs. spot) that best fits the pending workload at the current market price, and it also actively consolidates and deprovisions underutilized or aging nodes to control cost.

Teams adopt it when:

- They're running large or bursty Kubernetes clusters (batch jobs, ML training/inference, CI runners, event-driven services) and are tired of managing dozens of statically-sized node groups per instance type/zone combination.
- Cost is a real lever — Karpenter's bin-packing and spot-aware consolidation typically cut EC2/GKE/AKS node spend meaningfully compared to a Cluster Autoscaler tuned around fixed ASGs.
- Scale-up latency matters (e.g., spiky inference workloads, CI) — Karpenter launches instances directly via cloud APIs instead of resizing an Auto Scaling Group, which is usually faster.
- Platform engineers want a single, flexible provisioning policy ("any of these instance families, any of these zones, spot preferred") instead of maintaining many node groups by hand.

It's overkill for small, stable clusters with predictable load, where a couple of manually sized node groups (or the Cluster Autoscaler) are simpler to reason about.

## Basic usage examples

1. **Install via Helm** (AWS example):
   ```bash
   helm install karpenter oci://public.ecr.aws/karpenter/karpenter \
     --version "1.0.6" \
     --namespace kube-system \
     --set settings.clusterName=my-cluster \
     --set settings.interruptionQueue=my-cluster-karpenter
   ```

2. **Define a NodePool** — the policy describing what kinds of nodes Karpenter is allowed to create:
   ```yaml
   apiVersion: karpenter.sh/v1
   kind: NodePool
   metadata:
     name: default
   spec:
     template:
       spec:
         requirements:
           - key: kubernetes.io/arch
             operator: In
             values: ["amd64"]
           - key: karpenter.sh/capacity-type
             operator: In
             values: ["spot", "on-demand"]
         nodeClassRef:
           group: karpenter.k8s.aws
           kind: EC2NodeClass
           name: default
     limits:
       cpu: 1000
     disruption:
       consolidationPolicy: WhenEmptyOrUnderutilized
       expireAfter: 720h
   ```

3. **Trigger and observe scaling**: deploy a workload that requests more capacity than exists, then watch Karpenter provision nodes for it:
   ```bash
   kubectl apply -f my-deployment.yaml
   kubectl get nodeclaims -w
   kubectl logs -n kube-system -l app.kubernetes.io/name=karpenter -f
   ```

## Common pitfalls

- **Overly broad NodePool requirements** (e.g., allowing every instance family and no resource limits) can lead to surprising instance choices or runaway cost — always set `limits` on CPU/memory per NodePool.
- **Consolidation surprises**: `WhenEmptyOrUnderutilized` will actively terminate and reschedule pods onto cheaper nodes, which can disrupt latency-sensitive or stateful workloads if you haven't set proper PodDisruptionBudgets or `do-not-disrupt` annotations.
- **Spot interruption handling requires setup**: on AWS you need the interruption queue (SQS) wired up, or Karpenter won't get advance notice of spot reclamation and pods will be evicted abruptly.
- **NodeClass/cloud IAM misconfiguration** is a frequent first-run blocker — Karpenter needs permissions to describe and launch instances, and a missing subnet/security-group tag (`karpenter.sh/discovery`) silently prevents node creation with no obvious error beyond "no matching NodePool."
- **Migrating from Cluster Autoscaler** should be gradual — run both briefly with mutually exclusive node selectors/taints rather than a hard cutover, since misaligned NodePools can cause scheduling deadlock.
- **API version churn**: Karpenter's CRDs moved from `v1beta1` to `v1` (provisioners renamed to NodePools), so tutorials and Helm charts referencing old API groups will fail on current releases — check your installed version against the docs you're following.
