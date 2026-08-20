# Backstage

Backstage is an open-source framework (originally built at Spotify) for
building an internal developer portal — a single catalog and UI where
engineers can discover every service, its owner, its docs, and its
deployment status, instead of that knowledge being scattered across wikis,
Slack threads, and tribal memory.

## Why teams adopt it

Past a certain org size, "which team owns this service, where's the repo,
and how do I deploy a new one" stops being answerable by asking around.
Backstage centralizes that into a **software catalog**: every service,
library, website, and API is registered as an *entity* with metadata (owner,
lifecycle stage, links to repo/docs/dashboards) that other tools can query.

Typical adoption triggers:

- **Service sprawl with no single inventory.** Once a company has 50+
  microservices, nobody — not even the platform team — has an accurate
  mental model of what exists, who owns it, and what depends on what.
  Backstage's catalog becomes the source of truth, generated from YAML files
  (`catalog-info.yaml`) checked into each repo.
- **Inconsistent "golden path" for new services.** Without a scaffolder,
  every team bootstraps a new service differently — different CI configs,
  different logging setup, different Dockerfile conventions. Backstage
  **Software Templates** let a platform team codify "click a button, fill in
  a name, get a repo with CI/CD, observability, and on-call already wired
  up" — cutting new-service setup from days to minutes.
- **Docs and dashboards scattered everywhere.** TechDocs (Backstage's
  docs-as-code plugin) renders each service's Markdown docs, checked into
  its own repo, right next to the catalog entry — so docs live with the code
  and don't rot in a separate wiki.
- **Plugin ecosystem needs.** Backstage ships plugins (and a large
  community ecosystem) for Kubernetes, CI status, cost insights, PagerDuty,
  ArgoCD sync state, and more — all surfaced per-service in one place rather
  than requiring engineers to bounce between a dozen dashboards.

It's most valuable for platform engineering teams at a scale (typically
100+ engineers, dozens-to-hundreds of services) where developer experience
and standardization have become an explicit investment, not just something
each team figures out on its own.

## Getting started

**1. Scaffold a new Backstage app:**

```bash
npx @backstage/create-app@latest
cd my-backstage-app
yarn install
yarn dev   # runs frontend on :3000, backend on :7007
```

**2. Register a service by adding a `catalog-info.yaml` to its repo:**

```yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: payments-service
  description: Handles payment processing and refunds
  annotations:
    github.com/project-slug: acme/payments-service
    backstage.io/techdocs-ref: dir:.
spec:
  type: service
  lifecycle: production
  owner: team-payments
  system: checkout
```

Then register it in the catalog, either via the UI ("Register Existing
Component") or by adding it to `app-config.yaml`:

```yaml
catalog:
  locations:
    - type: url
      target: https://github.com/acme/payments-service/blob/main/catalog-info.yaml
```

**3. Query the catalog** via the built-in API once entities are registered:

```bash
curl http://localhost:7007/api/catalog/entities?filter=kind=component
```

From here, the service shows up in the catalog UI with its owner, links,
and (if TechDocs is configured) rendered documentation — and other plugins
(CI status, Kubernetes pods, on-call) attach to the same entity page.

## Common pitfalls

- **Catalog rot.** The catalog is only as good as the YAML files backing
  it. Without enforcement (a CI check requiring `catalog-info.yaml`, or a
  bot that flags services missing an owner), the catalog drifts out of sync
  with reality just like the wiki it replaced.
- **Underestimating the operational cost.** Backstage is a Node.js/React
  app plus a backend and database (Postgres) that your platform team now
  owns and runs — upgrades, plugin compatibility, and auth integration are
  real ongoing work, not a one-time setup.
- **Plugin version compatibility.** The plugin ecosystem moves fast and
  isn't always in lockstep with core Backstage releases; upgrading core can
  break third-party plugins, so pin versions and test upgrades in a non-prod
  instance first.
- **Over-customizing the scaffolder templates.** Software Templates are
  powerful but easy to over-engineer with dozens of conditional inputs.
  Start with a small number of opinionated "golden path" templates rather
  than trying to parameterize every possible service variant up front.
- **Ownership metadata without enforcement drifts.** `owner: team-payments`
  in YAML doesn't stay accurate on its own — pair it with org data (e.g.,
  synced from an identity provider or ownership file) rather than letting it
  be hand-edited and forgotten after a team reorg.
