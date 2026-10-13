# pgvector

pgvector is a PostgreSQL extension that adds a native `vector` column type plus exact and approximate nearest-neighbor search, so you can store embeddings next to your relational data and run similarity queries in plain SQL instead of operating a separate vector database.

## Why teams adopt it

Most AI features (RAG, semantic search, recommendations, deduplication) need "find the K most similar embeddings," but also need filters, joins, transactions, and backups on the surrounding data. pgvector lets the embedding live in the same row as the document, tenant ID, ACLs, and timestamps, which removes a whole class of consistency problems (dual writes, stale indexes, permission leaks across stores).

Common adoption triggers:
- You already run Postgres (RDS, Cloud SQL, Aurora, Supabase, Neon, self-hosted) and want RAG without a new system to operate, secure, and pay for.
- Your queries combine similarity with relational predicates (`WHERE tenant_id = ... AND created_at > ...`) or joins.
- Your corpus is in the thousands to low tens of millions of vectors, where a single well-tuned Postgres node is enough.
- You want ACID semantics, point-in-time recovery, and existing tooling (migrations, replicas, pg_dump) to cover your vectors too.

Teams outgrow it when they need billions of vectors, very high QPS on a single index, or specialized features (sharded ANN, built-in hybrid ranking); at that point Qdrant, Milvus, or similar become attractive.

## Basic usage

**1. Enable the extension and create a table:**
```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE documents (
  id         bigserial PRIMARY KEY,
  tenant_id  int NOT NULL,
  body       text,
  embedding  vector(1536)   -- dimension must match your embedding model
);
```

**2. Insert embeddings and run a nearest-neighbor query:**
```sql
INSERT INTO documents (tenant_id, body, embedding)
VALUES (1, 'hello world', '[0.01, -0.02, ...]');

-- <=> cosine distance, <-> L2 distance, <#> negative inner product
SELECT id, body, embedding <=> $1 AS distance
FROM documents
WHERE tenant_id = 1
ORDER BY embedding <=> $1
LIMIT 5;
```

**3. Add an HNSW index for approximate search, and tune recall at query time:**
```sql
CREATE INDEX ON documents
  USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);

SET hnsw.ef_search = 100;  -- higher = better recall, slower queries
```
Use `ivfflat` instead if build time or memory matter more than query speed/recall; it requires data to exist before building and a `lists` setting (roughly rows/1000 up to 1M rows).

## Pitfalls to watch for

- **Operator/index mismatch:** the index operator class (`vector_cosine_ops`, `vector_l2_ops`, `vector_ip_ops`) must match the distance operator in your query, or Postgres silently falls back to a sequential scan. Confirm with `EXPLAIN`.
- **Filtered ANN recall:** with an approximate index, `WHERE` filters are applied after the index scan returns `ef_search` candidates, so selective filters can return fewer than `LIMIT` rows. Raise `ef_search`, use partial indexes or partitioning per tenant, or enable iterative scans (available in recent versions).
- **Index build cost:** HNSW builds are memory- and time-hungry. Raise `maintenance_work_mem` and `max_parallel_maintenance_workers`, and build with `CREATE INDEX CONCURRENTLY` on live tables.
- **Dimension and storage limits:** `vector` indexes are limited to 2,000 dimensions; use `halfvec` (up to 4,000) or binary quantization for larger or cheaper embeddings. Vectors also bloat rows and TOAST, so mind table size and cache hit rate.
- **Changing models means re-embedding:** embeddings from different models are not comparable. Store the model name/version in a column and plan backfills.
- **Don't skip measuring recall:** compare ANN results against an exact scan (disable the index) on a sample to validate your `ef_search`/`lists` settings rather than guessing.
- **Memory fit:** HNSW performs best when the index fits in RAM (`shared_buffers` plus OS cache); size instances accordingly.
