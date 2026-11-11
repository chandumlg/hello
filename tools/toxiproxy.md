# Toxiproxy

Toxiproxy is a programmable TCP proxy from Shopify that sits between your service and its dependencies (databases, caches, queues, upstream APIs) and injects network failures on demand, like latency, dropped connections, bandwidth limits and timeouts, so you can test how your code behaves when the network misbehaves.

## Why teams adopt it

Most resilience bugs show up only under degraded networks. Examples: a missing client timeout, a retry storm, a connection pool that never recovers, a circuit breaker that never opens. Unit tests with mocked clients don't catch these, because the mock never behaves like a half-open socket. Full chaos platforms (Chaos Mesh, Gremlin, Litmus) are built for clusters and staging environments, and they're too heavy to run inside a `go test` or `pytest` loop. Toxiproxy fills that gap. It's one small Go binary with an HTTP control API, it's deterministic, and test code can drive it directly.

You point your app at the proxy's listen port instead of the real dependency. The proxy forwards bytes untouched until a test adds a **toxic**. Toxics apply per direction: **downstream** is data coming from the dependency back to your app, and **upstream** is data your app sends to the dependency. A **toxicity** value between 0.0 and 1.0 sets the probability that a given connection gets the toxic.

Built-in toxics:
- `latency`: adds delay plus optional jitter to data.
- `bandwidth`: caps throughput in KB/s.
- `timeout`: stops all data and closes the connection after N ms. With `timeout=0`, it holds the connection open forever, which is a black hole.
- `reset_peer`: sends a TCP RST, simulating "connection reset by peer".
- `slow_close`: delays the socket close.
- `slicer`: splits data into small chunks with delays between them. Good for finding framing and partial-read bugs.
- `limit_data`: closes the connection after N bytes.
- Disabling a proxy entirely simulates a hard outage: the port refuses connections.

Common adoption triggers:
- You've had an incident where a slow dependency, not a dead one, took a service down, and you want a regression test for it.
- You need to verify timeouts, retries with backoff, circuit breakers and pool eviction in CI, not just in a design doc.
- A platform team wants a standard "degraded dependency" harness that every service's integration suite can use.
- You're reproducing a flaky production failure (Redis failover, replica lag, a cross-region RTT spike) on a laptop.

## Basic usage

**1. Run the server and proxy Redis through it with the CLI:**
```bash
# Start the server. The control API listens on :8474.
docker run -d --name toxiproxy -p 8474:8474 -p 26379:26379 \
  ghcr.io/shopify/toxiproxy

# Create a proxy. Bind 0.0.0.0 inside the container, not localhost.
toxiproxy-cli create --listen 0.0.0.0:26379 --upstream host.docker.internal:6379 redis

# Add 1s of latency (±200ms jitter) to responses coming back to the client.
toxiproxy-cli toxic add --type latency --attribute latency=1000 --attribute jitter=200 redis

redis-cli -p 26379 PING       # takes about 1s now
toxiproxy-cli inspect redis   # shows active toxics
toxiproxy-cli toggle redis    # disable the proxy to simulate a full outage
toxiproxy-cli toxic remove --toxicName latency_downstream redis
```

**2. Drive it from a Go integration test:**
```go
import toxiproxy "github.com/Shopify/toxiproxy/v2/client"

func TestCheckoutSurvivesSlowDB(t *testing.T) {
    tp := toxiproxy.NewClient("localhost:8474")
    proxy, err := tp.CreateProxy("postgres", "0.0.0.0:25432", "postgres:5432")
    require.NoError(t, err)
    t.Cleanup(func() { proxy.Delete() })

    // The app under test connects to :25432 instead of :5432.
    svc := newCheckoutService("postgres://app@localhost:25432/shop")

    // Black-hole the DB: no data, connection never closes.
    _, err = proxy.AddToxic("blackhole", "timeout", "downstream", 1.0,
        toxiproxy.Attributes{"timeout": 0})
    require.NoError(t, err)

    start := time.Now()
    _, err = svc.PlaceOrder(ctx, order)
    require.ErrorIs(t, err, ErrDependencyUnavailable)
    require.Less(t, time.Since(start), 3*time.Second) // our deadline is enforced

    proxy.RemoveToxic("blackhole")
    _, err = svc.PlaceOrder(ctx, order) // the pool must recover on its own
    require.NoError(t, err)
}
```
Python (`toxiproxy-python`), Ruby, Java, .NET and Node clients exist too. Testcontainers ships a Toxiproxy module that wires the container network for you.

**3. Use the raw HTTP API and declarative config (works from any language or CI step):**
```bash
# toxiproxy.json: load at boot with `toxiproxy-server -config toxiproxy.json`
# [
#   {"name": "kafka",  "listen": "0.0.0.0:29092", "upstream": "kafka:9092"},
#   {"name": "orders", "listen": "0.0.0.0:28080", "upstream": "orders-api:8080"}
# ]

# Make 30% of new connections to orders-api get reset by the peer.
curl -s -X POST localhost:8474/proxies/orders/toxics -d '{
  "name": "flaky", "type": "reset_peer", "stream": "downstream",
  "toxicity": 0.3, "attributes": {"timeout": 0}
}'

# Throttle the Kafka link to 64 KB/s to test producer backpressure.
curl -s -X POST localhost:8474/proxies/kafka/toxics -d '{
  "type": "bandwidth", "stream": "upstream", "attributes": {"rate": 64}
}'

# Between tests: remove all toxics and re-enable all proxies.
curl -s -X POST localhost:8474/reset
```

## Common pitfalls

- **It works at layer 4 only.** Toxiproxy sees TCP bytes, not requests. It can't return an HTTP 503, delay one gRPC method, or corrupt a specific SQL response, and it doesn't support UDP (so no DNS or QUIC). For L7 faults, use Envoy/Istio fault injection or a mock server such as WireMock. TLS passes through as opaque bytes, which is convenient but means you can't inspect or alter it.
- **Toxics apply per chunk and per connection, not per request.** `latency` delays each chunk of data, so a large response streamed in many chunks can be delayed far more than you expect. `toxicity` is decided once per connection, so with pooled, long-lived connections, "30% flaky" can mean that every request fails or none do. Size pools and keep-alive settings in tests on purpose.
- **Toxics leak between tests.** State lives in the server process, so a test that fails before cleanup leaves the latency in place for every later test. Always `POST /reset` (or remove the proxy) in teardown or `t.Cleanup`, and avoid sharing one proxy across parallel tests.
- **Listen addresses and ports inside containers.** A proxy listening on `127.0.0.1` inside a container is unreachable from the host or other containers. Use `0.0.0.0`, publish every listen port (not just 8474), and make sure the upstream hostname resolves from inside the Toxiproxy container's network.
- **Clients bypass the proxy.** Redis Cluster/Sentinel, MongoDB replica sets, Kafka and Cassandra hand out the real node addresses during discovery, so after the first call your client connects directly and skips the proxy. Configure advertised listeners or announce addresses to point at proxy ports, or proxy every node.
- **Configuration doesn't persist.** Proxies created through the API vanish on restart. Use `-config` or `POST /populate` so CI environments start the same way every time.
- **The control API has no authentication.** Anyone who can reach :8474 can black-hole your traffic. Keep it on test networks only. Running it in production paths "just for experiments" is how a test tool turns into an outage.
- **Test the recovery, not just the failure.** The bugs that matter most usually appear *after* the toxic is removed: pools full of dead sockets, circuit breakers stuck open, consumers that never rejoin. Assert that the service returns to healthy within a bounded time.
