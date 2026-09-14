# DSPy

## What it is

DSPy is a Python framework from Stanford NLP for building LLM applications as
**programs instead of hand-tuned prompt strings** — it solves the problem of
prompt engineering being brittle and non-portable: a prompt tuned for GPT-4
often falls apart on Claude or Llama, and every small change to a pipeline
means re-tweaking wording by hand until it works again.

## Primary use cases

DSPy's core idea is separation of concerns: you declare *what* each step of
your LLM pipeline should do (its inputs, outputs, and the task), and DSPy's
**optimizers** ("compilers") figure out *how* to prompt the model to do it —
generating and refining few-shot examples and instructions automatically,
using a metric and a small training/validation set you provide.

Teams reach for it when:

- **Prompt pipelines have grown multi-step** (retrieve → rerank → generate →
  verify) and hand-editing prompts for every step has become unmanageable.
- **They need to swap models** (e.g., move from a hosted frontier model to a
  cheaper self-hosted one) without re-deriving prompts from scratch — DSPy
  re-optimizes the prompts/few-shot examples for the new model instead.
- **They want measurable, repeatable improvement** — a metric-driven "compile
  step" that can be re-run in CI, instead of prompt tweaks that are never
  validated against a held-out set.
- **RAG, agents, and multi-hop reasoning pipelines** where structured
  intermediate steps (query rewriting, tool calls, self-critique) benefit
  from typed signatures rather than one giant freeform prompt.

It's a poor fit for a single one-off prompt call where there's nothing to
optimize against — plain API calls are simpler there.

## Basic usage

**1. Install and configure a model:**

```bash
pip install dspy
```

```python
import dspy

lm = dspy.LM("openai/gpt-4o-mini", api_key="...")
dspy.configure(lm=lm)
```

**2. Define a task with a Signature and a Module:**

```python
class AnswerQuestion(dspy.Signature):
    """Answer the question using the given context."""
    context: str = dspy.InputField()
    question: str = dspy.InputField()
    answer: str = dspy.OutputField()

qa = dspy.ChainOfThought(AnswerQuestion)

result = qa(context="DSPy was released by Stanford NLP.",
            question="Who released DSPy?")
print(result.answer)
```

`dspy.ChainOfThought` (or `dspy.Predict`, `dspy.ReAct`, etc.) turns the
signature into a working prompt automatically — you never write the prompt
text yourself.

**3. Optimize the pipeline against a metric:**

```python
def exact_match(example, pred, trace=None):
    return example.answer.lower() == pred.answer.lower()

optimizer = dspy.MIPROv2(metric=exact_match, auto="light")
compiled_qa = optimizer.compile(qa, trainset=train_examples)
```

`compiled_qa` now has optimizer-selected instructions and few-shot examples
baked in — call it exactly like `qa`, but with measurably better accuracy on
your metric.

## Pitfalls to watch for

- **Optimization needs real examples and a real metric.** Without a
  representative training/validation set and a meaningful metric function,
  the optimizer has nothing to optimize against — you'll get generic prompts
  and no measurable improvement over hand-written ones.
- **Compilation costs tokens and money.** Optimizers like `MIPROv2` or
  `BootstrapFewShot` make many LLM calls during compilation (proposing,
  testing, and scoring candidate prompts) — budget for this, especially
  against paid APIs, and use cheaper/smaller models for the optimization
  loop where possible.
- **Compiled artifacts are tied to the model they were compiled against.**
  Swapping the underlying LM after compiling usually means re-running the
  optimizer — a prompt/few-shot set tuned for one model doesn't reliably
  transfer to another.
- **Debuggability is different from raw prompting.** Since DSPy generates the
  actual prompt text, you have to inspect the compiled program
  (`dspy.inspect_history()`) to see what's really being sent to the model —
  teams used to hand-crafting prompts can find this indirection disorienting
  at first.
- **It's still a fast-moving library.** APIs (especially around optimizers
  and signatures) have changed significantly between versions — pin your
  version and check the changelog before upgrading in production.
