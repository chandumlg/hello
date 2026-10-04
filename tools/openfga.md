# OpenFGA

OpenFGA is an open-source (CNCF) fine-grained authorization server, inspired by Google's Zanzibar, that lets you model "can user X do action Y on object Z?" as relationships in a graph and answer that question with a single API call instead of scattering permission logic through every service.

## Why teams adopt it

Hard-coded role checks (`if user.role == "admin"`) work until you need sharing, nested groups, per-resource permissions, or org hierarchies — then every service reinvents a slightly different, slightly buggy authorization layer. OpenFGA centralizes this as **relationship-based access control (ReBAC)**, which can also express RBAC and ABAC-style rules.

Common adoption triggers:
- Building Google-Docs-style sharing: "viewers of a folder can view every document in it," "editors can share with others."
- Multi-tenant SaaS where permissions depend on org → team → project → resource hierarchies.
- Platform teams wanting one authorization decision point (and one audit story) across many services and languages, rather than per-service policy code.
- Replacing a hand-rolled permissions table that has grown joins no one wants to touch.
- AI/RAG systems that must filter retrieved documents by what the *requesting user* may see (`ListObjects` / `Check` as a pre- or post-retrieval filter).

Core concepts: an **authorization model** (types and relations, written in a small DSL), **tuples** (facts like `user:anne is viewer of document:roadmap`), and the queries **Check**, **ListObjects**, **ListUsers**, and **Expand**.

## Basic usage

**1. Run the server locally:**
```bash
docker run -p 8080:8080 -p 3000:3000 openfga/openfga run
# HTTP API on :8080, Playground UI on :3000 (in-memory store; use Postgres/MySQL for real deployments)
```

**2. Create a store and a model (using the `fga` CLI):**
```bash
brew install openfga/tap/fga
export FGA_API_URL=http://localhost:8080

fga store create --name docs     # note the returned store id
export FGA_STORE_ID=<id>

cat > model.fga <<'MODEL'
model
  schema 1.1

type user

type folder
  relations
    define viewer: [user]

type document
  relations
    define parent: [folder]
    define editor: [user]
    define viewer: [user] or editor or viewer from parent
MODEL

fga model write --file model.fga
```

**3. Write tuples and ask questions:**
```bash
fga tuple write user:anne viewer folder:eng
fga tuple write folder:eng parent document:roadmap
fga tuple write user:bob editor document:roadmap

fga query check user:anne viewer document:roadmap     # allowed: true (inherited via folder)
fga query check user:carol viewer document:roadmap    # allowed: false
fga query list-objects user:bob viewer document       # which documents can bob view?
```

In application code you call the same `Check` API through an SDK (Go, Java, Node, Python, .NET) at your enforcement points, typically in middleware.

## Pitfalls and things to watch

- **Tuple sync is your job.** OpenFGA stores authorization facts, not your business data. When a user joins a team or a document is moved, your application must write/delete the corresponding tuples. Drift between your database and the tuple store is the most common source of bugs; write tuples in the same code path (or via an outbox/CDC pipeline) as the domain change.
- **Model design is the hard part.** Deep hierarchies, wide fan-out groups, and heavy use of `from` traversals make `Check` more expensive and `ListObjects` slower. Prototype models against realistic data volumes and use the model test format (`fga model test`) in CI.
- **Consistency.** Reads are eventually consistent by default with caching; newly written tuples may not be visible instantly. Use consistency options (e.g. higher-consistency reads) for "just granted access, now redirect" flows, and understand the "new enemy" problem when revoking access.
- **Model changes are versioned.** Each model write creates a new immutable model ID; tuples must conform to the model that evaluates them. Pin or pass `authorization_model_id` deliberately during migrations, and avoid removing relations that live tuples still use.
- **It's a new critical-path dependency.** Every request now depends on authorization latency and availability. Run it with a production datastore, size connection pools, use batch check for list pages, and decide up front whether you fail closed or open on outage (fail closed is almost always right).
- **Not an authentication system.** It answers "may this identity do this?", not "who is this?" — pair it with your IdP/OIDC. Also note it doesn't natively evaluate arbitrary attributes beyond conditions (CEL-based) on tuples, so keep rule complexity modest.
