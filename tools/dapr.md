# Dapr

Dapr (Distributed Application Runtime) is a CNCF sidecar runtime that gives microservices portable building blocks — service invocation, pub/sub, state, secrets, workflows, bindings — behind a uniform HTTP/gRPC API, so application code doesn't get hard-wired to a specific broker, database, or cloud SDK.

## Primary use cases

- **Portable pub/sub and state**: your service publishes to `orders` and saves state under a key; whether that lands on Kafka, RabbitMQ, or Azure Service Bus, and Redis, Postgres, or DynamoDB, is a YAML "component" file, not a code change. Local dev can use Redis while production uses a managed cloud service.
- **Service-to-service calls with built-ins**: service invocation gives name-based discovery, mTLS between sidecars, retries, timeouts, and circuit breakers via resiliency policies — without a full service mesh or per-language client libraries.
- **Polyglot platforms**: when teams write Go, Python, Java, and .NET, Dapr gives them the same distributed-systems primitives through one API (SDKs are optional thin wrappers).
- **Workflows and actors**: durable workflow orchestration (code-defined, replay-based) and virtual actors for stateful, single-threaded-per-key entities.
- **Cross-cutting concerns**: secrets retrieval, distributed locks, configuration, and tracing/metrics (OpenTelemetry) are provided uniformly by the sidecar.

A team typically adopts Dapr when a platform group wants to standardize how dozens of services talk to infrastructure and avoid vendor lock-in, or when multi-cloud/on-prem portability is a requirement. It is a weaker fit for a small number of services on a single stack, where a plain client library is simpler.

## Basic usage

**1. Install the CLI and run a service with a sidecar locally**

```bash
brew install dapr/tap/dapr-cli    # or the install script from docs.dapr.io
dapr init                         # pulls placement, scheduler, Redis, Zipkin containers

dapr run --app-id orders --app-port 8000 --dapr-http-port 3500 -- python app.py
```

**2. Use the building blocks through the sidecar's HTTP API**

```bash
# Save and read state (default component name: "statestore")
curl -X POST localhost:3500/v1.0/state/statestore \
  -H 'Content-Type: application/json' \
  -d '[{"key":"order-1","value":{"sku":"abc","qty":2}}]'
curl localhost:3500/v1.0/state/statestore/order-1

# Call another service by app-id, not by host/port
curl localhost:3500/v1.0/invoke/inventory/method/stock/abc

# Publish an event (component "pubsub", topic "orders")
curl -X POST localhost:3500/v1.0/publish/pubsub/orders \
  -H 'Content-Type: application/json' -d '{"id":"order-1"}'
```

**3. Swap the backing infrastructure with a component file only**

```yaml
# components/pubsub.yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: pubsub
spec:
  type: pubsub.kafka          # was pubsub.redis
  version: v1
  metadata:
    - name: brokers
      value: "kafka:9092"
    - name: authType
      value: "none"
```

On Kubernetes, install with `dapr init -k` (or Helm) and annotate pods with `dapr.io/enabled: "true"` and `dapr.io/app-id: orders`; the injector adds the sidecar.

## Common pitfalls

- **Lowest-common-denominator abstraction**: the API exposes shared features; broker- or database-specific capabilities (Kafka partitions/consumer-group tuning, SQL queries, Redis data structures) are limited or reachable only through component metadata. Check the component's support level (stable/beta/alpha) first.
- **Operational overhead**: a sidecar per pod adds memory/CPU and an extra network hop, plus a control plane (operator, placement, sentry, scheduler) to run and upgrade. Measure latency for hot paths.
- **Pub/sub semantics are at-least-once**: handlers must be idempotent; configure dead-letter topics and retry resiliency explicitly.
- **State consistency defaults**: concurrency and consistency are per-request options, and "transactions" or ETags only work on stores that support them. Don't assume uniform guarantees across state stores.
- **Overlap with a mesh**: running Dapr mTLS and retries alongside Istio/Linkerd can double up retries and confuse tracing; decide which layer owns each concern.
- **Component scoping and secrets**: components are visible to all apps in a namespace unless `scopes` are set, and credentials should come from a secret store reference, not inline YAML.
- **Version skew**: keep the control plane and sidecar versions aligned, and upgrade the control plane before the runtimes.
