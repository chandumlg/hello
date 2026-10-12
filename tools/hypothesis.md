# Hypothesis

Hypothesis is a property-based testing library for Python that generates hundreds of structured, adversarial inputs for your code, checks that stated invariants hold, and automatically shrinks any failure to a minimal reproducing example.

## Primary use cases

- **Parsers, serializers and codecs**: assert round-trip properties (`decode(encode(x)) == x`) across inputs no human would write by hand: empty strings, odd Unicode, huge ints, NaNs.
- **Data-structure and algorithm invariants**: sorting stays sorted, a cache never exceeds capacity, a merge is associative, idempotent operations really are idempotent.
- **Differential testing**: compare a fast or rewritten implementation against a slow, obviously-correct reference over the same generated inputs. This is very useful during migrations and optimizations.
- **Stateful systems**: `RuleBasedStateMachine` generates random sequences of operations (create, update, delete, retry) against a model, catching bugs that only appear after specific interleavings, such as in APIs, state machines and queues.
- **Schema-driven API fuzzing**: tools built on Hypothesis (e.g. Schemathesis) generate requests from an OpenAPI spec and check for 5xx responses and contract violations.

A team usually adopts it after an incident caused by an input nobody thought to test, or when example-based suites plateau on code with a large input space (money, time zones, text, serialization). It complements unit tests rather than replacing them: keep examples for documented behavior and add properties for invariants.

## Basic usage

**1. Install and write a first property:**

```bash
pip install hypothesis pytest
```

```python
from hypothesis import given, strategies as st

@given(st.lists(st.integers()))
def test_sort_is_idempotent_and_ordered(xs):
    once = sorted(xs)
    assert sorted(once) == once
    assert all(a <= b for a, b in zip(once, once[1:]))
```

Run it with plain `pytest`. On failure, Hypothesis prints the shrunk counterexample, and it stores it in the `.hypothesis/` example database so later runs replay it first.

**2. Build custom strategies and pin regressions:**

```python
from hypothesis import given, example, strategies as st
from dataclasses import dataclass

@dataclass
class Money:
    cents: int
    currency: str

money = st.builds(
    Money,
    cents=st.integers(min_value=0, max_value=10**12),
    currency=st.sampled_from(["USD", "EUR", "INR"]),
)

@given(money)
@example(Money(0, "USD"))          # always run this explicit case too
def test_money_roundtrip(m):
    assert parse_money(format_money(m)) == m
```

**3. Stateful testing against a simple model:**

```python
from hypothesis.stateful import RuleBasedStateMachine, rule, invariant
from hypothesis import strategies as st

class CacheMachine(RuleBasedStateMachine):
    def __init__(self):
        super().__init__()
        self.cache = LRUCache(capacity=3)
        self.model = {}

    @rule(k=st.integers(0, 5), v=st.integers())
    def put(self, k, v):
        self.cache.put(k, v)
        self.model[k] = v

    @rule(k=st.integers(0, 5))
    def get(self, k):
        got = self.cache.get(k)
        assert got is None or got == self.model[k]

    @invariant()
    def within_capacity(self):
        assert len(self.cache) <= 3

TestCache = CacheMachine.TestCase
```

Use profiles in CI to trade speed for depth:

```python
from hypothesis import settings
settings.register_profile("ci", max_examples=1000, deadline=None)
settings.load_profile("ci")   # e.g. via HYPOTHESIS_PROFILE env in conftest.py
```

## Common pitfalls

- **Weak properties**: if the assertion just re-implements the code under test, the test proves nothing. Prefer round-trips, invariants, oracles and "doesn't crash" checks.
- **Over-filtering**: `assume()` and `.filter()` that reject most inputs trigger `FailedHealthCheck`. Build valid data constructively with `st.builds`, `st.composite` and `flatmap` instead.
- **Flaky-looking failures**: Hypothesis flags non-deterministic tests (`Flaky`) and the `deadline` health check (default 200ms per example). Fix hidden state, time or randomness in the test, and only raise or disable the deadline deliberately.
- **Side effects between examples**: each example runs the test function many times, so shared databases, files or module globals leak state. Reset in the test body. Function-scoped pytest fixtures run only once per test, not per example, which triggers a health check.
- **Ignoring the example database in CI**: `.hypothesis/` is local, so a CI-found failure may not reproduce on a laptop. Copy the printed `@reproduce_failure` blob or `@example(...)`, or persist the DB as a CI cache. Commit the minimal case as an explicit `@example`.
- **Slow I/O in properties**: hundreds of runs against a real DB or network add up fast. Test pure logic with properties and keep I/O-bound checks small, or use fewer `max_examples` there.
- **Unbounded strategies**: unconstrained `st.text()` or `st.integers()` can produce values your domain forbids, leading to false failures. Encode real constraints in the strategy, not in `try/except`.
