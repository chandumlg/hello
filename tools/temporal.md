# Temporal

Temporal is an open-source durable execution platform that lets you write long-running, stateful business logic as plain code, while the platform guarantees it keeps running correctly across crashes, deploys, and infrastructure failures — without you hand-rolling retries, timers, or state machines.

## Why teams adopt it

Staff and platform engineers reach for Temporal when a process spans more than a single request/response cycle: it needs to wait hours or days, call multiple unreliable external services, retry failed steps without losing progress, or coordinate work across several microservices. Instead of persisting state to a database and building a bespoke state machine plus a cron job to resume stuck work, you write a "workflow" as an ordinary function; Temporal's server records every step (a durable "event history") so that if your process crashes mid-execution, it resumes exactly where it left off — no lost state, no duplicated side effects.

Common adoption triggers:
- Multi-step business processes (order fulfillment, user onboarding, payment reconciliation, infrastructure provisioning) that need to survive service restarts and can take minutes to weeks to complete.
- AI/ML and agent systems that chain multiple LLM calls, tool invocations, and human-in-the-loop approval steps, where any single call can fail or take a long time and you need durable, resumable orchestration rather than an in-memory loop.
- Replacing a tangle of cron jobs, message queues, and manually-tracked "job status" database columns with a single system that handles retries, backoff, timeouts, and cancellation natively.
- Coordinating distributed transactions/sagas across microservices where you need compensating actions if a later step fails.

## Basic usage

**1. Define a workflow and an activity (Python SDK):**
```python
from temporalio import workflow, activity
from datetime import timedelta

@activity.defn
async def charge_credit_card(order_id: str) -> str:
    # calls an external, potentially flaky payment API
    return f"charged-{order_id}"

@workflow.defn
class OrderWorkflow:
    @workflow.run
    async def run(self, order_id: str) -> str:
        return await workflow.execute_activity(
            charge_credit_card,
            order_id,
            start_to_close_timeout=timedelta(seconds=30),
        )
```

**2. Run a worker that executes workflows/activities, and start a local dev server:**
```bash
# Spin up a local Temporal server + Web UI for development
temporal server start-dev

# In your application, run a worker process that polls a task queue
python worker.py
```

**3. Kick off a workflow execution from your application code:**
```python
from temporalio.client import Client

async def main():
    client = await Client.connect("localhost:7233")
    result = await client.execute_workflow(
        OrderWorkflow.run,
        "order-123",
        id="order-workflow-order-123",
        task_queue="orders-task-queue",
    )
    print(result)
```

## Pitfalls to watch out for

- **Workflow code must be deterministic.** Temporal replays workflow history to reconstruct state, so workflow functions can't call `random()`, read the system clock, make network calls, or do anything non-deterministic directly — that logic belongs in activities. Violating this causes replay failures that are confusing the first time you hit them.
- **It's not a message queue.** Temporal is built for durable, stateful orchestration with retries and history, not high-throughput pub/sub; using it as a generic event bus is the wrong tool for the job and gets expensive.
- **Event history has limits.** Very long-running or high-iteration workflows (e.g., a tight loop running millions of times) can bloat history size and hit server limits — use `continue-as-new` to periodically reset history for long-lived workflows.
- **Versioning workflow code is tricky.** Deploying a code change to a workflow definition while instances are still in-flight can break determinism for those in-progress executions; Temporal provides patching APIs (`workflow.patched`) for this, but it requires discipline and planning.
- **You still run infrastructure.** Self-hosting Temporal means operating the Temporal server, a database (Cassandra/Postgres/MySQL), and visibility store — many teams instead use Temporal Cloud to avoid this operational burden.
