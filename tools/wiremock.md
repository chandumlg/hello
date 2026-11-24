# WireMock

**WireMock is an HTTP API simulator: it stands up a real HTTP server that returns stubbed responses you define, so you can test and develop against third-party or internal APIs without depending on the real thing.**

## Primary use cases and when to adopt it

- **Integration tests without flaky dependencies:** Replace Stripe, an OAuth provider, a partner API or a sibling microservice with a deterministic stub. Your code makes real HTTP calls over real sockets (real serialization, real timeouts, real TLS if you want), but the answers are under your control.
- **Testing the unhappy paths:** Real APIs rarely return a 503, a 30-second stall, a malformed body or a dropped connection on demand. WireMock does, through fault injection and fixed or random delays, so you can actually verify retries, circuit breakers and timeout handling.
- **Contract-ish verification of outbound calls:** Beyond stubbing, WireMock records every request it receives. Tests can assert "we called `POST /v1/charges` exactly once with this idempotency key and this JSON body."
- **Parallel frontend/backend development:** Teams stub an API that doesn't exist yet from an agreed OpenAPI spec or hand-written mappings, so consumers aren't blocked on producers.
- **Record and playback:** Proxy traffic through WireMock to a real service once, capture the responses as stub files, then replay them offline in CI.
- **Shared dev/staging environments:** Run WireMock as a container (or WireMock Cloud) standing in for expensive or rate-limited external systems in ephemeral preview environments.

**Adopt it when** your test suite is slow or flaky because it calls real external services, when you can't reproduce failure modes in a dependency, or when a sandbox environment is missing, rate-limited or costs money per call. It's language-agnostic: the core is Java, but the standalone server/Docker image is driven by JSON files or a REST admin API, and there are client libraries for Java, Python, Go, .NET, Node and more. It pairs well with **Testcontainers** (lifecycle) and **Pact** (consumer-driven contracts). Pact checks that both sides agree on a contract. WireMock simulates one side so you can test your behavior.

## Getting started

### 1. Run it standalone with Docker and JSON stubs

```bash
mkdir -p wiremock/mappings wiremock/__files

cat > wiremock/mappings/get-user.json <<'JSON'
{
  "request": {
    "method": "GET",
    "urlPathPattern": "/api/users/[0-9]+"
  },
  "response": {
    "status": 200,
    "headers": { "Content-Type": "application/json" },
    "jsonBody": { "id": 42, "name": "Ada", "plan": "pro" }
  }
}
JSON

docker run -d --name wiremock -p 8080:8080 \
  -v "$PWD/wiremock:/home/wiremock" \
  wiremock/wiremock:3.9.1 --verbose

curl -s localhost:8080/api/users/42
# {"id":42,"name":"Ada","plan":"pro"}
```

Stubs can also be added at runtime through the admin API, which is how non-JVM test suites usually drive it:

```bash
curl -s -X POST localhost:8080/__admin/mappings -d '{
  "request":  { "method": "POST", "url": "/api/payments" },
  "response": { "status": 503, "fixedDelayMilliseconds": 2000 }
}'

# What did the system under test actually send?
curl -s localhost:8080/__admin/requests | jq '.requests[].request.url'
```

### 2. Use it inside a JUnit 5 test (Java/Kotlin)

```java
@WireMockTest(httpPort = 8089)
class PaymentClientTest {

  @Test
  void retriesOnServiceUnavailable() {
    stubFor(post("/v1/charges")
        .inScenario("flaky").whenScenarioStateIs(STARTED)
        .willReturn(serviceUnavailable())
        .willSetStateTo("recovered"));

    stubFor(post("/v1/charges")
        .inScenario("flaky").whenScenarioStateIs("recovered")
        .willReturn(okJson("{\"id\":\"ch_123\",\"status\":\"succeeded\"}")));

    var client = new PaymentClient("http://localhost:8089");
    var result = client.charge(1000, "usd");

    assertEquals("succeeded", result.status());
    verify(2, postRequestedFor(urlEqualTo("/v1/charges"))
        .withHeader("Idempotency-Key", matching(".+"))
        .withRequestBody(matchingJsonPath("$.amount", equalTo("1000"))));
  }
}
```

Two features show up here. **Scenarios** are small state machines that let one endpoint return different responses on successive calls (first a failure, then a success), which is how you test retry logic. **Verification** asserts on what your client actually sent.

### 3. Record real traffic, then replay it offline

```bash
docker run -d --name wm-rec -p 8080:8080 -v "$PWD/wiremock:/home/wiremock" \
  wiremock/wiremock:3.9.1

# Start recording: WireMock proxies to the real API and captures responses
curl -s -X POST localhost:8080/__admin/recordings/start \
  -d '{"targetBaseUrl": "https://api.github.com"}'

curl -s localhost:8080/repos/wiremock/wiremock > /dev/null   # exercise the API

# Stop: captured stubs are written to wiremock/mappings and __files
curl -s -X POST localhost:8080/__admin/recordings/stop
```

Commit the generated mappings and your CI suite now runs with no network access. Trim them by hand first, though (see pitfalls).

Also worth knowing: **response templating** (`"transformers": ["response-template"]`, enabled globally in 3.x) lets responses echo request data, e.g. `{{jsonPath request.body '$.orderId'}}`. **Faults** (`"fault": "CONNECTION_RESET_BY_PEER"`, `"EMPTY_RESPONSE"`, `"MALFORMED_RESPONSE_CHUNK"`) simulate network-level failures that a plain status code can't.

## Common pitfalls

- **Stubs drift from reality.** A WireMock stub is your *belief* about the API, and it doesn't fail when the provider changes. Green tests can hide a broken integration. Mitigate by generating stubs from the provider's OpenAPI spec, periodically re-recording, pairing with Pact or provider contract tests, and keeping a small number of real end-to-end smoke tests.
- **Overly loose request matching.** `urlPathPattern: ".*"` with no body matchers means your test passes even if the client sends garbage. Match on the method, path, important headers and key body fields, and add `verify(...)` assertions. A stub *responding* proves nothing about what was *sent*.
- **Overly strict matching.** The opposite problem: `equalToJson` on a full body with timestamps or generated IDs breaks on every run. Use `matchingJsonPath`, `ignoreExtraElements`, or `${json-unit.any-string}` placeholders.
- **Unmatched requests return 404 silently.** A typo in a stub just looks like "the API returned 404", which your code may handle gracefully, so the test can still pass. Run with `--verbose`, check `/__admin/requests/unmatched` (or use `WireMockExtension` with `failOnUnmatchedRequests(true)`), and read the near-miss diff WireMock prints.
- **Stub priority surprises.** When several stubs match, the most recently added wins unless you set `"priority"` (lower number wins). Catch-all fallback stubs should get a high priority number so they never shadow specific ones.
- **Shared state between tests.** Scenarios, stubs and the request journal persist on a long-lived server. Call `WireMock.reset()` / `POST /__admin/reset` between tests, or use a server per test class. Use dynamic ports (`@WireMockTest` without a fixed port, or `--port 0`) to avoid collisions when tests run in parallel.
- **Recordings capture secrets and noise.** Recorded mappings can contain auth headers, tokens, PII and volatile fields. Scrub them before committing (`"captureHeaders"` and `"requestBodyPattern"` options, or just edit), and delete the stubs you don't need. Hundreds of auto-generated files become unmaintainable fast.
- **Base URLs hardcoded in the client.** WireMock only helps if the code under test lets you inject the endpoint (config, env var, DI). Make the base URL configurable everywhere before adopting it. Don't rely on DNS or `/etc/hosts` tricks.
- **Not a load-testing target or a production fake.** WireMock handles moderate throughput fine, but it isn't tuned as a high-RPS mock for benchmarks (use k6 against a purpose-built stub, or tune Jetty threads carefully). Don't let a "temporary" WireMock quietly become a production dependency.
- **2.x → 3.x migration.** 3.x requires Java 11+, moved to the `org.wiremock` group ID for some artifacts, enables templating by default and changed the JUnit 5 extension APIs. Pin versions and read the upgrade notes rather than bumping blindly.
