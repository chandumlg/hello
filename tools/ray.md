# Ray

## What it is

Ray is an open-source framework for scaling Python workloads — from a single script to a cluster of hundreds of machines — without rewriting your code around a distributed-systems API. It solves the problem of "my model training / data pipeline / hyperparameter sweep runs fine on my laptop but I need it to run on 50 GPUs by Friday" by turning ordinary Python functions and classes into distributed tasks and actors with a couple of decorators.

## Primary use cases

Ray shows up most in AI/ML infrastructure because it was built by the team that later founded Anyscale, specifically to handle the messy parallelism that shows up in ML workloads:

- **Distributed training** — Ray Train wraps PyTorch/TensorFlow/Hugging Face training loops to run across a GPU cluster with minimal code changes.
- **Hyperparameter tuning** — Ray Tune runs hundreds of trials in parallel with early-stopping schedulers (ASHA, PBT) baked in.
- **Batch inference / data preprocessing at scale** — Ray Data streams large datasets through transformation and inference pipelines without loading everything into memory.
- **Model serving** — Ray Serve deploys models as scalable, composable microservices, including multi-model pipelines (useful for agentic/RAG systems chaining several models).
- **General distributed compute** — Ray Core is the low-level primitive (tasks + actors) underneath all of the above; teams use it directly for things like distributed simulation, reinforcement learning, or parallelizing any CPU-bound Python job.

A team typically adopts Ray when they've outgrown `multiprocessing` or a single-machine GPU box, and don't want to hand-roll their own job scheduler, or drop straight into raw Kubernetes/Spark for what is fundamentally a Python-native compute problem. It's especially common in LLM training/fine-tuning and RAG-serving stacks (vLLM uses Ray internally for multi-GPU serving, for example).

## Basic usage

Install and initialize:

```bash
pip install ray
```

Turn a function into a distributed task:

```python
import ray
ray.init()  # connects to a local cluster, or a remote one via ray.init(address="ray://<head-node>:10001")

@ray.remote
def square(x):
    return x * x

futures = [square.remote(i) for i in range(10)]
results = ray.get(futures)  # [0, 1, 4, 9, ..., 81], executed in parallel
```

Stateful distributed workers with actors:

```python
@ray.remote
class Counter:
    def __init__(self):
        self.n = 0
    def increment(self):
        self.n += 1
        return self.n

c = Counter.remote()
ray.get([c.increment.remote() for _ in range(5)])  # 5, running on one worker process
```

Launch a real multi-node cluster (e.g. on AWS/GCP/Kubernetes) with the Ray cluster launcher:

```bash
ray up cluster.yaml     # spins up head + worker nodes
ray submit cluster.yaml my_script.py
```

## Common pitfalls

- **Serialization overhead.** Every argument passed into `.remote()` and every return value gets pickled and sent over the object store. Passing large objects (big NumPy arrays, whole dataframes) repeatedly kills throughput — use `ray.put()` once and pass the resulting object reference instead of re-serializing the same data on every call.
- **`ray.get()` blocks and can create false serialization.** Calling `ray.get()` inside a loop right after each `.remote()` call defeats parallelism — you're waiting on each task before launching the next. Launch all the futures first, then `ray.get()` the list.
- **The object store is memory, not infinite.** Ray spills to disk once the object store fills up, but that's slow; watch memory usage on tasks that produce large outputs, especially in Ray Data pipelines over big datasets.
- **Local `ray.init()` vs. cluster `ray.init(address=...)` confusion.** Code that "works" locally can silently run single-node instead of hitting the cluster if the address isn't set (or is stale), giving misleadingly slow benchmarks.
- **Version skew between client and cluster.** Ray is strict about the Ray (and often Python) version matching between the driver and the cluster nodes — a mismatch fails at connection time with a not-always-obvious error.
- **Actors are stateful and single-threaded by default.** If you need concurrency within one actor, you have to explicitly opt into `max_concurrency` or use async methods — a naive actor pool becomes a bottleneck if you fan out too many calls to too few actors.
