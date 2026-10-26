# etcd

## What it is and the problem it solves

etcd is a strongly consistent, distributed key-value store built on the Raft consensus algorithm, giving distributed systems a reliable place to keep small, critical pieces of state (configuration, leader leases, service membership) that must never diverge between nodes.

## Primary use cases and when to adopt it

- **Kubernetes control plane**: etcd is the only stateful component of the API server; every object (Pods, Secrets, CRDs) lives there. Most teams "adopt" it implicitly, and the real question is how to operate and back it up.
- **Leader election and distributed locks**: leases plus transactions (`Txn`) give you fencing-friendly primitives. `go.etcd.io/etcd/client/v3/concurrency` ships ready-made election and mutex recipes.
- **Service discovery and dynamic config**: clients `Watch` a key prefix and get an ordered, gap-free stream of changes (via revisions), which beats polling.
- **Coordination for in-house platforms**: schedulers, shard maps, feature-flag backends, and metadata for storage systems (e.g. CoreDNS, Patroni can use it as a DCS).

Adopt it directly when you need linearizable reads and compare-and-swap semantics over a *small* dataset. Don't adopt it as a general database: the default storage quota is 2 GiB (recommended max ~8 GiB), and every write goes through consensus and an fsync.

## Getting started

**1. Run a local single node and do basic reads and writes** (etcd v3 API):

```bash
docker run -d --name etcd -p 2379:2379 gcr.io/etcd-development/etcd:v3.5.16 \
  etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379

export ETCDCTL_API=3
etcdctl put /config/feature-x enabled
etcdctl get /config/ --prefix
etcdctl watch /config/ --prefix        # streams changes in another terminal
```

**2. Leases (TTL keys) and compare-and-swap transactions:**

```bash
LEASE=$(etcdctl lease grant 30 | awk '{print $2}')
etcdctl put /locks/job-42 owner-a --lease=$LEASE   # key vanishes in 30s unless kept alive
etcdctl lease keep-alive $LEASE

# Only set if the key does not yet exist (create revision == 0)
etcdctl txn <<'TXN'
create("/locks/job-43") = "0"

put /locks/job-43 owner-a


TXN
```

**3. Inspect health and take a snapshot backup:**

```bash
etcdctl endpoint status --cluster -w table   # leader, DB size, raft index
etcdctl endpoint health --cluster
etcdctl snapshot save backup.db
etcdutl snapshot status backup.db -w table    # offline verification (etcdutl, not etcdctl)
```

For a real cluster, run an odd number of members (3 or 5), with mTLS between peers and clients (`--peer-*` and `--cert-file/--key-file/--trusted-ca-file` flags).

## Common pitfalls

- **Disk latency is your latency.** Raft requires fsync on every commit. Slow or shared disks cause missed heartbeats, leader flapping, and "apply request took too long" warnings. Use dedicated SSD/NVMe; watch `etcd_disk_wal_fsync_duration_seconds` (p99 should be under ~10ms).
- **Database growth and the quota alarm.** etcd keeps MVCC history. Without compaction and defragmentation the DB hits `--quota-backend-bytes`, raises a `NOSPACE` alarm, and goes read-only. Enable auto-compaction (`--auto-compaction-retention`), run `etcdctl defrag` one member at a time, then `etcdctl alarm disarm`.
- **Cluster size is not a scaling knob.** More members means *slower* writes (larger quorum), not more throughput. 3 members tolerate 1 failure; 5 tolerate 2; even sizes add risk without benefit.
- **Quorum loss is an outage.** Losing a majority makes the cluster unavailable for writes (and linearizable reads). Recovery means restoring a snapshot into a new cluster (`etcdutl snapshot restore`), so practice it and back up regularly. Never restore the same snapshot onto only some members.
- **Large values and hot keys.** Requests are capped (~1.5 MiB by default); big values and heavy watch fan-out hurt everyone. Keep values small and store blobs elsewhere.
- **Serializable vs linearizable reads.** `--consistency=s` (serializable) reads can be stale but are faster and survive partitions; use the default linearizable reads when correctness matters.
- **Version and API drift.** The v2 API is gone, and `etcdctl snapshot` restore/status moved to `etcdutl`. Upgrade one minor version at a time and check client/server compatibility.
