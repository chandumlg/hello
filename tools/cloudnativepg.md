# CloudNativePG

CloudNativePG (CNPG) is a CNCF Kubernetes operator that runs highly available PostgreSQL clusters natively on Kubernetes — handling primary/standby replication, automated failover, backups to object storage, point-in-time recovery, and rolling upgrades — so you can treat Postgres as a declarative Kubernetes resource instead of a hand-tended pet.

## Primary use cases and when to adopt it

CNPG is unusual among Postgres-on-Kubernetes operators in that it does **not** use StatefulSets or an external consensus store (no Patroni, no etcd sidecar). The operator itself manages Pods and PVCs directly and uses the Kubernetes API as the source of truth for which instance is primary. That makes its behavior easier to reason about and integrates cleanly with GitOps.

Teams typically adopt it when:

- **You run a platform on Kubernetes and want self-service Postgres.** Product teams get a database by committing a `Cluster` manifest; the platform team owns the operator, backup policy, and monitoring once.
- **Managed Postgres (RDS/Cloud SQL) is too expensive, too restrictive, or unavailable** — e.g. on-prem, air-gapped, sovereign-cloud, or multi-cloud environments, or when you need extensions (pgvector, PostGIS, TimescaleDB) that a managed provider lags on.
- **You want ephemeral databases per preview environment or per test run.** Spinning up a cluster from a backup (or empty) is a single resource, and teardown is `kubectl delete`.
- **You need disaster recovery across regions.** CNPG supports "replica clusters" that continuously stream from object-storage WAL archives or a remote primary, and can be promoted in another Kubernetes cluster.
- **You want Postgres observability built in.** Each instance exposes Prometheus metrics on port 9187 with a customizable query set, and the operator ships a Grafana dashboard.

Key features worth knowing: synchronous or async replication with configurable quorum, automated switchover/failover, continuous WAL archiving and base backups to S3/GCS/Azure (via the Barman Cloud plugin in recent releases), volume-snapshot backups, PITR, declarative roles/databases, PgBouncer via a `Pooler` resource, in-place minor upgrades, and major-version upgrades via import or `pg_upgrade`-style flows (newer releases).

It's probably *not* the right choice if your org has no Kubernetes storage expertise, or if a managed database already meets your needs — you are signing up to own Postgres operations, just with much better tooling.

## Getting started

**1. Install the operator and create a 3-instance cluster:**

```bash
# Install the operator (pin to the current release from cloudnative-pg.io)
kubectl apply --server-side -f \
  https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.25/releases/cnpg-1.25.0.yaml

# Optional but recommended: the kubectl plugin
curl -sSfL https://github.com/cloudnative-pg/cloudnative-pg/raw/main/hack/install-cnpg-plugin.sh \
  | sudo sh -s -- -b /usr/local/bin
```

```yaml
# pg-cluster.yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: orders-db
spec:
  instances: 3                 # 1 primary + 2 streaming replicas
  imageName: ghcr.io/cloudnative-pg/postgresql:17
  storage:
    size: 50Gi
    storageClass: fast-ssd
  postgresql:
    parameters:
      max_connections: "200"
      shared_buffers: "1GB"
  bootstrap:
    initdb:
      database: orders
      owner: orders_app
  affinity:
    enablePodAntiAffinity: true
    topologyKey: topology.kubernetes.io/zone   # spread across AZs
  monitoring:
    enablePodMonitor: true
```

```bash
kubectl apply -f pg-cluster.yaml
kubectl cnpg status orders-db
```

CNPG creates three Services: `orders-db-rw` (always points to the primary), `orders-db-ro` (replicas only), and `orders-db-r` (any instance). Application credentials land in the Secret `orders-db-app`, which includes a ready-made `uri` key:

```yaml
env:
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: orders-db-app
        key: uri
```

**2. Continuous backups to object storage and point-in-time recovery:**

```yaml
# Add to the Cluster spec (in-tree barmanObjectStore; newer releases
# move this to the Barman Cloud plugin with an ObjectStore resource)
spec:
  backup:
    retentionPolicy: "30d"
    barmanObjectStore:
      destinationPath: s3://my-pg-backups/orders-db
      s3Credentials:
        accessKeyId:     { name: aws-creds, key: ACCESS_KEY_ID }
        secretAccessKey: { name: aws-creds, key: SECRET_ACCESS_KEY }
      wal:
        compression: gzip
---
apiVersion: postgresql.cnpg.io/v1
kind: ScheduledBackup
metadata:
  name: orders-db-nightly
spec:
  schedule: "0 0 2 * * *"      # six-field cron: 02:00 daily
  cluster:
    name: orders-db
```

Restoring to a point in time is a *new* cluster bootstrapped from the archive:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: orders-db-restore
spec:
  instances: 1
  storage: { size: 50Gi }
  bootstrap:
    recovery:
      source: orders-db
      recoveryTarget:
        targetTime: "2026-11-26 14:05:00+00"
  externalClusters:
    - name: orders-db
      barmanObjectStore:
        destinationPath: s3://my-pg-backups/orders-db
        s3Credentials:
          accessKeyId:     { name: aws-creds, key: ACCESS_KEY_ID }
          secretAccessKey: { name: aws-creds, key: SECRET_ACCESS_KEY }
```

**3. Day-2 operations with the kubectl plugin:**

```bash
kubectl cnpg backup orders-db                     # on-demand backup
kubectl cnpg promote orders-db orders-db-2        # planned switchover to instance 2
kubectl cnpg psql orders-db -- -c 'select now()'  # psql into the primary
kubectl cnpg restart orders-db                    # rolling restart (replicas first)
kubectl cnpg report cluster orders-db             # support bundle for debugging
```

Changing `imageName` to a newer minor version (e.g. `17.2` → `17.4`) triggers a rolling update: replicas are updated first, then a switchover, then the old primary — controlled by `primaryUpdateStrategy: unsupervised | supervised`.

## Common pitfalls

- **Untested backups aren't backups.** Configure `ScheduledBackup` *and* WAL archiving, then regularly restore into a throwaway cluster. Check `kubectl cnpg status` for "Continuous Backup status: OK" — a misconfigured bucket or expired credential silently leaves you with only replicas, which replicate your `DROP TABLE` too.
- **Storage is the real HA boundary.** Use a storage class with `volumeBindingMode: WaitForFirstConsumer` and spread instances across zones; three instances on the same node or zonal disk is one failure domain. Local NVMe gives great performance but means a lost node = rebuild that replica from scratch (fine, but plan for the I/O).
- **Size PVCs with growth headroom, and separate WAL.** A full disk is the most common Postgres outage. Use `walStorage` for a dedicated WAL volume, make sure the storage class supports expansion, and alert on the `cnpg_pg_wal` / filesystem metrics.
- **Don't fight the operator.** Changing `postgresql.conf` with `ALTER SYSTEM`, editing Pods directly, or manually promoting with `pg_ctl` causes drift the operator will revert or misinterpret. Make all changes via the `Cluster` spec (some parameters are operator-managed and cannot be overridden at all).
- **Connection storms during failover.** Failover takes seconds, but clients connected to the old primary get dropped. Use the `-rw` Service (never a Pod IP), put PgBouncer in front via a `Pooler` resource, and make sure apps retry. Also review `.spec.failoverDelay` and `switchoverDelay` rather than assuming defaults suit you.
- **Synchronous replication trades availability for durability.** With `minSyncReplicas`/`maxSyncReplicas` (or the newer `synchronous` stanza) set strictly, losing replicas can block writes on the primary. Understand the quorum you've configured before an incident does it for you.
- **Operator upgrades restart instances.** Upgrading the operator can roll every cluster's instance manager. Read the release notes, stage it in non-prod, and know whether your version supports in-place instance-manager upgrades.
- **Major-version upgrades need a plan.** Bumping `imageName` from 16 to 17 is not a rolling minor update. Use `bootstrap.initdb.import` (logical dump/restore into a new cluster), logical replication for near-zero downtime, or the in-place major upgrade support in recent releases — and rehearse it.
- **Plugin migration for backups.** Recent CNPG releases deprecate the in-tree `barmanObjectStore` in favor of the Barman Cloud CNPG-I plugin. New deployments should start on the plugin to avoid a migration later.
