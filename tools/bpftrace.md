# bpftrace

bpftrace is a high-level tracing language for Linux that compiles short awk-style scripts into eBPF programs, letting you ask live questions of a running kernel or process ("who is calling `fsync` and how slow is it?") without recompiling, restarting, or attaching a debugger.

## Primary use cases

- **Production latency forensics**: histogram block-I/O, syscall, scheduler run-queue, or TCP retransmit latency on a live host to find which layer is actually slow, with low overhead since aggregation happens in-kernel.
- **Attributing resource use**: which process, container, or stack is generating disk writes, page faults, context switches, or outbound connections? Answers questions metrics dashboards can't, because they lack per-event detail.
- **Dynamic instrumentation of user-space**: attach uprobes to functions in Go, C, Rust, or the JVM (e.g. `malloc`, an OpenSSL call, a hot function in your service) to count calls, measure duration, or read arguments, with no code change.
- **Debugging "works on my machine" kernel behavior**: confirm which syscalls a process makes, why `connect()` is returning `ETIMEDOUT`, or where a container hits cgroup throttling.
- **Prototyping before building a tool**: bpftrace is the quick sketch; once a probe proves valuable, teams graduate it to libbpf/BCC or fold it into an always-on agent (Parca, Pyroscope, Cilium/Tetragon).

Teams adopt it when they have hit an incident where logs, metrics and traces all looked fine but the system was still slow, and someone said "I need to see what the kernel is doing." It is typically owned by platform/SRE engineers and used ad hoc on production nodes, rather than deployed as a permanent service.

## Basic usage

**1. Install and run a one-liner (requires root, kernel ≥ 4.9, ideally ≥ 5.x with BTF)**

```bash
sudo apt install bpftrace      # or: dnf install bpftrace

# Count syscalls by process name, Ctrl-C to print
sudo bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }'

# List available probes
sudo bpftrace -l 'tracepoint:syscalls:sys_enter_*' | head
```

**2. Latency histogram: how long does `read()` take, per process?**

```bash
sudo bpftrace -e '
tracepoint:syscalls:sys_enter_read { @start[tid] = nsecs; }
tracepoint:syscalls:sys_exit_read /@start[tid]/ {
  @us[comm] = hist((nsecs - @start[tid]) / 1000);
  delete(@start[tid]);
}'
```

This stores a timestamp at entry, computes the delta at exit, and keeps a power-of-two histogram per command, all in kernel maps. The `/@start[tid]/` filter skips exits whose entry was missed.

**3. Trace a user-space function with a uprobe**

```bash
# Who is allocating large blocks in my service? (pid 1234)
sudo bpftrace -p 1234 -e '
uprobe:/lib/x86_64-linux-gnu/libc.so.6:malloc /arg0 > 1048576/ {
  printf("%s big malloc %d bytes\n%s\n", comm, arg0, ustack);
}'
```

Save larger scripts as `.bt` files and run with `sudo bpftrace script.bt`. Many ready-made ones ship in the `tools/` directory of the bpftrace repo (`biolatency.bt`, `tcpretrans.bt`, `opensnoop.bt`).

## Common pitfalls

- **Root and kernel requirements**: needs `CAP_BPF`/`CAP_PERFMON` or root, and a kernel with eBPF and (for portable struct access) BTF. In Kubernetes you need a privileged pod or a debug node shell, which is a security review item.
- **Overhead is not zero**: tracing very hot events (`sched_switch`, every `kmalloc`, per-packet probes) or printing per event can slow production. Aggregate in maps (`count()`, `hist()`) rather than `printf` per event, and filter early with predicates.
- **Map and string limits**: default map entries (`BPFTRACE_MAP_KEYS_MAX`), string size (`BPFTRACE_STRLEN`), and BPF stack of 512 bytes bound what a script can do. Hitting them causes dropped data or load failures.
- **Probe stability**: `kprobe`s target internal kernel functions that change between versions; prefer `tracepoint`s or `fentry` where available. Uprobes on stripped binaries or Go (non-standard ABI, goroutine stacks, `uretprobe` instability) need care.
- **Containers and pid namespaces**: `pid`/`tid` are host-level IDs; `comm` is truncated to 15 chars. Use cgroup IDs (`cgroup`) to filter by container.
- **Clock and time semantics**: `nsecs` is monotonic; measuring wall-clock latency across CPUs or including off-CPU time requires thought (use `sched` tracepoints for off-CPU analysis).
- **Not a monitoring system**: output is ephemeral. For continuous profiling or security observability, use an always-on eBPF product built for it, and use bpftrace to investigate.
