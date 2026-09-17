# Redis

Redis is an in-memory data structure store that solves the problem of needing sub-millisecond reads and writes for data that doesn't need to live in a full relational database — caches, session stores, counters, queues, leaderboards, and real-time pub/sub all fit naturally into its data model.

## Primary use cases

- **Caching**: the canonical use case — sit Redis in front of a slower database or API to absorb read load and cut latency from tens of milliseconds to sub-millisecond. Cache-aside (read-through) is the most common pattern.
- **Session storage**: web applications store session tokens and user state in Redis because it's fast and supports TTLs natively, so sessions expire without a cleanup job.
- **Rate limiting and counters**: atomic `INCR`/`EXPIRE` combos make Redis a natural fit for request throttling, leaderboards, and analytics counters.
- **Pub/sub and lightweight messaging**: Redis's `PUBLISH`/`SUBSCRIBE` and Streams (`XADD`/`XREAD`) support fan-out messaging and simple event pipelines when you don't need Kafka-grade durability guarantees.
- **Distributed locks and coordination**: `SET key value NX EX ttl` gives a simple mutual-exclusion primitive across processes (the Redlock algorithm formalizes this for multi-node safety, though it's debated for strict correctness).

A team adopts Redis when a data access pattern is read-heavy, latency-sensitive, and tolerant of eventual persistence — i.e., losing the last few seconds of data on a crash is acceptable, or you're using it purely as a cache with a durable source of truth behind it.

## Basic usage

**1. Run it locally and connect with the CLI:**

```bash
docker run -d --name redis -p 6379:6379 redis:7
redis-cli -h localhost -p 6379
```

**2. Basic key-value operations with expiry (a cache):**

```
SET user:42:profile '{"name":"Ada"}' EX 300   # expires in 300s
GET user:42:profile
DEL user:42:profile
```

**3. Data structures beyond strings — a sorted set for a leaderboard, a hash for an object, and atomic increment for a rate limiter:**

```
ZADD leaderboard 1500 "player1" 1800 "player2"
ZREVRANGE leaderboard 0 2 WITHSCORES        # top 3 scores

HSET user:42 name "Ada" plan "pro"
HGETALL user:42

INCR api:ratelimit:user42
EXPIRE api:ratelimit:user42 60
```

From a language client (Python example, using `redis-py`):

```python
import redis
r = redis.Redis(host="localhost", port=6379, decode_responses=True)
r.set("greeting", "hello", ex=60)
print(r.get("greeting"))
```

## Common pitfalls

- **Treating it as a primary database without a persistence strategy.** Redis offers RDB snapshots and AOF logs, but neither matches the durability guarantees of a transactional database — know your data-loss tolerance before relying on Redis as source of truth.
- **Unbounded key growth.** Every key without a TTL is a potential memory leak; set `maxmemory` and an eviction policy (`allkeys-lru`, `volatile-ttl`, etc.) so Redis degrades gracefully under memory pressure instead of OOM-killing.
- **`KEYS *` in production.** It's O(n) and blocks the single-threaded event loop; use `SCAN` for iteration instead.
- **Hot keys and single-threaded bottlenecks.** Redis processes commands on one thread per shard, so a single very hot key (a viral leaderboard entry, a global counter) can saturate a node even when overall cluster capacity is fine. Consider key sharding or local caching for extreme hot spots.
- **Cluster mode complexity.** Redis Cluster shards data via hash slots, which breaks multi-key operations (transactions, `MGET` across shards) unless keys share a hash tag (`{user:42}:profile`). Plan your key naming scheme for cluster mode from the start — retrofitting it is painful.
- **Cache stampede on expiry.** When a hot key expires, many concurrent requests can miss the cache simultaneously and hammer the backing store; mitigate with jittered TTLs, request coalescing, or probabilistic early expiration.
