# Falco

Falco is an open-source runtime security tool (originally from Sysdig, now a CNCF graduated project) that watches system calls and Kubernetes activity in real time and raises alerts when behavior violates a rule — like "a shell was spawned inside a container" or "a process wrote to `/etc/shadow`" — catching threats that only show up once something is actually running.

## Why teams adopt it

Every other security tool in a typical pipeline answers a *before it runs* question: Semgrep checks your source for bug patterns, Trivy scans images and IaC for known CVEs and misconfigurations, OPA enforces policy at admission time. None of them can tell you what's happening *right now* inside a running container — a compromised dependency spawning a reverse shell, a pod reading `/proc/1/environ` to steal secrets, a cryptominer binary being dropped and executed. Falco fills that gap by tapping into the kernel (via eBPF or a kernel module) and evaluating every system call against a rule set, so it catches zero-day and supply-chain attacks that passed every static check because the malicious behavior only exists at runtime.

Teams typically adopt Falco when:
- They run multi-tenant Kubernetes clusters and need intrusion detection at the container/host level, not just network-layer visibility.
- Compliance requirements (PCI-DSS, SOC 2, FedRAMP) call for runtime threat detection and audit logging, not just static scanning.
- They've had (or want to get ahead of) an incident involving a compromised container — e.g., an attacker exec'ing into a pod, escalating privileges, or exfiltrating data — and want alerting on the specific syscall patterns that indicate it.
- They want to feed security events into existing pipelines (Falcosidekick routes alerts to Slack, PagerDuty, Elasticsearch, SIEM tools, etc.) rather than build detection logic from scratch.
- They're already using Sysdig/CNCF tooling and want a vendor-neutral, well-supported runtime layer that integrates with Kubernetes audit logs as well as raw syscalls.

## Basic usage

**1. Install and run Falco on a node (via Helm, the common path for Kubernetes):**
```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update
helm install falco falcosecurity/falco \
  --namespace falco --create-namespace \
  --set driver.kind=modern_ebpf
```
This deploys Falco as a DaemonSet so every node gets its own instance watching syscalls.

**2. Write a custom rule to detect a specific behavior:**
```yaml
# custom-rules.yaml
- rule: Shell Spawned in Container
  desc: Detect an interactive shell launched inside any container
  condition: >
    spawned_process and container and
    proc.name in (bash, sh, zsh)
  output: >
    Shell spawned in container (user=%user.name container=%container.name
    command=%proc.cmdline image=%container.image.repository)
  priority: WARNING
  tags: [container, shell]
```
Load it alongside the defaults (`falco_rules.yaml`) by mounting it and adding it to Falco's config, then Falco starts emitting alerts the moment a matching process spawns.

**3. Tail alerts locally to see the engine work before deploying it cluster-wide:**
```bash
# Run Falco directly against the local kernel (useful for testing rules)
sudo falco -r custom-rules.yaml

# In another terminal, trigger the rule
docker run --rm -it alpine sh
# Falco prints a WARNING-priority alert for the shell spawn
```

## Common pitfalls

- **Rule noise drowns out signal.** The default rule set is broad and will fire constantly in a normal cluster (package installs during builds, legitimate debug shells, service meshes exec'ing health checks). Teams that skip tuning end up muting Falco in Slack within a week. Budget real time to write exceptions (`exceptions:` blocks) or disable rules that don't fit your environment, rather than disabling Falco alerts wholesale.
- **Kernel/driver compatibility.** The older kernel-module driver needs to match your exact kernel version and can fail silently on unsupported distros or managed nodes (GKE, EKS with custom AMIs). Modern eBPF driver (`driver.kind=modern_ebpf`, requires kernel 5.8+) avoids most of this — check node kernel versions before rolling out broadly, especially on mixed-version fleets.
- **Detection without response is just logging.** Falco alerts don't do anything on their own — pair it with Falcosidekick (or custom webhook consumers) to route alerts somewhere actionable, and decide up front whether you want automated response (e.g., killing the pod) versus alert-only, since automated kill actions on false positives can take down production workloads.
- **Performance overhead is real but often overstated.** Falco runs in-kernel via eBPF and is generally low-overhead, but very high syscall-volume workloads (high-throughput databases, heavy I/O services) can see measurable CPU cost — benchmark on a representative node before assuming it's free.
- **It sees containers, not necessarily VMs or serverless.** Falco's syscall-level visibility applies to hosts and containers it can instrument; it doesn't cover managed serverless functions (Lambda, Cloud Run) the same way, so a "runtime security" strategy that includes those needs a different tool alongside Falco.
