# Sentry

## What problem does it solve?

Sentry is an error-tracking and application-monitoring platform that automatically captures, aggregates, and enriches exceptions, stack traces, and performance data from running applications so engineers stop relying on scattered logs and user bug reports to find out something broke in production.

## Primary use cases and when a team adopts it

- **Production error monitoring**: every unhandled exception (backend or frontend) is captured with a full stack trace, breadcrumbs (recent logs, HTTP calls, UI clicks) leading up to the crash, the release/commit it happened on, and which users/percentage of traffic were affected.
- **Release health tracking**: Sentry ties errors to specific releases, so a team can see a spike immediately after a deploy and correlate it with the commit that introduced it — this is often the trigger for adopting it, right after a "silent failure in prod" incident.
- **Performance monitoring / tracing**: distributed traces across services (similar in spirit to OpenTelemetry) surface slow endpoints, N+1 queries, and latency regressions, with the trace linked directly to the code that caused it.
- **Session Replay**: for frontend apps, Sentry can record a lightweight DOM-based replay of the user's session leading up to an error, which is invaluable for reproducing UI bugs that are otherwise "unreproducible."
- A team typically adopts Sentry once they have real users in production and are tired of debugging from `grep`-ing logs or waiting for a support ticket — it's usually one of the first observability tools added after logging, often before a full metrics/tracing stack like Prometheus/Grafana is in place.

## Basic usage examples

**1. Initialize the SDK (Node.js backend example):**
```javascript
// instrument.js — must be imported before any other module
const Sentry = require("@sentry/node");

Sentry.init({
  dsn: "https://<public_key>@o0.ingest.sentry.io/<project_id>",
  environment: process.env.NODE_ENV,
  release: process.env.GIT_SHA,
  tracesSampleRate: 0.1, // capture 10% of transactions for performance monitoring
});
```

**2. Capture an exception manually (in addition to automatic capture of unhandled errors):**
```javascript
try {
  await chargeCustomer(order);
} catch (err) {
  Sentry.captureException(err, {
    tags: { orderId: order.id },
    extra: { orderPayload: order },
  });
  throw err;
}
```

**3. Set up source maps (frontend) so minified stack traces are readable, using the Sentry CLI in your build pipeline:**
```bash
sentry-cli releases new "$GIT_SHA"
sentry-cli releases files "$GIT_SHA" upload-sourcemaps ./dist --url-prefix "~/static/js"
sentry-cli releases finalize "$GIT_SHA"
```

## Common pitfalls

- **PII leakage**: breadcrumbs, request bodies, and `extra` context can easily capture sensitive data (auth tokens, emails, payment info) unless you configure `beforeSend` scrubbing or enable Sentry's built-in data scrubbing — this is a real compliance risk, not just noise.
- **Unbounded event volume and cost**: Sentry bills by event volume, and a bad deploy or a noisy dependency can generate millions of events in minutes and blow through quota (or get you rate-limited) — set sensible `tracesSampleRate`/`sampleRate`, and use `ignoreErrors` or fingerprinting rules to suppress known-noisy errors instead of letting them flood the project.
- **Missing source maps or release tagging**: if releases aren't tagged and source maps aren't uploaded on every deploy, stack traces show minified/transpiled code, making errors nearly impossible to triage — this is the single most common "Sentry isn't useful" complaint and is almost always a CI pipeline gap.
- **Alert fatigue**: default alert rules fire on every new issue; without grouping/fingerprinting tuned and alert thresholds (e.g., "issue affects >N users") set deliberately, on-call channels get flooded and people start ignoring Sentry notifications entirely.
- **Confusing it with metrics/tracing tools**: Sentry's performance monitoring overlaps with OpenTelemetry/Prometheus but isn't a replacement for high-cardinality metrics or long-term aggregation — it's optimized for "what broke and why," not for dashboards of business/infra metrics.
