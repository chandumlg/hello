# lakeFS

lakeFS adds Git-style version control (branches, commits, merges, reverts) to a data lake on object storage, so you can change petabytes of S3, GCS or Azure Blob data in isolation and publish it atomically, without copying it.

## Why teams adopt it

A data lake is usually one big mutable bucket. A pipeline that writes half its output and then crashes leaves consumers reading a mix of old and new files. "Testing in staging" means copying terabytes into another prefix. Undoing a bad backfill means restoring object versions file by file, assuming versioning was even on. lakeFS addresses this by putting a versioned metadata layer in front of the bucket:

- **Zero-copy branches.** A branch is a pointer to a commit, and a commit is a tree of metadata ("ranges" and "metaranges") stored in your own bucket. Creating a branch of a 500 TB repository takes milliseconds and copies no data. Only the objects you change on the branch are written as new objects.
- **Atomic publish.** Writers work on a branch. When the output is complete and validated, you merge it into `main`, and every reader of `main` sees the whole change at once, or none of it.
- **Instant rollback.** `main` points at a commit, and reverting is a metadata operation. "Yesterday's load was wrong" turns into one command instead of an incident.
- **Reproducibility.** Every commit ID is an immutable snapshot of the lake. ML teams record the commit a model was trained on and can read exactly that data again months later.
- **Hooks (CI/CD for data).** Actions run on `pre-commit`, `pre-merge` and similar events, either as webhooks or as Lua scripts. Use them to enforce schema checks, required metadata, PII scans or format rules before data reaches `main`.
- **It works with existing tools.** lakeFS exposes an **S3-compatible gateway**, so Spark, Trino, DuckDB, pandas, boto3 and the AWS CLI work against it unchanged. The bucket name becomes the repository and the first path segment becomes the branch: `s3://analytics/main/events/...`.

The architecture is a stateless Go server (the API, the web UI and the S3 gateway) plus a key-value metadata store (PostgreSQL or DynamoDB in production). Data and commit metadata live in your object store under the repository's *storage namespace*. The CLI is `lakectl`, and there are Python, Java and Go SDKs.

Common adoption triggers:
- Partial writes or bad backfills have corrupted downstream dashboards or models, and recovering took days.
- You need a production-like environment to test a dbt/Spark/Airflow change, and copying the data is too slow or too expensive.
- ML or compliance work requires showing exactly which data produced a given result.
- Many independent jobs write to one lake, and you want a "merge gate" where quality checks run before data is published.

## Basic usage

**1. Run it locally and create a repository:**
```bash
# Quickstart mode: local storage and an embedded KV store, with fixed demo
# credentials printed on startup. Fine for learning, not durable.
docker run --pull always --name lakefs -p 8000:8000 \
  treeverse/lakefs:latest run --quickstart

# Point lakectl at it (interactive: endpoint http://localhost:8000, plus the key pair)
lakectl config

# Create a repository. In production the storage namespace is a real bucket
# prefix, e.g. s3://my-datalake/analytics. lakeFS owns everything under it.
lakectl repo create lakefs://analytics local://analytics

# Bring existing data under version control WITHOUT copying it
lakectl import --from s3://raw-bucket/events/ --to lakefs://analytics/main/events/
```

**2. The isolate → validate → merge loop:**
```bash
# Branch off main. This copies metadata only and is instant.
lakectl branch create lakefs://analytics/etl-2026-11-20 \
  --source lakefs://analytics/main

# Write to the branch (your Spark/dbt job would normally do this)
lakectl fs upload lakefs://analytics/etl-2026-11-20/events/dt=2026-11-20/part-0.parquet \
  --source ./part-0.parquet

# See what changed relative to main
lakectl diff lakefs://analytics/main lakefs://analytics/etl-2026-11-20

# Commit with metadata you can query later (job run IDs, git SHA of the code)
lakectl commit lakefs://analytics/etl-2026-11-20 \
  -m "Daily events load" --meta airflow_run_id=scheduled__2026-11-20 --meta code_sha=3f9c2e1

# Publish atomically. pre-merge hooks run here and can block the merge.
lakectl merge lakefs://analytics/etl-2026-11-20 lakefs://analytics/main

# Something wrong after the fact? Revert the merge commit on main.
lakectl log lakefs://analytics/main --amount 5
lakectl branch revert lakefs://analytics/main <merge-commit-id> --parent-number 1

# Stop anyone from writing straight to main
lakectl branch-protect add lakefs://analytics main
```

**3. Use it from engines through the S3 gateway, and gate merges with a hook:**
```python
# Spark: point s3a at lakeFS. Paths are s3a://<repo>/<branch-or-commit>/<path>
spark.conf.set("spark.hadoop.fs.s3a.endpoint", "https://lakefs.example.com")
spark.conf.set("spark.hadoop.fs.s3a.path.style.access", "true")
spark.conf.set("spark.hadoop.fs.s3a.access.key", LAKEFS_KEY)
spark.conf.set("spark.hadoop.fs.s3a.secret.key", LAKEFS_SECRET)

df = spark.read.parquet("s3a://analytics/main/events/")
# Read an exact historical snapshot by commit ID (reproducible training sets)
train = spark.read.parquet("s3a://analytics/a1b2c3d4e5/events/")
```
```python
# Python SDK (pip install lakefs): branch, write, commit, merge in code
import lakefs

repo = lakefs.repository("analytics")
branch = repo.branch("feature-dedupe").create(source_reference="main")
branch.object("events/fixed.parquet").upload(data=open("fixed.parquet", "rb").read())
branch.commit(message="Deduplicate events", metadata={"ticket": "DATA-912"})
branch.merge_into(repo.branch("main"))
```
```yaml
# _lakefs_actions/pre-merge-checks.yaml, committed INTO the repo like any object.
# Runs before anything merges into main; a non-2xx webhook response blocks the merge.
name: Validate before publishing to main
on:
  pre-merge:
    branches:
      - main
hooks:
  - id: schema_and_quality_check
    type: webhook
    properties:
      url: "http://data-quality-svc.internal/lakefs/validate"
      timeout: 2m
```

## Pitfalls and things to watch out for

- **Never write to the storage namespace directly.** Objects under the repository's bucket prefix have content-addressed physical names that lakeFS manages. Writing, deleting or applying lifecycle rules there behind lakeFS's back corrupts the repository. All writes must go through lakeFS (the API, the gateway or the SDKs).
- **Deleting data doesn't free storage until garbage collection runs.** Old commits keep referencing old objects, which is the whole point. Set GC retention rules (`lakectl gc set-config`) and schedule the GC job (a Spark job in the open-source version), or storage costs grow without limit. Abandoned branches with uncommitted writes add to this, so delete branches when a job finishes.
- **Conflicts are per object, not per row.** If both sides of a merge changed the same path, it's a conflict, and lakeFS can't reconcile the contents of two Parquet files. Design so that each job writes its own partitions, or merge with an explicit `--strategy source-wins` / `dest-wins` once you understand the consequences.
- **Uncommitted changes are shared by everyone on that branch.** Branches are not per-writer sandboxes. Two jobs writing to the same branch see each other's uncommitted data and can commit each other's half-written output. The rule is one branch per job run.
- **The S3 gateway is in the data path.** With a plain `s3a://` endpoint, every byte goes through the lakeFS servers, which then need to be sized and scaled like a proxy. For heavy Spark workloads, use the lakeFS Hadoop FileSystem (`lakefs://`) or presigned URLs, which fetch metadata from lakeFS but read and write data directly against S3.
- **It complements table formats, it doesn't replace them.** Iceberg and Delta have their own snapshots and time travel at the table level, and lakeFS versions files across the whole lake. Together they give cross-table atomic merges and environment isolation, but be clear about which layer is the source of truth for "the current version." For example, a Delta table's `_delta_log` lives on the lakeFS branch along with its data.
- **Quickstart isn't production.** Production needs a real KV store (PostgreSQL or DynamoDB), backups of that store, and HA for the stateless servers. Lose the metadata store without backups and you have a bucket full of opaque objects.
- **Know the open-source vs. Enterprise/Cloud split before you commit to it.** Features like fine-grained RBAC, SSO and some catalog integrations sit in the commercial tiers. Check the current feature matrix against your security requirements early.
- **Object stores don't rename, and neither does lakeFS.** Rename-heavy commit protocols (old Hadoop output committers, `_temporary` directories) do slow copy-plus-delete through the gateway. Use the S3A magic committer or a table format's commit protocol instead.

**Related tools:** Project Nessie (Git-like branching at the Iceberg catalog level rather than the file level), DVC (data versioning tied to a Git repo, aimed at ML artifacts on the client side), and the native branching and tagging in Iceberg and Delta for single-table workflows.
