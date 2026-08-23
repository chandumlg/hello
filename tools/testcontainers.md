# Testcontainers

Testcontainers is a library that spins up real, throwaway instances of databases, message
brokers, and other backing services inside Docker containers for the lifetime of a test run, so
integration tests exercise the actual dependency instead of a mock or an in-memory stand-in.

## Why teams adopt it

Unit tests with mocked databases give false confidence: a mocked Postgres driver doesn't catch a
broken migration, a subtle SQL dialect difference, or a connection-pool exhaustion bug. Testcontainers
closes that gap by letting CI and local test suites talk to a genuine Postgres, Kafka, Redis, or
Elasticsearch container that's created before the test and destroyed after — no shared "test
database" to pollute, no snowflake environment to keep in sync.

Teams reach for it once they're tired of one of two failure modes: tests that pass locally against
a lightweight fake but fail in production against the real service, or a shared integration test
environment that's flaky because multiple CI runs stomp on the same database. It's a natural fit
for:

- **Service-level integration tests** — verify your repository layer, migrations, and queries
  against the real database engine and version you run in production.
- **Contract tests against infrastructure dependencies** — Kafka producers/consumers, Redis
  caching logic, S3-compatible object storage (via LocalStack or MinIO images).
- **CI pipelines** — since each run gets fresh, isolated containers, tests are reproducible and
  parallelizable without cross-test contamination.

It's less useful for pure unit tests (no I/O) or performance testing at scale — the container
startup overhead (seconds per container) makes it a poor fit for anything you need to run
thousands of times per second.

## Basic usage

Testcontainers has official bindings for Java, Go, Python, Node.js, .NET, and others. The shape is
the same everywhere: declare the container, start it, get a connection detail (host/port), point
your code at it.

**Java (JUnit 5), starting a Postgres container:**

```java
@Testcontainers
class OrderRepositoryTest {

    @Container
    static PostgreSQLContainer<?> postgres =
        new PostgreSQLContainer<>("postgres:16-alpine");

    @Test
    void savesAndReadsOrder() {
        var dataSource = DataSourceBuilder.create()
            .url(postgres.getJdbcUrl())
            .username(postgres.getUsername())
            .password(postgres.getPassword())
            .build();
        // run migrations, exercise the repository, assert
    }
}
```

**Python:**

```python
from testcontainers.postgres import PostgresContainer

with PostgresContainer("postgres:16-alpine") as postgres:
    engine = create_engine(postgres.get_connection_url())
    # run your test against `engine`
```

**Go:**

```go
ctx := context.Background()
container, err := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("test"),
    postgres.WithUsername("test"),
    postgres.WithPassword("test"),
)
defer container.Terminate(ctx)

connStr, _ := container.ConnectionString(ctx)
```

For anything without a dedicated module, use `GenericContainer` (Java/Go) or the equivalent to
start an arbitrary image and wait for a log line or exposed port before proceeding.

## Common pitfalls

- **Docker-in-Docker in CI.** Testcontainers needs a Docker daemon reachable from the test runner.
  Most hosted CI (GitHub Actions, GitLab, CircleCI) supports this out of the box, but self-hosted
  or sandboxed runners often don't — check for `/var/run/docker.sock` access before debugging test
  failures that are really environment failures.
- **Startup cost adds up.** Each container adds real seconds to test runs. Share a container across
  a whole test class/module instead of starting one per test method, and use the `Ryuk` resource
  reaper's default cleanup rather than tearing containers down manually mid-suite.
- **Port and state leakage between tests.** If you reuse a container across tests, truncate tables
  or reset state between tests explicitly — Testcontainers isolates you from *other test runs*, not
  from test pollution *within* a run.
- **Version drift from production.** Pin the container image tag to the same major version you run
  in production (e.g. `postgres:16-alpine`, not `postgres:latest`) — otherwise a passing test suite
  can hide a real incompatibility that only shows up when prod upgrades.
- **Not automatically parallel-safe.** Tests using a shared static container across parallel test
  classes can race each other. Either give each parallel worker its own container or ensure test
  data is namespaced/isolated.
- **Not a substitute for a working local Docker setup.** If a developer's machine can't run Docker
  (some corporate laptops, some CI vendors), the whole approach is blocked — have a documented
  fallback (mocks for local dev, containers only in CI) if this is a real constraint for your team.
