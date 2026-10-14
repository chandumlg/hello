# Litestream

Litestream is an open-source tool that continuously streams a SQLite database's changes to object storage (S3, GCS, Azure Blob, SFTP, etc.) so you can run SQLite in production with durable backups and point-in-time restore, without a database server.

## Primary use cases

- **SQLite as a production database**: small-to-mid services, internal tools, and edge/single-node apps where SQLite's simplicity and speed (no network hop, no connection pool) beat running Postgres, but "the disk died and we lost everything" is unacceptable.
- **Disaster recovery for single-node apps**: continuous replication gives an RPO of roughly seconds instead of the hours you'd get from nightly `.backup` cron jobs.
- **Cheap, stateless-ish deployments**: on ephemeral platforms (containers, Fly.io, Nomad, Kubernetes with emptyDir), restore the DB from object storage on boot and replicate while running.
- **Point-in-time recovery**: restore to a specific timestamp or generation to recover from a bad migration or accidental `DELETE`.
- **Per-tenant / per-service databases**: many small SQLite files, each replicated independently by one Litestream process.

Teams adopt Litestream when they've decided SQLite fits the workload (single writer, modest write volume, read-heavy) and want operational safety without moving to a client/server database. It is *not* a high-availability or multi-writer solution — see pitfalls.

How it works: Litestream runs as a separate process (or wraps your app) and takes over SQLite's WAL checkpointing. It tails the write-ahead log, copies new WAL frames into shadow WAL segments, and periodically uploads them as LTX/WAL segments to the replica, plus periodic snapshots so restore doesn't have to replay unbounded history.

## Basic usage

**1. Install and configure** (`/etc/litestream.yml`):

```yaml
dbs:
  - path: /data/app.db
    replicas:
      - url: s3://my-bucket/app.db
        # credentials via LITESTREAM_ACCESS_KEY_ID / LITESTREAM_SECRET_ACCESS_KEY
        # or the usual AWS credential chain
        retention: 72h
        snapshot-interval: 6h
```

```bash
brew install litestream            # or download a release / use the Docker image
litestream replicate -config /etc/litestream.yml
```

**2. Run your app under Litestream** so it restarts together with the app (handy in containers):

```bash
litestream replicate -exec "/app/server --db /data/app.db"
```

**3. Restore** on a fresh machine, or to a point in time:

```bash
# Restore latest state (only if the local file doesn't exist)
litestream restore -if-db-not-exists -if-replica-exists /data/app.db

# Restore as of a timestamp
litestream restore -timestamp 2026-10-14T09:30:00Z -o /tmp/recovered.db s3://my-bucket/app.db

# Inspect what's available
litestream snapshots s3://my-bucket/app.db
litestream wal s3://my-bucket/app.db
```

A typical container entrypoint is: `restore -if-db-not-exists -if-replica-exists`, then `replicate -exec <app>`.

## Common pitfalls

- **Single writer, no HA**: replication is one-way and asynchronous. Running two instances against the same replica path corrupts the replica lineage. Failover means restoring on a new node, with potentially a few seconds of lost writes. If you need synchronous replication or failover, look at LiteFS, rqlite, or Postgres.
- **Async means data-loss window**: the default sync interval is about 1 second; anything not yet uploaded is lost in a crash. Tune `sync-interval`, and understand the cost/API-call trade-off.
- **WAL mode is required**: the database must be in WAL journal mode, and nothing else may checkpoint it. Don't run your own `PRAGMA wal_checkpoint(TRUNCATE)` or let other processes with different settings checkpoint — Litestream needs to manage this. Also avoid `PRAGMA journal_mode` changes at runtime.
- **Long-running read transactions** can block checkpoints and let the WAL grow unbounded; watch the `-wal` file size.
- **Local filesystems only**: the DB must be on a local disk, not NFS/EFS/network mounts where SQLite locking and shared memory are unreliable.
- **Test your restores**: a replica you've never restored from is a hope, not a backup. Automate a periodic `restore` + `PRAGMA integrity_check` in CI or a cron job.
- **Retention and cost**: snapshots plus WAL segments accumulate; set `retention` and `snapshot-interval` deliberately, and add bucket lifecycle rules. Frequent syncs produce many small PUTs.
- **Version differences**: newer Litestream releases changed the replica file format (LTX) and some config options; pin the version and read the release notes before upgrading, and make sure the restore binary matches the replica format.
- **Monitoring**: expose Litestream's Prometheus metrics (`addr:` in config) and alert on replication lag, otherwise silent failures (expired credentials, bucket permission changes) go unnoticed until you need the backup.
