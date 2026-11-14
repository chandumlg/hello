# LlamaIndex

LlamaIndex is an open-source Python (and TypeScript) framework for connecting LLMs to your own data: it loads documents, splits and embeds them, stores them in an index, and retrieves the relevant pieces at query time so the model answers from your content instead of from memory.

## Why teams adopt it

A retrieval-augmented generation (RAG) prototype is about thirty lines of code: chunk some text, embed it, do a nearest-neighbour search, paste the hits into a prompt. The production version is where the work is. You need parsers for PDFs and HTML, a chunking strategy, incremental re-ingestion when documents change, metadata filtering, hybrid search, reranking, citations, and some way to measure whether any change made answers better. LlamaIndex gives you a named, swappable component for each of those stages, so you spend your time tuning them instead of writing the plumbing between them.

The core concepts:
- **Document**: a loaded source item (a file, a page, a database row) with text and metadata.
- **Node**: a chunk of a Document, with metadata and a link back to its source. Nodes are what get embedded and retrieved.
- **Reader**: loads Documents from a source. `SimpleDirectoryReader` handles local files, and LlamaHub has connectors for Slack, Notion, S3, databases and so on.
- **Node parser**: splits Documents into Nodes (by sentence, by token count, by Markdown heading, by code structure).
- **Index**: a structure over Nodes that supports retrieval. `VectorStoreIndex` is the one almost everyone uses. It can keep vectors in memory or delegate to an external store such as pgvector, Qdrant or OpenSearch.
- **Retriever**: given a query, returns scored Nodes.
- **Node postprocessor**: filters or reorders retrieved Nodes (similarity cutoff, reranker).
- **Response synthesizer**: turns the query plus Nodes into an LLM answer.
- **Query engine**: retriever, postprocessors and synthesizer wired together. A **chat engine** is the same with conversation history.

How it compares with its neighbours:
- **LangChain/LangGraph** is a general framework for LLM applications and agent control flow. LlamaIndex is narrower and deeper on ingestion and retrieval. Many teams use a LlamaIndex query engine as a tool inside a LangGraph agent.
- **DSPy** optimises prompts against a metric. It doesn't load or index your data.
- **A vector database** (Qdrant, pgvector) stores and searches vectors. LlamaIndex sits above it and handles everything before the write and after the search.

Common adoption triggers:
- You're building question answering or search over internal documents, tickets, runbooks or code.
- A hand-rolled RAG pipeline works but every experiment (new chunk size, new embedding model, adding a reranker) means rewriting glue code.
- You need answers with citations back to source documents.
- You want to try several vector stores or model providers without changing application code.
- Your data lives in many systems and you'd rather not write a loader for each.

## Basic usage

### 1. Index a folder and ask a question

```bash
pip install llama-index
export OPENAI_API_KEY=...
```

```python
from llama_index.core import SimpleDirectoryReader, VectorStoreIndex

documents = SimpleDirectoryReader("./docs", recursive=True).load_data()
index = VectorStoreIndex.from_documents(documents)

query_engine = index.as_query_engine(similarity_top_k=5)
response = query_engine.query("How do we rotate the database credentials?")

print(response)
for hit in response.source_nodes:
    print(f"{hit.score:.3f}  {hit.node.metadata.get('file_name')}")
```

`from_documents` chunks, embeds and indexes in one call, with everything held in memory. `source_nodes` is the part to get used to reading: it shows which chunks the answer was built from and how well they scored, and most quality problems are visible there before you look at the generated text.

### 2. Choose your own models and chunking

The `llama-index` package defaults to OpenAI for both the LLM and embeddings. Other providers are separate integration packages, and `Settings` sets the global defaults.

```bash
pip install llama-index-llms-anthropic llama-index-embeddings-huggingface
```

```python
from llama_index.core import Settings
from llama_index.core.node_parser import SentenceSplitter
from llama_index.embeddings.huggingface import HuggingFaceEmbedding
from llama_index.llms.anthropic import Anthropic

Settings.llm = Anthropic(model="claude-sonnet-5-5")
Settings.embed_model = HuggingFaceEmbedding(model_name="BAAI/bge-small-en-v1.5")
Settings.node_parser = SentenceSplitter(chunk_size=512, chunk_overlap=64)
```

Set these before building the index. The embedding model in particular is baked into the stored vectors, so the same model has to be configured at query time.

### 3. Ingest into a real vector store, incrementally

For anything beyond a demo, split ingestion from querying. An `IngestionPipeline` runs the transformations and writes to an external store. With a docstore attached, it hashes each document and skips the ones that haven't changed on the next run.

```bash
pip install llama-index-vector-stores-qdrant
```

```python
import qdrant_client
from llama_index.core import SimpleDirectoryReader, VectorStoreIndex
from llama_index.core.ingestion import IngestionPipeline
from llama_index.core.node_parser import SentenceSplitter
from llama_index.core.storage.docstore import SimpleDocumentStore
from llama_index.embeddings.huggingface import HuggingFaceEmbedding
from llama_index.vector_stores.qdrant import QdrantVectorStore

embed_model = HuggingFaceEmbedding(model_name="BAAI/bge-small-en-v1.5")
client = qdrant_client.QdrantClient(url="http://localhost:6333")
vector_store = QdrantVectorStore(client=client, collection_name="runbooks")

pipeline = IngestionPipeline(
    transformations=[SentenceSplitter(chunk_size=512, chunk_overlap=64), embed_model],
    vector_store=vector_store,
    docstore=SimpleDocumentStore(),
)

# filename_as_id gives each document a stable ID, which deduplication relies on
documents = SimpleDirectoryReader("./docs", filename_as_id=True).load_data()
pipeline.run(documents=documents)
pipeline.persist("./pipeline_state")   # call pipeline.load("./pipeline_state") before the next run

# Query side, typically a different process:
index = VectorStoreIndex.from_vector_store(vector_store, embed_model=embed_model)
retriever = index.as_retriever(similarity_top_k=10)
for hit in retriever.retrieve("credential rotation"):
    print(f"{hit.score:.3f}  {hit.node.get_content()[:80]}")
```

Calling the retriever directly, without the LLM, is the quickest way to debug retrieval. It's cheap, it's deterministic, and it tells you whether the right chunk was found at all.

## Pitfalls and things to watch out for

- **The defaults are for demos.** Out of the box you get OpenAI models, 1024-token chunks, the top 2 results, and an in-memory index that re-embeds everything on each start. Each of those is a decision you should make deliberately. `similarity_top_k=2` in particular is often too low once you have more than a handful of documents.
- **Retrieval quality is usually the problem, not the LLM.** When answers are wrong, look at `source_nodes` first. If the right chunk isn't there, no prompt change will fix it. The usual fixes are better parsing, different chunk sizes, hybrid (keyword plus vector) search, metadata filters, and a reranker over a larger candidate set.
- **Parsing sets the ceiling.** PDFs with tables, multi-column layouts and scanned pages come out of the default readers as scrambled text, and everything downstream inherits that. Inspect the parsed text of a few representative documents before tuning anything else. A dedicated parser (LlamaParse, Unstructured, Docling) is often the biggest single improvement.
- **Changing the embedding model means re-indexing everything.** Vectors from different models aren't comparable, and a mismatch between ingest and query time doesn't raise an error if the dimensions happen to match. It just returns poor results. Record the embedding model name alongside the collection.
- **Global `Settings` is convenient and hazardous.** It's process-wide mutable state. In a service with several pipelines, or in tests, pass `llm=` and `embed_model=` to components explicitly so one code path can't change another's models.
- **The API moves quickly.** The project has reorganised its packages and replaced core abstractions more than once (`ServiceContext` gave way to `Settings`, and integrations moved into separate `llama-index-*` packages), so tutorials and LLM-generated code are often out of date. Pin versions, upgrade deliberately, and check the current docs when an import fails. The agent and workflow APIs change faster than the indexing and retrieval core.
- **Abstraction hides cost and latency.** A single `query()` can make several LLM calls depending on the response mode (`refine` makes one per chunk, `tree_summarize` makes a tree of them), and some query engines and metadata extractors add more. Turn on tracing (LlamaIndex has callback and OpenTelemetry-based instrumentation, and integrates with Langfuse and similar tools) and look at what one request does.
- **Deletions and updates need a plan.** Deduplication depends on stable document IDs and a persisted docstore. Without both, re-running ingestion duplicates chunks, and documents removed from the source stay retrievable. Decide early how a deleted or permission-changed document gets removed from the index.
- **Access control isn't built in.** If documents have different audiences, store the ACL as node metadata and apply a metadata filter on every query, enforced server-side by the vector store. Don't rely on the prompt to keep restricted content out of answers.
- **Retrieved text is untrusted input.** A document in the index can contain instructions aimed at the model. Treat indexed content as a prompt-injection surface, especially if the query engine feeds an agent that can call tools.
- **Evaluate before you tune.** Build a small set of questions with known source documents and measure retrieval (hit rate, MRR) and answer quality (faithfulness, relevancy) on each change. LlamaIndex ships evaluators in `llama_index.core.evaluation`, and Promptfoo or Ragas work too. Without a fixed set, chunk-size and top-k changes are guesswork.
- **Don't reach for it when you don't need retrieval.** If the whole corpus fits comfortably in the model's context window, or the task is one fixed prompt, a direct SDK call is simpler to operate and debug.
