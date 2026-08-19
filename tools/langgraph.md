# LangGraph

LangGraph is a library for building stateful, multi-step AI agents as explicit graphs of nodes and edges, solving the problem that plain prompt-chaining code becomes unmanageable once an agent needs loops, branching, retries, or a human to step in mid-task.

## Primary use cases

LangGraph targets teams building agents that do more than a single request/response round-trip to an LLM:

- **Multi-step agents with tool use** — an agent that plans, calls tools, inspects results, and decides whether to loop back or finish, rather than a fixed chain of calls.
- **Long-running or human-in-the-loop workflows** — flows that need to pause for approval (e.g., "confirm before sending this email"), persist state, and resume later, sometimes hours or days afterward.
- **Multi-agent systems** — coordinating several specialized agents (a researcher, a coder, a reviewer) where control needs to hand off between them based on state, not just a linear pipeline.
- **Anything needing reliable state and observability** — because the graph and its state are explicit and serializable, you get checkpointing, replay, and time-travel debugging for free, which ad hoc "while loop calling an LLM" code does not give you.

A team typically reaches for it once a LangChain-style chain (or a bespoke Python loop around an LLM call) starts accumulating hand-rolled control flow — `if`/`while` logic to decide "call the tool again vs. respond vs. ask the user" — and that logic needs to be inspectable, testable, and resumable in production.

## Basic usage

Install it:

```bash
pip install langgraph
```

**1. A minimal two-node graph** — state flows through nodes, edges decide what runs next:

```python
from typing import TypedDict
from langgraph.graph import StateGraph, START, END

class State(TypedDict):
    input: str
    output: str

def uppercase(state: State) -> State:
    return {"output": state["input"].upper()}

graph = StateGraph(State)
graph.add_node("shout", uppercase)
graph.add_edge(START, "shout")
graph.add_edge("shout", END)

app = graph.compile()
print(app.invoke({"input": "hello"}))  # {'input': 'hello', 'output': 'HELLO'}
```

**2. A tool-calling agent loop with conditional routing** — the core pattern LangGraph is built for: keep calling tools until the model produces a final answer.

```python
from langgraph.graph import StateGraph, MessagesState, START, END
from langgraph.prebuilt import ToolNode
from langchain_anthropic import ChatAnthropic

def search(query: str) -> str:
    """Look up information."""
    return f"results for {query}"

llm = ChatAnthropic(model="claude-sonnet-5").bind_tools([search])

def call_model(state: MessagesState):
    return {"messages": [llm.invoke(state["messages"])]}

def should_continue(state: MessagesState):
    last = state["messages"][-1]
    return "tools" if last.tool_calls else END

graph = StateGraph(MessagesState)
graph.add_node("agent", call_model)
graph.add_node("tools", ToolNode([search]))
graph.add_edge(START, "agent")
graph.add_conditional_edges("agent", should_continue, ["tools", END])
graph.add_edge("tools", "agent")

app = graph.compile()
```

**3. Persisting state with a checkpointer** — enables pause/resume and human-in-the-loop approval:

```python
from langgraph.checkpoint.memory import InMemorySaver

app = graph.compile(checkpointer=InMemorySaver())
config = {"configurable": {"thread_id": "conversation-1"}}

app.invoke({"messages": [("user", "search for langgraph")]}, config)
# later, in a new process, resume the same thread:
app.invoke({"messages": [("user", "now search for kafka")]}, config)
```

## Common pitfalls

- **State schema sprawl.** It's tempting to dump everything (raw messages, intermediate scratch data, tool outputs) into one big `TypedDict`. Keep the state minimal and typed; a bloated shared state makes nodes implicitly coupled and hard to reason about.
- **Unbounded loops.** A conditional edge that keeps routing back into the agent node with no exit condition (or no recursion limit) will spin forever burning tokens. Always set `recursion_limit` in the config and make sure the "should I stop" edge has a real terminal condition.
- **In-memory checkpointer in production.** `InMemorySaver` is fine for local dev, but it loses all state on restart. Production deployments need a persistent checkpointer (Postgres, Redis, etc. via `langgraph-checkpoint-*` packages) or human-in-the-loop and resumability silently stop working after a deploy.
- **Treating it as a replacement for simple chains.** If a task really is a fixed, linear sequence of LLM/tool calls with no branching or looping, a plain function or a LangChain `Runnable` chain is simpler and easier to debug — reach for LangGraph when you actually need cycles, branching, or persisted state, not by default.
- **Version churn.** LangGraph's API (especially around `MessagesState`, prebuilt agents, and checkpointer packages) has moved fast across versions; pin versions and check the migration notes when upgrading, since node/edge signatures have changed between releases.
