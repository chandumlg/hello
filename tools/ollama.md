# Ollama

Ollama is a local runtime and model manager for open-weight LLMs — it solves the
problem of "getting a quantized model running behind a stable API on my own
machine or server" by packaging model download, hardware-aware loading, and an
OpenAI-compatible HTTP endpoint into one binary.

## What problem it solves

Running an open model by hand means choosing a format (GGUF, safetensors),
picking a quantization, wiring up llama.cpp or similar, figuring out GPU
offload, and writing a server around it. Ollama wraps that: `ollama pull`
fetches a versioned, quantized model from a registry, a background server loads
it on demand (GPU if available, CPU/Metal otherwise), unloads it when idle, and
exposes a REST API on `localhost:11434`. Think "Docker for local models",
including a Dockerfile-like `Modelfile` for custom variants.

## Primary use cases and when to adopt

- **Local development against LLMs**: build and iterate on agents/RAG without
  API cost, rate limits, or sending data off-machine.
- **Privacy-sensitive workloads**: code, logs, or documents that cannot leave a
  laptop or VPC.
- **CI and integration tests**: a small model (e.g. a 1-3B param one) gives a
  cheap, deterministic-ish stand-in for a hosted API.
- **Embeddings locally**: serve embedding models for vector-store pipelines.
- **Evaluating models quickly**: swap `llama`, `qwen`, `mistral`, `gemma`
  variants by changing one string.
- **Small internal deployments**: single-node team tools, edge, or air-gapped
  environments.

Adopt it when you want fast, low-ceremony local or single-node inference. Reach
for vLLM, TGI, or SGLang instead when you need high-concurrency throughput,
continuous batching at scale, or multi-GPU tensor parallelism in production.

## Getting started

### 1. Install, pull, and chat

```bash
curl -fsSL https://ollama.com/install.sh | sh   # or brew install ollama / Docker
ollama serve &                                   # server on :11434 (often auto-started)
ollama pull llama3.2                             # download default quantized tag
ollama run llama3.2 "Explain write-ahead logs in two sentences."
ollama list                                      # installed models
ollama ps                                        # loaded models, memory, GPU/CPU split
```

### 2. Call it from code (native and OpenAI-compatible APIs)

```bash
curl http://localhost:11434/api/chat -d '{
  "model": "llama3.2",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": false
}'
```

```python
from openai import OpenAI
client = OpenAI(base_url="http://localhost:11434/v1", api_key="ollama")  # key ignored
r = client.chat.completions.create(
    model="llama3.2",
    messages=[{"role": "user", "content": "Summarize CAP theorem."}],
)
print(r.choices[0].message.content)
```

Because of the `/v1` compatibility layer, most frameworks (LangChain, LiteLLM,
LlamaIndex) work by only changing `base_url`.

### 3. Customize with a Modelfile

```dockerfile
# Modelfile
FROM llama3.2
PARAMETER temperature 0.2
PARAMETER num_ctx 8192
SYSTEM "You are a terse SRE assistant. Answer with commands first."
```

```bash
ollama create sre-bot -f Modelfile
ollama run sre-bot
```

## Common pitfalls

- **Default context window is small.** Ollama historically defaults to a modest
  `num_ctx` (a few thousand tokens), and silently truncates longer prompts.
  Set `num_ctx` explicitly (Modelfile or per-request `options`) and check
  the current default for your version. Larger contexts raise KV-cache memory.
- **Silent CPU spill.** If a model doesn't fit in VRAM, layers are offloaded
  to CPU and throughput collapses. Check `ollama ps` (PROCESSOR column) rather
  than assuming the GPU is used.
- **Quantization trades quality.** Default tags are usually ~4-bit. Fine for
  dev; evaluate before assuming parity with the hosted or full-precision model.
  Pin explicit tags (`llama3.2:3b-instruct-q8_0`), not `latest`, for
  reproducibility.
- **Not a high-throughput server.** Concurrency is governed by
  `OLLAMA_NUM_PARALLEL` and `OLLAMA_MAX_LOADED_MODELS`; it won't match vLLM's
  batching under load.
- **Keep-alive surprises.** Models unload after ~5 minutes idle by default
  (`OLLAMA_KEEP_ALIVE`), so first requests pay a cold-load cost.
- **Network exposure.** It binds to localhost with no authentication. If you set
  `OLLAMA_HOST=0.0.0.0`, put it behind a reverse proxy with auth/TLS.
- **Tool-calling and structured output vary by model.** Support depends on the
  model's training and chat template; test your agent's function-calling
  against the exact model you deploy.
- **Check licenses.** Registry models carry their own licenses (Llama, Gemma,
  etc.) that may restrict commercial use.
