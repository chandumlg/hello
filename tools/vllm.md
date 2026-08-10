# vLLM

vLLM is a high-throughput, memory-efficient inference and serving engine for
large language models — it solves the problem of GPU memory waste and low
throughput that plagues naive LLM serving, letting you serve more concurrent
requests on the same hardware.

## What problem it solves

Running an LLM for inference sounds simple — load the weights, call
`generate()` — but doing it at production scale is not. Naive serving
(e.g. plain HuggingFace `transformers` generation) allocates a fixed,
worst-case chunk of GPU memory for each request's KV cache (the growing
set of attention key/value tensors accumulated per token), which wastes
huge amounts of memory on padding and fragmentation. That means fewer
concurrent users per GPU and higher latency under load.

vLLM's core contribution is **PagedAttention**, an algorithm that manages
the KV cache the way an OS manages virtual memory — in fixed-size, non-
contiguous "pages" — eliminating fragmentation and allowing memory to be
shared across requests (e.g. for beam search or shared prompt prefixes).
The practical result is 2-24x higher throughput than vanilla
`transformers`-based serving with the same latency budget.

## Primary use cases and when to adopt it

- **Self-hosting open-weight models** (Llama, Qwen, Mistral, DeepSeek,
  etc.) behind an OpenAI-compatible API instead of paying per-token to a
  hosted provider, for cost, data-residency, or customization reasons.
- **High-concurrency inference services** where many users hit the same
  model simultaneously — vLLM's continuous batching keeps GPU utilization
  high instead of processing requests one at a time or in static batches.
- **Multi-LoRA serving** — hosting many fine-tuned adapters on top of one
  base model and swapping them per-request without reloading weights.
- **Structured/constrained generation** (JSON mode, grammars) and
  speculative decoding for latency-sensitive applications.

Adopt vLLM when you're moving from "call the OpenAI API" or "prototype
with `transformers`" to actually operating your own inference
infrastructure — typically once cost, latency SLAs, or data control become
real constraints. It's overkill for a single-user local experiment; that's
what `ollama` or `llama.cpp` are for.

## Basic usage

**1. Install and run an OpenAI-compatible server:**

```bash
pip install vllm

vllm serve meta-llama/Llama-3.1-8B-Instruct \
  --port 8000 \
  --max-model-len 8192
```

This starts an HTTP server exposing `/v1/chat/completions`, so any
OpenAI-SDK client works against it by just changing the base URL:

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:8000/v1", api_key="not-needed")

resp = client.chat.completions.create(
    model="meta-llama/Llama-3.1-8B-Instruct",
    messages=[{"role": "user", "content": "Explain PagedAttention in one sentence."}],
)
print(resp.choices[0].message.content)
```

**2. Offline batch inference (no server needed):**

```python
from vllm import LLM, SamplingParams

llm = LLM(model="meta-llama/Llama-3.1-8B-Instruct")
params = SamplingParams(temperature=0.7, max_tokens=200)

outputs = llm.generate(["Summarize the plot of Dune.", "What is PagedAttention?"], params)
for out in outputs:
    print(out.outputs[0].text)
```

**3. Serving multiple LoRA adapters on one base model:**

```bash
vllm serve meta-llama/Llama-3.1-8B-Instruct \
  --enable-lora \
  --lora-modules support-bot=/path/to/lora-adapter-1 sql-writer=/path/to/lora-adapter-2
```

Clients then pick the adapter by name in the `model` field of the
request, avoiding the cost of hosting N full copies of the base model.

## Common pitfalls

- **GPU memory sizing is not automatic-safe.** `--gpu-memory-utilization`
  defaults to 0.9 (90% of VRAM); on a shared GPU or alongside other
  processes this can OOM the box. Tune it down explicitly in shared
  environments.
- **`--max-model-len` vs. actual context needs.** Setting this too high
  pre-allocates KV cache pages for a context length you rarely use,
  shrinking effective concurrency. Set it to what your workload actually
  needs, not the model's theoretical max.
- **Cold start cost.** Loading large models (especially with tensor
  parallelism across multiple GPUs) can take minutes; this matters for
  autoscaling policies — don't scale-to-zero aggressively unless you can
  tolerate multi-minute cold starts.
- **Version/model compatibility churn.** New model architectures often
  need a vLLM release to support them; pin your vLLM version per model
  and check the "supported models" list before assuming day-one support
  for a brand-new release.
- **Quantization tradeoffs.** AWQ/GPTQ/FP8 quantized weights reduce
  memory and increase throughput but can shift output quality — validate
  eval metrics after switching quantization, don't assume parity with the
  full-precision model.
- **Tensor vs. pipeline parallelism confusion.** `--tensor-parallel-size`
  splits each layer across GPUs (needs fast NVLink/interconnect);
  `--pipeline-parallel-size` splits layers across GPUs (more tolerant of
  slower interconnects). Picking the wrong one for your hardware topology
  can tank throughput.
