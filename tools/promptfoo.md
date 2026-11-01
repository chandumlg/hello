# Promptfoo

Promptfoo is an open-source CLI and library for systematically evaluating and red-teaming LLM prompts, models, and agents — solving the problem that prompt and model changes are usually shipped on vibes instead of regression-tested like code.

## What problem it solves

LLM behavior is non-deterministic and sensitive to small changes: a reworded system prompt, a model upgrade, or a new retrieval strategy can silently break cases that worked yesterday. Without an eval harness, teams eyeball a few outputs in a playground and hope. Promptfoo treats prompts like code under test: you declare a set of test cases, run them against one or more prompts/providers, and assert on the outputs, getting a pass/fail matrix you can diff across versions and run in CI.

## Primary use cases and when a team would adopt it

- **Prompt regression testing** — a suite of inputs with assertions (exact match, regex, JSON schema, `contains`, cost/latency thresholds) that gates prompt changes in CI.
- **Model comparison** — run the same cases against GPT, Claude, Llama via Ollama, etc., and compare quality, latency, and cost side by side before migrating.
- **LLM-as-judge grading** — rubric-based assertions (`llm-rubric`, `model-graded-closedqa`) for subjective qualities like tone, faithfulness, or helpfulness.
- **RAG and agent evaluation** — check groundedness against retrieved context, tool-call correctness, or end-to-end workflows via custom Python/JS providers.
- **Red teaming** — generate adversarial probes (prompt injection, jailbreaks, PII leakage, harmful content) against your app and report vulnerabilities.

Adopt it when an LLM feature moves from prototype to production, when more than one person edits prompts, or when you need evidence (for security review or a model migration) that quality did not regress.

## Getting started

**1. Install and scaffold**

```bash
npm install -g promptfoo        # or: npx promptfoo@latest init
promptfoo init                  # writes promptfooconfig.yaml
export ANTHROPIC_API_KEY=...    # keys come from env vars
```

**2. Declare prompts, providers, and tests** in `promptfooconfig.yaml`:

```yaml
description: Support-ticket summarizer
prompts:
  - "Summarize this ticket in one sentence: {{ticket}}"
  - file://prompts/summarize_v2.txt
providers:
  - anthropic:messages:claude-sonnet-4-5
  - ollama:chat:llama3
defaultTest:
  options:
    provider: anthropic:messages:claude-sonnet-4-5   # grader model
tests:
  - vars:
      ticket: "Login fails with 500 after the 4.2 deploy; started 09:10 UTC."
    assert:
      - type: icontains
        value: "login"
      - type: javascript
        value: output.length < 200
      - type: llm-rubric
        value: "Mentions the deploy as a likely cause without inventing details"
      - type: latency
        threshold: 5000
  - vars: { ticket: file://fixtures/long_ticket.txt }
    assert:
      - type: not-contains
        value: "As an AI"
```

**3. Run, inspect, and gate CI**

```bash
promptfoo eval                      # run the matrix (prompts x providers x tests)
promptfoo view                      # local web UI to compare outputs
promptfoo eval -o results.json --fail-on-error
promptfoo eval --no-cache           # force fresh calls
```

In CI, run `promptfoo eval` on pull requests that touch `prompts/`; a non-zero exit code on failed assertions blocks the merge. Use `--filter-pattern` or `--filter-first-n` to keep PR runs cheap.

**4. Red team (optional)**

```bash
promptfoo redteam init              # choose target + plugins (injection, pii, harmful, ...)
promptfoo redteam run               # generate attacks, run them, score results
promptfoo redteam report
```

## Common pitfalls

- **Judge bias and drift** — `llm-rubric` graders are themselves LLMs; pin the grader model, keep rubrics narrow and specific, and prefer deterministic assertions (regex, JSON schema, code) wherever possible.
- **Flaky tests** — non-determinism means a single run is weak evidence. Set `temperature: 0` where supported, use `repeat`/thresholds, and gate on pass *rate* (`--threshold`-style) rather than every case passing.
- **Unrepresentative test sets** — a dozen hand-written cases give false confidence. Mine real production traces (e.g. from Langfuse), include edge cases, and grow the suite from every bug found.
- **Cost and rate limits** — prompts x providers x tests x repeats multiplies fast, plus judge calls. Use the cache, concurrency limits (`-j`), and subsets on PRs, with the full suite nightly.
- **Eval overfitting** — iterating prompts until the suite is green can overfit to it. Hold out a set you don't tune against.
- **Data handling** — evals send fixtures to third-party APIs and the web viewer / sharing features can upload results; scrub PII and review telemetry/sharing settings (`PROMPTFOO_DISABLE_TELEMETRY=1`) for sensitive data.
- **Red team results need triage** — generated attacks can yield false positives; treat reports as leads to verify, not a certification of safety.
