# Instructor

Instructor is a Python library (with ports in TypeScript, Go, Ruby, Elixir, and Rust) that makes LLMs return **validated, typed objects** instead of free-form text: you declare the shape you want as a Pydantic model, and Instructor handles the schema prompting, tool/JSON-mode plumbing, parsing, validation, and automatic retries when the model gets it wrong.

## Primary use cases and when to adopt it

Most production LLM work is not chat — it's *extraction and classification* feeding into ordinary code: pull fields out of an invoice, tag a support ticket, turn a user's sentence into a query plan, decide which tool to call. That code needs a `dict` with the right keys and types, not prose. Hand-rolling this ("respond ONLY in JSON…", `json.loads`, try/except, re-prompt) is the same 80 lines in every repo, and it's subtly fragile.

Instructor's core idea is small: patch the provider client so `create()` accepts a `response_model=` argument. Under the hood it converts the Pydantic model into a JSON schema, uses the provider's best structured-output mechanism (tool calling, JSON mode, or native structured outputs, selected via a `Mode`), parses the response into your model, and — if Pydantic validation fails — sends the validation error back to the model and asks it to fix its answer, up to `max_retries` times.

Teams typically adopt it when:

- **LLM output feeds a pipeline or database.** Entity extraction, document parsing, ETL enrichment, labeling at scale — anywhere a malformed response would otherwise crash a job or poison a table.
- **You want business rules enforced on model output.** Because it's just Pydantic, you get `Field(ge=0)`, enums/`Literal`s, regex constraints, and arbitrary `@field_validator` logic — and violations become retry feedback rather than exceptions in prod.
- **You're multi-provider or expect to switch.** The same `response_model` code works across OpenAI, Anthropic, Gemini, Mistral, Cohere, Groq, Ollama/vLLM (OpenAI-compatible), LiteLLM, and others; only the client construction changes.
- **You want structure without adopting a framework.** Instructor is deliberately thin — no chains, agents, or vector-store abstractions. It sits comfortably inside LangGraph nodes, DSPy-free codebases, FastAPI handlers, or plain scripts.

Reach for something else if you need **token-level guaranteed** grammar conformance on a self-hosted model (look at constrained decoding: Outlines, XGrammar, vLLM guided decoding), or if you want a full agent runtime (Pydantic AI, LangGraph). Note also that many providers now offer native "structured outputs"; Instructor still adds value on top via validators, retries, streaming partials, and a uniform API across vendors.

## Getting started

```bash
uv add instructor            # or: pip install instructor
uv add "instructor[anthropic]"  # extras pull in provider SDKs
```

**1. Basic extraction into a typed object**

```python
import instructor
from pydantic import BaseModel, Field
from typing import Literal

# Provider-agnostic constructor ("provider/model"); reads the usual API key env vars.
client = instructor.from_provider("openai/gpt-4o-mini")
# Equivalent classic form:  instructor.from_openai(openai.OpenAI())

class Ticket(BaseModel):
    summary: str = Field(description="One-sentence summary of the issue")
    severity: Literal["low", "medium", "high", "critical"]
    component: Literal["billing", "auth", "api", "ui", "other"]
    customer_impacted: bool

ticket = client.chat.completions.create(
    response_model=Ticket,
    messages=[{"role": "user", "content":
        "Since this morning's deploy nobody on our team can log in via SSO, "
        "we're locked out of the dashboard entirely."}],
)

print(ticket.severity, ticket.component)   # e.g. "critical" "auth"
print(type(ticket))                         # <class '__main__.Ticket'> — not a dict
```

Field descriptions and type names are sent to the model as part of the schema, so they are effectively prompt engineering — name and describe fields the way you'd brief a human.

**2. Validators as self-correcting guardrails**

```python
from pydantic import BaseModel, field_validator, model_validator

class LineItem(BaseModel):
    description: str
    quantity: int = Field(gt=0)
    unit_price: float = Field(ge=0)

class Invoice(BaseModel):
    vendor: str
    items: list[LineItem]
    total: float

    @model_validator(mode="after")
    def total_matches_items(self):
        computed = round(sum(i.quantity * i.unit_price for i in self.items), 2)
        if abs(computed - self.total) > 0.01:
            # This message is fed back to the LLM on retry.
            raise ValueError(f"total {self.total} != sum of line items {computed}; re-read the document")
        return self

invoice = client.chat.completions.create(
    response_model=Invoice,
    max_retries=3,                       # re-ask with the validation error up to 3 times
    messages=[{"role": "user", "content": invoice_text}],
)
```

If all retries fail, Instructor raises `instructor.exceptions.InstructorRetryException`, which carries the attempt count and the last completion — catch it and route the record to a dead-letter queue or human review rather than letting a batch job die.

**3. Lists, streaming partials, and swapping providers**

```python
from typing import Iterable

class Person(BaseModel):
    name: str
    role: str

anthropic_client = instructor.from_provider("anthropic/claude-sonnet-4-5")  # pick your current model

# Extract many objects; yields each one as soon as it's complete.
for person in anthropic_client.chat.completions.create_iterable(
    response_model=Person,
    max_tokens=1024,
    messages=[{"role": "user", "content": meeting_notes}],
):
    print(person)

# Stream a single object as it fills in — useful for UIs.
for partial in client.chat.completions.create_partial(
    response_model=Ticket,
    messages=[{"role": "user", "content": "..."}],
):
    print(partial.model_dump(exclude_none=True))
```

For async services use `instructor.from_provider("openai/gpt-4o-mini", async_client=True)` (or wrap `AsyncOpenAI`) and `await client.chat.completions.create(...)`. To get token usage alongside the object, call `create_with_completion(...)`, which returns `(model_instance, raw_completion)`.

## Common pitfalls

- **Retries multiply cost and latency.** Each failed validation is a full additional LLM call with a longer context (the original prompt, the bad answer, and the error). A tight validator on a weak model can quietly 3x your bill. Log retry counts (Instructor exposes hooks such as `completion:kwargs`, `parse:error`, and `completion:response` via `client.on(...)`) and alert on them.
- **Validation proves shape, not truth.** A perfectly typed `Invoice` can still contain a hallucinated vendor. Use validators for cross-field consistency checks where you can, add explicit "unknown"/`Optional` escape hatches so the model isn't forced to invent values, and evaluate accuracy separately (e.g. promptfoo, Langfuse datasets).
- **Make "I don't know" representable.** If every field is required, the model *will* fill every field. Use `Optional[...] = None`, an `"other"` enum member, or a `confidence` field — otherwise you're incentivizing fabrication.
- **Mode mismatches.** Not every model supports every mode (tools vs. JSON vs. native structured outputs), especially local or OpenAI-compatible servers. If you see parse errors with Ollama/vLLM, explicitly set `mode=instructor.Mode.JSON` (or `MD_JSON`) and use a model that's decent at following schemas.
- **Schema complexity hurts quality.** Deeply nested models, huge enums, and recursive types inflate the prompt and increase failure rates. Prefer flat models, split hard extractions into multiple calls, and keep field descriptions short and specific.
- **Native structured-output limits leak through.** Some providers' strict schema modes reject certain JSON-schema features (e.g. defaults, `additionalProperties`, some unions). If a model works in one mode and errors in another, that's usually why.
- **Don't validate side effects inside validators.** Validators may run several times per request due to retries; keep them pure (no DB writes, no external API calls that aren't idempotent).
- **Pin the version.** Instructor moves quickly (provider integrations, `from_provider`, hooks, and mode names have evolved across 1.x releases). Pin it in your lockfile and read the changelog before upgrading.
