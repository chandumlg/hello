# HashiCorp Nomad

Nomad is a single-binary workload orchestrator that schedules containers, VMs, raw executables, and batch jobs across a cluster of machines, giving you Kubernetes-style scheduling with a fraction of the operational surface area.

## Why teams adopt it

Kubernetes is the default answer, but it brings a large control plane, a sprawling API surface, and a steep learning curve. Nomad is one Go binary that runs as server or client, uses Raft for consensus, and treats non-container workloads (Java jars, legacy binaries, QEMU VMs, GPU batch jobs) as first-class via task drivers.

Common adoption triggers:
- A mixed fleet: containers alongside legacy binaries or VMs that you cannot or will not containerize.
- A small platform team that wants scheduling, rolling deploys, and service discovery without running a full Kubernetes stack.
- Batch and ML workloads (periodic jobs, parameterized dispatch, GPU device plugins) that benefit from a very high-throughput scheduler.
- Multi-region or multi-datacenter clusters with federation built in, and tight integration with Consul (service mesh/discovery) and Vault (secrets).
- Edge or on-prem deployments where a lightweight agent matters.

## Basic usage

**1. Run a local dev agent and check the cluster:**
```bash
nomad agent -dev                 # server + client in one process
export NOMAD_ADDR=http://127.0.0.1:4646
nomad node status
nomad server members
```

**2. Define and deploy a job (`web.nomad.hcl`):**
```hcl
job "web" {
  datacenters = ["dc1"]
  type        = "service"

  group "app" {
    count = 3

    network {
      port "http" { to = 8080 }
    }

    update {
      max_parallel     = 1
      min_healthy_time = "10s"
      auto_revert      = true
    }

    task "server" {
      driver = "docker"
      config {
        image = "ghcr.io/example/web:1.4.2"
        ports = ["http"]
      }
      resources {
        cpu    = 200   # MHz
        memory = 256   # MB
      }
    }
  }
}
```
```bash
nomad job plan web.nomad.hcl     # dry-run: shows placements and the check-index
nomad job run web.nomad.hcl
nomad job status web
nomad alloc logs -f <alloc-id> server
```

**3. Batch jobs: periodic and parameterized dispatch:**
```hcl
job "etl" {
  type = "batch"
  periodic {
    crons            = ["0 2 * * *"]
    prohibit_overlap = true
  }
  group "g" {
    task "run" {
      driver = "exec"
      config { command = "/usr/local/bin/etl.sh" }
    }
  }
}
```
```bash
nomad job run etl.nomad.hcl
nomad job periodic force etl     # trigger now for testing
```

## Pitfalls and things to watch

- **Plan before run.** `nomad job plan` prints a `-check-index`; pass it to `nomad job run` to avoid clobbering a concurrent change.
- **Resources are reservations, not limits for CPU.** CPU is scheduled in MHz shares; memory is enforced by default. Under-declared resources cause noisy neighbors, over-declared ones waste capacity. Use `memory_max` for oversubscription deliberately.
- **Server count and quorum.** Run 3 or 5 servers, never 2 or 4; losing quorum halts scheduling (running allocations keep running). Back up with `nomad operator snapshot save`.
- **Enable ACLs and TLS early.** They are off by default; retrofitting is painful. Bootstrap with `nomad acl bootstrap` and store the token safely.
- **Ecosystem is thinner than Kubernetes.** No CRDs/operators ecosystem, fewer managed offerings, and ingress/storage often means assembling Consul, CSI plugins, and a load balancer yourself.
- **Licensing.** HashiCorp moved Nomad (like Terraform and Vault) to the Business Source License in 2023; check that it fits your use, and note the community fork situation before standardizing.
- **Update and restart behavior.** Without `update` stanzas, job changes roll out all at once; set `max_parallel`, health checks, and `auto_revert` for services. Also tune `restart` and `reschedule` policies so crash loops don't hammer the cluster.
