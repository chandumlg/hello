# Grafana Pyroscope

Grafana Pyroscope is an open-source continuous profiling platform that continuously collects low-overhead CPU, memory, and lock profiles from running services and stores them for querying, so engineers can see *which lines of code* are responsible for a cost, latency, or memory problem instead of inferring it from metrics and traces.

## Primary use cases

- **Finding the hot path behind a CPU bill**: metrics say a service uses 40 cores; a flame graph says 30% of that is JSON serialization in one handler. Continuous profiling turns "it's expensive" into a specific function to fix.
- **Diagnosing memory leaks and allocation pressure**: compare heap/alloc profiles across time ranges or deploys to see what is growing, without having to reproduce the leak locally or catch the process with a manual `pprof` snapshot.
- **Performance regression detection**: diff profiles between two versions (via labels like `version=v1.4.2`) to see exactly which functions got slower or more allocation-heavy after a release.
- **"It happened an hour ago" debugging**: ad-hoc profiling only helps while the problem is occurring. Because profiles are always being collected, you can go back to the incident window and inspect it after the fact.
- **Correlating with traces**: with span profiles (Go and some other SDKs), jump from a slow trace in Tempo to the flame graph for just that span, closing the gap between tracing ("this span was slow") and profiling ("because of this function").

A team typically adopts Pyroscope when it already has metrics, logs, and traces, and still keeps hitting questions those can't answer ("why is this function slow?"), or when infrastructure cost reduction becomes a priority and engineers need code-level targets. It slots naturally into a Grafana stack, but runs standalone too.

## Basic usage

**1. Run a server locally and push profiles from a Go app**

```bash
docker run -it -p 4040:4040 grafana/pyroscope
# UI at http://localhost:4040
```

```go
import "github.com/grafana/pyroscope-go"

func main() {
    pyroscope.Start(pyroscope.Config{
        ApplicationName: "checkout.api",
        ServerAddress:   "http://localhost:4040",
        Tags:            map[string]string{"region": "us-east-1", "version": "v1.4.2"},
        ProfileTypes: []pyroscope.ProfileType{
            pyroscope.ProfileCPU,
            pyroscope.ProfileAllocObjects,
            pyroscope.ProfileInuseSpace,
        },
    })
    // ... run the app
}
```

SDKs exist for Go, Java, Python, Ruby, Node.js, .NET, and Rust (the Java SDK uses async-profiler under the hood). This is the "push" model.

**2. Zero-code profiling with Grafana Alloy (eBPF or pull)**

```alloy
pyroscope.write "default" {
  endpoint { url = "http://pyroscope:4040" }
}

// eBPF: profile every process on the node, no code changes
pyroscope.ebpf "node" {
  forward_to = [pyroscope.write.default.receiver]
  targets    = discovery.kubernetes.pods.targets
}

// Pull: scrape Go /debug/pprof endpoints like Prometheus scrapes metrics
pyroscope.scrape "go_services" {
  targets    = discovery.kubernetes.pods.targets
  forward_to = [pyroscope.write.default.receiver]
}
```

eBPF gets you fleet-wide CPU profiles for compiled languages without touching application code; pull mode reuses Go's built-in `pprof`.

**3. Querying: filter by label and compare**

In the UI (or Grafana's Profiles Drilldown), pick `process_cpu`, filter with a label selector such as `{service_name="checkout.api", version="v1.4.2"}`, and use **Comparison view** to diff a baseline range against a regression range. From the CLI you can run the same query against the HTTP API:

```bash
profilecli query merge \
  --query='process_cpu:cpu:nanoseconds:cpu:nanoseconds{service_name="checkout.api"}' \
  --from="now-1h" --to="now"
```

## Pitfalls and things to watch out for

- **Symbols matter**: eBPF profiling of stripped binaries, or JITed/interpreted runtimes without frame-pointer or perf-map support, yields flame graphs full of hex addresses. Build with symbols (and frame pointers for Go/C++/Rust where practical) and verify per-language support before rolling out fleet-wide.
- **Overhead is low, not zero**: typically a few percent at most (sampling ~100Hz), but heap and lock/mutex profiling can cost more. Enable expensive profile types selectively and measure on a canary first.
- **Label cardinality**: like Prometheus, labels index profiles. Putting request IDs or user IDs in tags explodes storage; stick to service, region, version, and similar low-cardinality dimensions.
- **Profiles are sampled aggregates**: a function that doesn't show up in a flame graph may just be rare. Interpret small differences with care, and use wall-clock/off-CPU views (not just CPU profiles) when a service is slow due to waiting on I/O or locks rather than computing.
- **Project history**: Pyroscope merged with Grafana Phlare and was re-architected as a horizontally scalable, object-storage-backed system (v1.0+). Older docs and the legacy standalone "Pyroscope agent/server" model differ from the current Grafana Pyroscope; check you are reading current docs.
- **Retention and cost**: profiles are less voluminous than logs, but still need an object store and a retention policy at fleet scale. Plan compaction and retention up front rather than after the disk fills.
