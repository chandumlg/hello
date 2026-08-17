# Qdrant

Qdrant is an open-source vector database that solves the problem of
finding "semantically similar" items — nearest neighbors in
high-dimensional embedding space — at low latency and production scale,
something relational and document databases aren't built to do
efficiently.

## What problem it solves

Modern AI applications — RAG pipelines, semantic search, recommendation
engines, deduplication, anomaly detection — represent text, images, or
other content as dense vectors (embeddings) produced by a model. The
core operation these systems need is: "given this vector, find the K
most similar vectors out of millions or billions, fast." Doing this with
a brute-force scan is O(n) per query and falls apart past a few thousand
vectors; doing it with a traditional index (B-tree, hash index) doesn't
work because there's no notion of "similar" in those structures.

Qdrant builds an approximate nearest-neighbor (ANN) index — using HNSW
(Hierarchical Navigable Small World graphs) — that turns similarity
search into a sub-linear-time operation, while also supporting exact
metadata filtering (e.g. "similar products, but only in stock and under
$50") combined with the vector search itself, which many ANN libraries
can't do without a slow post-filter pass. It's written in Rust, ships as
a single binary or container, and exposes both a REST and gRPC API.

## Primary use cases and when to adopt it

- **Retrieval-Augmented Generation (RAG)** — storing document/chunk
  embeddings and retrieving the top-K most relevant chunks to stuff into
  an LLM prompt at query time.
- **Semantic and hybrid search** — search that matches meaning rather
  than exact keywords, often combined with traditional full-text/BM25
  scoring (hybrid search) for best results.
- **Recommendation systems** — "find items similar to what this user
  liked," using user or item embeddings.
- **Deduplication and anomaly detection** — flagging near-duplicate
  records or outliers by embedding distance.

Adopt Qdrant when you've outgrown "loop over embeddings and compute
cosine similarity in Python/numpy" (typically past a few thousand to
tens of thousands of vectors, or once query latency/concurrency
matters), and you need filtered search, horizontal scaling, or
persistence that an in-memory list doesn't give you. For a single-user
prototype or a dataset that fits comfortably in memory, a simple numpy
or FAISS in-process index may still be enough — Qdrant earns its keep
once you need it as a shared, queryable service.

## Basic usage

**1. Run it locally with Docker:**

```bash
docker run -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant
```

This exposes the REST API on `6333` and gRPC on `6334`, with data
persisted to `./qdrant_storage`.

**2. Create a collection and insert vectors (Python client):**

```python
from qdrant_client import QdrantClient
from qdrant_client.models import VectorParams, Distance, PointStruct

client = QdrantClient(url="http://localhost:6333")

client.create_collection(
    collection_name="docs",
    vectors_config=VectorParams(size=384, distance=Distance.COSINE),
)

client.upsert(
    collection_name="docs",
    points=[
        PointStruct(id=1, vector=[0.1, 0.2, ...], payload={"source": "readme.md"}),
        PointStruct(id=2, vector=[0.05, 0.9, ...], payload={"source": "faq.md"}),
    ],
)
```

**3. Query for nearest neighbors, with metadata filtering:**

```python
from qdrant_client.models import Filter, FieldCondition, MatchValue

results = client.query_points(
    collection_name="docs",
    query=[0.12, 0.18, ...],
    query_filter=Filter(
        must=[FieldCondition(key="source", match=MatchValue(value="faq.md"))]
    ),
    limit=5,
)
```

## Common pitfalls

- **Vector size and distance metric are fixed per collection.** You
  choose the embedding dimension and distance function (cosine, dot,
  Euclidean) at collection-creation time; changing embedding models
  later usually means a new collection and a full re-index, not an
  in-place migration.
- **HNSW is approximate, not exact.** Recall is tunable (`ef`, `m`
  parameters) but trades off against speed and memory — don't assume
  100% recall by default, especially on large collections with
  aggressive performance settings.
- **Payload filtering can still be slow if unindexed.** Filtering on a
  payload field without a payload index falls back to a full scan of
  matching points; for fields you filter on often, create a payload
  index explicitly.
- **Memory is the main scaling constraint.** HNSW graphs are held in
  RAM by default for query speed; large collections need either enough
  memory to hold vectors + index, or explicit configuration for
  on-disk/mmap storage and quantization (scalar/product/binary) to trade
  some accuracy for a much smaller memory footprint.
- **Client and server versions can drift.** The Python/JS clients track
  the server API version — a client too far ahead or behind the server
  can hit unsupported-feature errors, so pin compatible versions
  together, especially across upgrades.
