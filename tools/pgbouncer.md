# PgBouncer

**PgBouncer is a lightweight connection pooler for PostgreSQL that lets thousands of client connections share a small number of real server connections, so the database isn't overwhelmed by per-connection process overhead.**

## Why it exists

Postgres forks a backend process per connection. Each costs memory (several MB or more), and performance degrades well before you hit `max_connections`, because of lock contention, snapshot scanning and context switching. Modern architectures make this worse: autoscaled app pods, serverless functions, and per-worker ORM pools can each open dozens of connections. PgBouncer sits between clients and Postgres, speaks the Postgres wire protocol, and multiplexes many client connections onto a small, bounded server pool.

## Primary use cases and when to adopt

- **Connection storms from horizontally scaled apps.** 200 pods x 20 pool connections = 4,000 connections. PgBouncer collapses that to maybe 50-100 server connections.
- **Serverless and short-lived workloads** (Lambda, Cloud Run, cron jobs) where connection setup cost (TCP + TLS + auth) dominates the query time.
- **Protecting the primary** with a hard cap per database/user, so a runaway service can't starve others.
- **Operational control:** `PAUSE`/`RESUME` lets you swap or fail over the backend without dropping clients; `RELOAD` changes config live.

Adopt it when you see `FATAL: too many connections`, high backend memory, or `idle` connections dominating `pg_stat_activity`. Many managed offerings (RDS Proxy, Supabase, Cloud SQL, Neon) bundle similar poolers, but self-hosted PgBouncer remains the common, transparent option.

### Pool modes (the key decision)

| Mode | Server connection returned to pool | Notes |
|---|---|---|
| `session` | when client disconnects | Safest, least multiplexing |
| `transaction` | at end of each transaction | Best reuse; the usual production choice |
| `statement` | after each statement | No multi-statement transactions; rare |

## Getting started

**1. Minimal `pgbouncer.ini`:**

```ini
[databases]
appdb = host=10.0.0.5 port=5432 dbname=appdb

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 5000     ; clients PgBouncer will accept
default_pool_size = 50     ; server conns per (db, user) pair
reserve_pool_size = 10
server_idle_timeout = 600
ignore_startup_parameters = extra_float_digits
```

`userlist.txt` holds `"appuser" "SCRAM-SHA-256$4096:..."` lines (copy the verifier from `pg_authid`), or use `auth_query` to look users up in the database instead.

**2. Run and connect (apps just change the port/host):**

```bash
pgbouncer -d /etc/pgbouncer/pgbouncer.ini
psql "host=localhost port=6432 dbname=appdb user=appuser"
```

**3. Inspect via the admin console:**

```bash
psql -h localhost -p 6432 -U pgbouncer pgbouncer   # needs admin_users set
pgbouncer=# SHOW POOLS;     -- cl_active, cl_waiting, sv_active, sv_idle, maxwait
pgbouncer=# SHOW STATS;     -- per-db qps, bytes, avg query/wait time
pgbouncer=# SHOW CLIENTS;
pgbouncer=# RELOAD;         -- re-read config without dropping clients
```

Watch `cl_waiting` and `maxwait`: if clients queue, the pool is too small or queries are too slow.

## Common pitfalls

- **Transaction mode breaks session state.** Session-level features don't survive across transactions: `SET` (without `SET LOCAL`), advisory locks (`pg_advisory_lock`), `LISTEN/NOTIFY`, `WITH HOLD` cursors, temp tables, and `DISCARD`-dependent logic. Use `SET LOCAL`, transaction-level advisory locks (`pg_advisory_xact_lock`), or route those clients to a session-mode pool/database entry.
- **Prepared statements.** Historically unsupported in transaction mode, causing `prepared statement "S_1" does not exist` errors (common with JDBC, asyncpg, Npgsql, Prisma). PgBouncer 1.21+ supports protocol-level prepared statements via `max_prepared_statements` (set it non-zero); on older versions, disable client-side prepared statements.
- **Pool sizing math.** `default_pool_size` is per (database, user) pair, so total server connections can multiply across users/databases. Keep the sum below Postgres `max_connections` minus headroom for superuser, replication and migrations.
- **Single-threaded.** One PgBouncer process uses one core. Past a point (heavy TLS, very high qps) run several instances with `so_reuseport = 1` behind the same port, or in front of separate Postgres instances.
- **TLS and auth mismatches.** Client-side and server-side TLS are configured separately (`client_tls_*`, `server_tls_*`). Mismatched `auth_type` or stale `userlist.txt` hashes after password rotation are a frequent cause of login failures; prefer `auth_query` to avoid drift.
- **Not a load balancer or failover manager.** It doesn't detect primary failure or split reads/writes. Pair it with Patroni/HAProxy or DNS/VIP failover, and use `PAUSE` during planned switchovers.
- **Hides client identity.** Server-side `pg_stat_activity` shows PgBouncer's connection, not the app. Use `application_name` (and `SHOW CLIENTS`) for attribution.
- **Idle-in-transaction clients pin server connections** in transaction mode. Set `idle_transaction_timeout` and fix application code that holds transactions open across network calls.
