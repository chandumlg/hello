# Atlas

Atlas is a database schema-as-code tool that compares the schema you declare with the schema a database actually has, then plans, lints and applies the migration between them, so schema changes go through the same review and CI process as application code.

## Why teams adopt it

Most migration tools (Flyway, Liquibase, golang-migrate, Alembic without autogenerate) are runners. You write the `ALTER TABLE` statements by hand, and the tool applies them in order and records which ones ran. Nothing in that loop knows what the schema is supposed to look like, so nothing can tell you that the migration takes a table lock for ten minutes, that it drops a column still in use, or that production has drifted from what the migration files describe.

Atlas treats the desired schema as the source of truth, the way Terraform treats infrastructure. You describe the schema in plain SQL (`CREATE TABLE ...`), in Atlas's HCL dialect, or through an ORM (GORM, Ent, SQLAlchemy, Django, Sequelize, TypeORM, Hibernate and others have providers). Atlas inspects the target, computes the diff and generates the DDL.

It supports two workflows, and choosing between them is the first decision:
- **Declarative** (`atlas schema apply`): Atlas diffs the desired schema against the live database and applies the result directly, with a plan you approve. No migration files exist. This suits development databases, ephemeral environments and teams comfortable with Terraform-style plan and apply.
- **Versioned** (`atlas migrate diff`, then `atlas migrate apply`): Atlas diffs the desired schema against the current migration directory and writes a new SQL migration file. You review, edit and commit that file, and it is what runs in production. This is the more common choice for production because the exact SQL is reviewed ahead of time.

The core concepts:
- **Dev database**: a scratch database Atlas uses to normalize and validate SQL and to replay the migration directory. `docker://postgres/16/dev` makes Atlas start and discard a container for this. It is not your local development database.
- **Migration directory and `atlas.sum`**: versioned SQL files plus a checksum file that covers the whole directory. If two branches each add a migration, `atlas.sum` conflicts in Git, which forces someone to look at the ordering instead of finding out at deploy time.
- **Revisions table**: `atlas_schema_revisions` in the target database records which versions were applied, including partially applied ones.
- **Lint analyzers**: checks over a planned change for destructive changes, data-dependent changes (adding a unique index to a column with duplicates), backward-incompatible changes (renames) and, for some databases, table-locking operations.
- **`atlas.hcl`**: a project file that defines named environments (URL, dev URL, schema source, migration directory, lint policy), so commands shrink to `--env prod`.

Common adoption triggers:
- An outage came from a migration that looked harmless in review: a lock on a large table, a dropped column, a `NOT NULL` added without a default.
- The ORM's auto-migrate is being used against production and nobody can say what SQL it will run.
- Staging and production schemas have drifted through hotfixes, and migrations pass in one and fail in the other.
- A platform team wants one schema change process across many services and several database engines (PostgreSQL, MySQL, MariaDB, SQLite, SQL Server, ClickHouse and others).
- Teams deploy with Kubernetes or Terraform and want migrations to be part of that flow. Atlas has a Kubernetes operator, a Terraform provider and GitHub Actions.

## Basic usage

**1. Inspect an existing database and apply a change declaratively**

```bash
# Install the CLI
curl -sSf https://atlasgo.sh | sh

# Dump the current schema as SQL
atlas schema inspect \
  --url "postgres://app:pass@localhost:5432/app?sslmode=disable&search_path=public" \
  --format '{{ sql . }}' > schema.sql
```

Edit `schema.sql` as if you were writing it from scratch, for example adding a column:

```sql
CREATE TABLE users (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email      text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_seen  timestamptz            -- new
);
```

```bash
atlas schema apply \
  --url "postgres://app:pass@localhost:5432/app?sslmode=disable&search_path=public" \
  --to file://schema.sql \
  --dev-url "docker://postgres/16/dev?search_path=public"
```

Atlas prints the plan (`ALTER TABLE "users" ADD COLUMN "last_seen" timestamptz NULL;`) and asks for approval before running it. Add `--dry-run` to see the plan only.

**2. Versioned workflow: generate, lint and apply migration files**

```bash
# Generate a migration from the difference between migrations/ and schema.sql
atlas migrate diff add_last_seen \
  --dir file://migrations \
  --to file://schema.sql \
  --dev-url "docker://postgres/16/dev?search_path=public"

ls migrations/
# 20261115093012_add_last_seen.sql  atlas.sum

# Analyze the newest migration for risky changes
atlas migrate lint \
  --dir file://migrations \
  --dev-url "docker://postgres/16/dev?search_path=public" \
  --latest 1

# See what is pending, then apply
atlas migrate status --dir file://migrations --url "$DATABASE_URL"
atlas migrate apply  --dir file://migrations --url "$DATABASE_URL"
```

If you hand-edit a generated file, run `atlas migrate hash --dir file://migrations` to update `atlas.sum`, otherwise the next command refuses to run. For a database that already has tables, the first apply needs `--baseline <version>` to tell Atlas which migration represents the existing state.

**3. A project file and a CI check**

```hcl
# atlas.hcl
env "local" {
  src = "file://schema.sql"
  url = "postgres://app:pass@localhost:5432/app?sslmode=disable&search_path=public"
  dev = "docker://postgres/16/dev?search_path=public"

  migration {
    dir = "file://migrations"
  }
}

env "prod" {
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://migrations"
  }
}

lint {
  destructive {
    error = true
  }
}
```

```bash
atlas migrate diff add_orders --env local
atlas migrate apply --env prod
```

```yaml
# .github/workflows/schema.yml
on:
  pull_request:
    paths: ["migrations/**"]
jobs:
  lint:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env: { POSTGRES_PASSWORD: pass, POSTGRES_DB: dev }
        ports: ["5432:5432"]
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: ariga/setup-atlas@v0
        with:
          cloud-token: ${{ secrets.ATLAS_CLOUD_TOKEN }}
      - uses: ariga/atlas-action/migrate/lint@v1
        with:
          dir: file://migrations
          dir-name: app
          dev-url: postgres://postgres:pass@localhost:5432/dev?sslmode=disable&search_path=public
```

The action comments on the pull request with the analysis of each new migration file. It reports to Atlas Cloud, so it needs a token and a migration directory registered there under `dir-name`.

## Pitfalls and things to watch out for

- **Check the licensing and edition before you design around a feature.** The binary from `atlasgo.sh` is distributed under Atlas's own EULA, and a separately built Community Edition is Apache 2.0. Several things teams assume are included sit behind a login or a paid plan in recent releases: views, triggers, functions, row-level security and other non-table objects, some database drivers (SQL Server, ClickHouse and others), and `migrate lint` itself. Confirm against the current feature matrix for the version you install, because this boundary has moved more than once.
- **Unsupported objects are invisible, not errors.** If your edition doesn't manage triggers or functions, Atlas doesn't see them. It won't drop them, but it also won't create them in a new environment, and a schema diff will report "in sync" when it isn't.
- **The dev database gets wiped.** Atlas creates and drops objects in whatever `--dev-url` points to. Use the `docker://` form or a dedicated empty database, never a database with data you want. It must also match the target's engine and major version, or the generated SQL can differ from what production accepts.
- **URL scope changes the output.** A URL with `search_path=public` (PostgreSQL) or a database name (MySQL) puts Atlas in single-schema mode, and generated SQL has no schema qualifiers. Without it, Atlas manages every schema and emits `CREATE SCHEMA` and qualified names. The target URL and dev URL must use the same scope, and mixing them produces confusing diffs.
- **A rename looks like a drop plus an add.** A diff can't distinguish "renamed `name` to `full_name`" from "dropped one column and added another". Interactively Atlas asks; in CI or with auto-approve it may generate the destructive version. Review generated files and keep the destructive-change lint as an error.
- **Generated SQL is correct, not necessarily safe at scale.** Atlas will happily emit `CREATE INDEX` without `CONCURRENTLY`, or a `NOT NULL` column addition that rewrites a large table on older engines. Lint catches some of this, but edit the file where needed. Statements that can't run in a transaction need the `-- atlas:txmode none` directive at the top of the file.
- **Atlas doesn't do expand and contract for you.** It plans the change from schema A to schema B. Sequencing a backward-compatible rollout (add the new column, dual-write, backfill, switch reads, drop the old one) across several deploys is still your job, as is the backfill. Data migrations can be written into the versioned files by hand, but the diff engine only generates DDL.
- **Partial failures on engines without transactional DDL.** On MySQL, each DDL statement commits implicitly, so a migration that fails on statement three leaves the first two applied. Atlas records the partial state in the revisions table and resumes from the failed statement on the next run, but only if you fix the cause without editing the statements that already ran.
- **"Down" is computed, not stored.** There are no hand-written down files by default. `atlas migrate down` plans a reversal by diffing against the earlier state, and reversing a dropped column restores the column, not the data. Treat restoring from backup or rolling forward with a fix as the real recovery path.
- **Out-of-order versions after a merge.** A branch whose migration has an older timestamp than one already applied in production is rejected by default on apply. Rebase and regenerate the file (the `atlas.sum` conflict is the prompt to do this), and don't reach for the flag that allows out-of-order execution unless you've confirmed the two changes are independent.
- **Don't run both auto-migrate and Atlas.** If the ORM still calls its own auto-migration at startup, it will change the schema behind Atlas's back and the next diff will try to undo it. Turn the ORM's migration off and use it only as the schema source.
- **Migration credentials are powerful.** The role that applies migrations needs DDL rights and usually ownership. Keep it separate from the application's runtime role, and prefer running migrations as a deploy step or a Kubernetes Job over running them at application startup, where several replicas race for the lock.
