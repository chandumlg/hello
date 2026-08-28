# Envoy Proxy

Envoy is an open-source, high-performance L4/L7 proxy built for cloud-native applications, and it solves the problem of making service-to-service networking (load balancing, retries, timeouts, TLS, observability) uniform and programmable instead of scattered across application code and a pile of mismatched client libraries.

## Primary use cases

Envoy sits at three common layers in a modern infrastructure stack:

- **Edge proxy / API gateway** — terminating TLS, routing HTTP(S)/gRPC traffic into a cluster, and handling rate limiting or authentication at the perimeter (this is what powers gateways like Ambassador, Contour, and Gloo).
- **Service mesh data plane** — deployed as a sidecar next to every service instance to transparently handle mutual TLS, retries, circuit breaking, and traffic shifting between services. Istio, Linkerd's alternative meshes, and AWS App Mesh all build on Envoy for exactly this reason.
- **Central/shared proxy tier** — a fleet of Envoy instances doing north-south or east-west load balancing with dynamic, API-driven configuration (via its "xDS" discovery protocol) instead of static config files and reloads.

A team typically adopts Envoy when they outgrow a simple reverse proxy (nginx/HAProxy with static config) and need dynamic service discovery, fine-grained per-route traffic policies, rich L7 observability (per-route latency/error metrics out of the box), or a common networking layer that's consistent across polyglot services — so retries, timeouts, and mTLS aren't reimplemented in every language's HTTP client.

## Basic usage examples

**1. Minimal standalone proxy (static config).** A bootstrap config that listens on 10000 and proxies to a backend cluster:

```yaml
# envoy.yaml
static_resources:
  listeners:
    - name: listener_0
      address:
        socket_address: { address: 0.0.0.0, port_value: 10000 }
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: backend
                      domains: ["*"]
                      routes:
                        - match: { prefix: "/" }
                          route: { cluster: backend_service }
                http_filters:
                  - name: envoy.filters.http.router
  clusters:
    - name: backend_service
      connect_timeout: 5s
      type: LOGICAL_DNS
      load_assignment:
        cluster_name: backend_service
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address: { address: backend.internal, port_value: 8080 }
```

```bash
docker run -d -p 10000:10000 -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml \
  envoyproxy/envoy:v1.31-latest
```

**2. Check admin/observability endpoints.** Envoy exposes a built-in admin interface with live stats and config dumps — invaluable for debugging routing issues:

```bash
curl http://localhost:9901/stats/prometheus   # Prometheus-format metrics
curl http://localhost:9901/config_dump        # currently active config
curl http://localhost:9901/clusters           # per-upstream health/state
```

**3. Add retries and timeouts to a route.** Instead of hand-rolling retry logic per client, declare it once in the route config:

```yaml
routes:
  - match: { prefix: "/api" }
    route:
      cluster: backend_service
      timeout: 2s
      retry_policy:
        retry_on: "5xx,connect-failure,refused-stream"
        num_retries: 3
        per_try_timeout: 0.5s
```

## Common pitfalls

- **Hand-writing static config doesn't scale.** Envoy is designed to be driven by its xDS APIs (CDS/EDS/LDS/RDS) from a control plane (Istio's istiod, or a custom one via go-control-plane) so config updates happen without restarts. Teams that start with static YAML files often hit a wall and need to migrate to dynamic config once they have more than a handful of routes/clusters.
- **The admin endpoint (`:9901` by default) is unauthenticated** and can expose internal topology or even let you drain listeners — never bind it to a public interface; keep it on localhost or a locked-down internal network.
- **Sidecar resource overhead adds up.** Running Envoy as a per-pod sidecar (as Istio does) roughly doubles the number of running containers and adds latency/CPU per hop; budget for it, and consider Envoy's "ambient mesh" mode or per-node proxies if per-pod sidecars become too costly at scale.
- **Retries can amplify outages.** A naive `num_retries: 3` on every route multiplies load on an already-struggling upstream during an incident. Pair retries with circuit breakers (`max_connections`/`max_pending_requests` in the cluster's `circuit_breakers`) and sane `per_try_timeout` values.
- **Cluster warm-up and passive health checks are not on by default** — without configuring active health checks or outlier detection, Envoy will happily keep sending traffic to a cluster member that's failing until it's removed from service discovery, which can be much slower than you'd like during a bad deploy.
