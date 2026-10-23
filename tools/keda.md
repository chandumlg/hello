# KEDA (Kubernetes Event-Driven Autoscaling)

KEDA scales Kubernetes workloads — including all the way down to zero — based on external event sources such as queue depth, stream lag, or a Prometheus query, instead of just CPU and memory.

## Why teams adopt it

The built-in Horizontal Pod Autoscaler (HPA) reacts to resource metrics. That's a poor proxy for many real workloads: a Kafka consumer sitting at 5% CPU can be a million messages behind, and an idle batch worker still costs money while waiting for jobs. KEDA is a CNCF graduated project that adds an event-driven layer on top of HPA. It ships ~60+ "scalers" (Kafka, RabbitMQ, SQS, Pulsar, Redis, Postgres, Prometheus, Cron, Azure/GCP queues, and more), handles the 0↔1 activation itself, and exposes metrics to the HPA for 1↔N scaling.

Typical use cases:
- **Queue/stream consumers** that should scale on lag or backlog rather than CPU.
- **Scale-to-zero** for bursty or rarely used workers, cutting cost on shared clusters.
- **Scheduled scaling** (Cron scaler) for predictable traffic windows.
- **Batch jobs**: `ScaledJob` spawns one Kubernetes Job per event/batch of events.
- **Scaling on business metrics** via the Prometheus scaler (e.g. requests in flight, pending orders).

Platform teams adopt it when many teams run asynchronous workers and want one declarative, consistent autoscaling API rather than hand-rolled custom-metrics adapters per workload.

## How it works

KEDA installs three pieces: an **operator** (reconciles `ScaledObject`/`ScaledJob`, manages the HPA, and does activation from/to zero), a **metrics server** (serves external metrics to the HPA), and optional **admission webhooks**. You don't replace your Deployment — you attach a `ScaledObject` to it.

## Getting started

**1. Install with Helm**

```bash
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace
kubectl get pods -n keda
```

**2. Scale a Kafka consumer on consumer-group lag**

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: orders-consumer
spec:
  scaleTargetRef:
    name: orders-consumer        # existing Deployment
  minReplicaCount: 0             # scale to zero when idle
  maxReplicaCount: 20
  cooldownPeriod: 300
  triggers:
    - type: kafka
      metadata:
        bootstrapServers: kafka:9092
        consumerGroup: orders
        topic: orders
        lagThreshold: "100"      # target ~100 msgs of lag per replica
```

```bash
kubectl apply -f scaledobject.yaml
kubectl get scaledobject orders-consumer   # READY / ACTIVE columns
kubectl get hpa                            # KEDA-managed HPA: keda-hpa-orders-consumer
```

**3. Authenticate to the event source without embedding secrets**

```yaml
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: sqs-auth
spec:
  podIdentity:
    provider: aws                # IRSA / workload identity instead of static keys
---
# reference it from a trigger:
#   triggers:
#     - type: aws-sqs-queue
#       authenticationRef: { name: sqs-auth }
#       metadata: { queueURL: ..., queueLength: "50", awsRegion: us-east-1 }
```

Use `ClusterTriggerAuthentication` to share one auth config across namespaces, and prefer pod identity or secret references over inline credentials.

## Pitfalls and things to watch

- **Don't run a ScaledObject and your own HPA on the same Deployment.** They fight; KEDA creates and owns an HPA for you. Migrate existing HPAs rather than layering.
- **Scale-to-zero needs a producer that tolerates latency.** The first event waits for pod scheduling, image pull, and startup. Cold start is real; use `minReplicaCount: 1` for latency-sensitive paths.
- **Parallelism is capped by the source.** For Kafka, replicas beyond the partition count do nothing (KEDA caps at partition count by default; `allowIdleConsumers` changes that). Size partitions before `maxReplicaCount`.
- **Polling interval and cooldown tuning.** `pollingInterval` (default 30s) bounds reaction time; `cooldownPeriod` governs only the final scale to zero, while HPA `behavior` stabilization governs 1↔N. Flapping usually means tuning those, not the threshold.
- **Metric semantics matter.** `lagThreshold`/`queueLength` is a per-replica *target* (the HPA uses AverageValue), not a trigger point for scaling. Misreading this leads to over- or under-scaling.
- **KEDA's own availability is on the critical path.** If the operator or metrics server is down, scaling stops (and activation from zero fails). Run it with multiple replicas and monitor its metrics; also be aware only one external metrics provider can own `external.metrics.k8s.io` in a cluster, which can conflict with other adapters (e.g. Prometheus Adapter).
- **Credentials and network reach.** The KEDA operator itself connects to the event source to read metrics, so it needs network access and permissions there, separate from your workload's.
- **Use `ScaledJob` for long-running, non-interruptible tasks.** Scaling a Deployment down can kill a pod mid-task; jobs run to completion.
- **Pausing for maintenance:** annotate with `autoscaling.keda.sh/paused-replicas: "N"` to pin replicas during incidents or migrations.
