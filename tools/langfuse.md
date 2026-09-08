# Langfuse

## What it is

Langfuse is an open-source observability and evaluation platform for LLM applications — it solves the problem that once an LLM app leaves a notebook, you can no longer see what actually happened inside a request: which prompt version ran, what the retrieved context was, how many tokens and dollars it cost, and why the model produced a bad answer for one user but not another.

## Primary use cases

- **Tracing** — every call to an LLM (and the surrounding chain: retrieval, tool calls, sub-agent hops) is captured as a nested trace with full inputs/outputs, latency, and token usage, so a multi-step agent run can be inspected step by step instead of guessed at from logs.
- **Prompt management** — prompts are versioned and stored server-side rather than hardcoded in the app, so a prompt tweak can be pushed and A/B tested without a redeploy, and every trace links back to the exact prompt version that produced it.
- **Evaluation** — traces can be scored automatically (via LLM-as-judge, custom functions, or user feedback like thumbs up/down) or manually annotated, giving a running quality signal instead of relying on spot-checking outputs.
- **Cost and latency monitoring** — token usage and spend roll up by model, user, or feature, which is usually the first concrete number a team needs when an LLM feature's cloud bill shows up.

A team adopts Langfuse once an LLM feature has real users and "why did it say that?" stops being answerable by reading application logs — typically alongside a framework like LangChain/LangGraph or a raw SDK, once debugging production LLM behavior becomes a recurring need rather than a one-off.

## Basic usage

**1. Run it (Langfuse Cloud or self-hosted) and get API keys**

```bash
# self-hosted via Docker Compose
git clone https://github.com/langfuse/langfuse.git
cd langfuse && docker compose up -d
# or just sign up at cloud.langfuse.com and grab a public/secret key pair
```

**2. Instrument a call with the Python SDK**

```python
from langfuse.decorators import observe
from langfuse.openai import openai  # drop-in wrapper around the OpenAI client

@observe()
def answer_question(question: str) -> str:
    response = openai.chat.completions.create(
        model="gpt-4o-mini",
        messages=[{"role": "user", "content": question}],
    )
    return response.choices[0].message.content

answer_question("What's the capital of France?")
# a full trace — prompt, completion, tokens, latency, cost — now appears in the UI
```

**3. Score a trace and fetch a managed prompt**

```python
from langfuse import Langfuse

langfuse = Langfuse()

# pull a versioned prompt instead of hardcoding it
prompt = langfuse.get_prompt("support-agent-system")
compiled = prompt.compile(user_name="Alex")

# attach a quality signal after the fact (e.g. from user feedback or a judge model)
langfuse.score(trace_id="<trace_id>", name="helpfulness", value=1)
```

## Common pitfalls

- **Tracing is opt-in per call site.** Unwrapped SDK calls or `@observe`-less functions produce no trace at all — it's easy to instrument the happy path and end up blind on error-handling branches or a secondary model call buried in a utility function.
- **Sensitive data flows straight into traces.** Full prompts and completions are stored by default, which means PII, secrets, or customer data end up in the observability backend unless you configure masking/redaction — decide this before shipping, not after an audit asks about it.
- **Self-hosting is more than one container.** The stack needs Postgres, ClickHouse, Redis, and object storage behind it; the single `docker compose up` demo setup is not a production topology and needs real sizing and backups for meaningful trace volume.
- **Async flushing can silently drop traces.** The SDK batches and flushes in the background; short-lived scripts (serverless functions, CLI tools, CI jobs) that exit immediately after the LLM call need an explicit `langfuse.flush()` or traces never make it out before the process dies.
- **Evaluation scores are only as good as the judge.** LLM-as-judge scoring inherits the judge model's own blind spots and inconsistency — treat automated scores as a triage signal to prioritize manual review, not as ground truth on their own.
