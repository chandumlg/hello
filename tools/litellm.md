# LiteLLM

LiteLLM is a unified proxy and SDK that lets you call 100+ LLM providers
(OpenAI, Anthropic, Azure, Bedrock, Vertex, Ollama, and more) through one
consistent, OpenAI-compatible interface — solving the problem of every
provider having its own SDK, auth scheme, and request/response shape.

## What problem it solves

The moment a team uses more than one LLM provider — say, Claude for one
workload and a self-hosted Llama model for another, or a fallback provider
for when the primary is rate-limited — the code has to branch on provider
SDKs, each with different auth, request formats, streaming semantics, and
error types. That coupling makes it painful to swap models, A/B test
providers, or add a fallback without touching application code throughout
the codebase.

LiteLLM normalizes all of that behind a single interface shaped like the
OpenAI SDK (`completion()` / `/v1/chat/completions`). Point it at
`anthropic/claude-opus-4-6`, `bedrock/anthropic.claude-3-sonnet`, or
`ollama/llama3` and the call site doesn't change. On top of the interface,
it ships a standalone **proxy server** that centralizes API keys, spend
tracking, rate limiting, retries/fallbacks, and caching for every team
hitting LLMs in an org — instead of each service reimplementing that
plumbing.

## Primary use cases and when a team would adopt it

- **Multi-provider abstraction** — an application or agent framework needs
  to swap models (cost, latency, capability) without rewriting call sites.
- **Centralized LLM gateway** — a platform team wants one place to hold
  provider API keys, enforce per-team budgets/rate limits, and see spend
  across the whole org, rather than keys scattered across every service.
- **Automatic fallbacks and retries** — production traffic should fail
  over to a secondary provider/model on rate limits or outages instead of
  erroring out.
- **Cost and usage observability** — teams need per-request/per-key cost
  tracking and logging (to Datadog, Langfuse, S3, etc.) across heterogeneous
  providers with inconsistent native billing APIs.

Adopt LiteLLM once you're past "call one provider's SDK directly" — either
because you need provider flexibility (self-hosted + hosted models, or
multiple hosted vendors) or because more than one team/service is calling
LLMs and you want shared governance (keys, budgets, rate limits) instead of
everyone rolling their own client.

## Basic usage

**1. Drop-in SDK replacement for provider SDKs:**

```python
from litellm import completion
import os

os.environ["ANTHROPIC_API_KEY"] = "sk-..."
os.environ["OPENAI_API_KEY"] = "sk-..."

# Same call shape regardless of provider — only the model string changes
resp = completion(
    model="claude-sonnet-5",
    messages=[{"role": "user", "content": "Explain the CAP theorem in one sentence."}],
)
print(resp.choices[0].message.content)

resp = completion(model="gpt-5", messages=[{"role": "user", "content": "Same question."}])
```

**2. Run the proxy server with a routing config:**

```yaml
# litellm_config.yaml
model_list:
  - model_name: primary
    litellm_params:
      model: anthropic/claude-sonnet-5
      api_key: os.environ/ANTHROPIC_API_KEY
  - model_name: primary
    litellm_params:
      model: bedrock/anthropic.claude-3-sonnet
      api_key: os.environ/AWS_ACCESS_KEY_ID

router_settings:
  fallbacks: [{"primary": ["primary"]}]
  num_retries: 2
```

```bash
pip install 'litellm[proxy]'
litellm --config litellm_config.yaml --port 4000
```

Any OpenAI-SDK client can now call it by changing only the base URL:

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:4000", api_key="litellm-master-key")
resp = client.chat.completions.create(
    model="primary",
    messages=[{"role": "user", "content": "Hello from the proxy."}],
)
```

**3. Set per-key spend limits and rate limits via the proxy's virtual keys:**

```bash
curl -X POST http://localhost:4000/key/generate \
  -H "Authorization: Bearer <master-key>" \
  -H "Content-Type: application/json" \
  -d '{"models": ["primary"], "max_budget": 50, "rpm_limit": 100}'
```

The returned key can be handed to a team or service; the proxy enforces
the budget and rate limit centrally, independent of any single provider's
own controls.

## Common pitfalls

- **Feature parity is not universal.** Not every provider supports every
  feature LiteLLM exposes (function calling, vision, JSON mode,
  prompt caching). Normalizing the interface doesn't normalize provider
  capability — check the provider-specific docs before assuming a param
  works everywhere.
- **Model name drift.** Provider model strings change often (deprecated
  versions, renamed SKUs); a hardcoded `model=` string in `model_list`
  can silently start failing when a provider retires a model. Pin
  versions deliberately and monitor for deprecation notices.
- **The proxy is another service to operate.** Once you centralize keys
  and routing behind the LiteLLM proxy, it becomes a single point of
  failure and a security boundary — for keys and PII in prompts. Give it
  the same production rigor (auth, TLS, HA deployment) as any other
  gateway, not "spun up on someone's laptop."
- **Cost tracking accuracy lags new models.** Per-token pricing tables are
  maintained by LiteLLM and can be stale for very recently released
  models, leading to inaccurate spend dashboards until updated — don't
  treat the cost numbers as billing-grade without cross-checking against
  the provider's actual invoice initially.
- **Fallback chains can mask real failures.** Aggressive automatic
  fallback to a secondary model on error is great for availability but can
  quietly degrade output quality (a weaker fallback model serving
  production traffic) without an obvious signal — alert on fallback rate,
  not just on hard failures.
- **Streaming and token-counting differences.** Token counts and stream
  chunk boundaries can differ subtly by provider even through the unified
  interface; if you have logic that depends on exact chunking or token
  counts, test it per-provider rather than assuming byte-for-byte parity.
