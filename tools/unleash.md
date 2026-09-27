# Unleash

Unleash is an open-source feature management platform that lets teams turn features on or off, roll them out gradually, and target specific users — all without deploying new code every time a flag needs to change.

## Primary use cases

Feature flags start as a scrappy `if (config.enabled)` check in one service, but that approach falls apart once an org has dozens of services, needs audit trails on who flipped what, or wants gradual rollouts instead of all-or-nothing releases. Unleash centralizes that logic into a single system that any service can query. Typical adopters:

- **Platform engineering teams** standardizing how every service exposes flags, so there's one dashboard and one API instead of five bespoke config tables scattered across the org.
- **Staff engineers driving progressive delivery** — shipping a risky change behind a flag, rolling it out to 1% of traffic, watching error rates, then dialing up to 100% without a redeploy, or killing it instantly if something breaks.
- **Teams running A/B tests or targeted rollouts** — enabling a feature only for beta customers, a specific region, or accounts on a certain plan, using Unleash's activation strategies and context fields (user ID, IP, custom attributes).
- **Orgs decoupling deploy from release** — code for a half-finished feature merges to main and ships to production dark, guarded by a flag, so deploys stay small and frequent while the feature itself launches on its own schedule.

A team typically adopts Unleash once flag sprawl becomes a liability: nobody can find which flags are stale, there's no audit log for who toggled a flag during an incident, or rollouts are still "deploy to 100% and pray" because building gradual-rollout logic per-service is too much overhead.

## Basic usage

Run the Unleash server (the self-hosted option, backed by Postgres) via Docker:

```bash
docker run -p 4242:4242 \
  -e DATABASE_URL="postgres://user:pass@host:5432/unleash" \
  unleashorg/unleash-server
```

Create a flag and a gradual rollout strategy through the Admin UI (or the API), then evaluate it from application code with an SDK. Example using the Node.js server-side SDK:

```javascript
const { initialize } = require('unleash-client');

const unleash = initialize({
  url: 'http://localhost:4242/api/',
  appName: 'checkout-service',
  customHeaders: { Authorization: '<client-api-token>' },
});

unleash.on('ready', () => {
  if (unleash.isEnabled('new-checkout-flow', { userId: '1234' })) {
    // serve the new flow
  } else {
    // serve the existing flow
  }
});
```

Target a rollout by percentage and by context (e.g., only internal staff or a specific plan tier) using a "Gradual rollout" strategy with a constraint, configured server-side — no code change needed to adjust the percentage or audience later:

```json
{
  "name": "gradualRollout",
  "parameters": { "rollout": "25", "stickiness": "userId" },
  "constraints": [
    { "contextName": "plan", "operator": "IN", "values": ["enterprise"] }
  ]
}
```

## Common pitfalls

- **Flag debt.** Flags are easy to add and easy to forget. Without a process to remove flags once a rollout hits 100% and stabilizes, the codebase accumulates dead conditionals and the flag list becomes unreadable. Treat flag cleanup as part of the "done" definition for a rollout, not an afterthought.
- **SDK caching and staleness.** Client SDKs poll for updates on an interval (or use streaming) and cache flag state locally so evaluation doesn't add latency or a hard dependency on the Unleash server being up. That means a flag toggle isn't instant everywhere — account for propagation delay, and make sure the SDK's local fallback behavior (what happens if the server is unreachable) matches what you want during an incident.
- **Stickiness and consistent bucketing.** Gradual rollouts hash on a stickiness field (usually a user or session ID) so the same user consistently lands on the same side of the flag. Picking an inconsistent or missing stickiness value (e.g., defaulting to random) causes users to flicker between old and new behavior on every request, which is confusing and breaks anything stateful.
- **Flags as a substitute for testing.** A flag reduces deploy risk but doesn't replace verifying the code path — teams sometimes ship untested code "because it's behind a flag" and then discover the disabled branch was broken all along the first time it's enabled for real traffic.
- **Too many context dimensions.** Combining several targeting constraints (region + plan + user attribute + percentage) makes it hard to reason about who actually sees a feature, and hard to debug support tickets like "why does this user see the old behavior." Keep targeting rules as simple as the rollout actually requires.
