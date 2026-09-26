# k9s

k9s is a terminal-based UI that lets you navigate, observe, and manage a Kubernetes cluster interactively instead of typing `kubectl get/describe/logs` over and over.

## Why teams adopt it

Anyone who spends real time operating Kubernetes — platform engineers on call, SREs debugging a rollout, staff engineers triaging a production incident — ends up running the same handful of `kubectl` commands dozens of times a day: list pods, describe the one that's crashing, tail its logs, exec into it, check its resource usage, restart it. k9s turns that workflow into a single always-on, vim-inspired dashboard: resources refresh live, you filter and drill down with a few keystrokes, and common actions (delete, scale, edit, port-forward, view logs) are one keypress away instead of a remembered flag combination.

Teams typically reach for it when:
- On-call engineers need to diagnose cluster issues fast, without hunting for the right `kubectl` incantation under pressure.
- You're managing multiple clusters/namespaces and want to switch context quickly without re-typing `--context`/`-n` on every command.
- You want a lower-friction alternative to a full web dashboard (like the Kubernetes Dashboard or a cloud console) that still works entirely from a terminal/SSH session, with no extra infrastructure to deploy.
- New team members need a faster way to build intuition about what's running in a cluster than memorizing `kubectl` syntax.

It isn't a replacement for `kubectl` in scripts or CI/CD — it's an interactive, human-in-the-loop tool layered on top of the same Kubernetes API.

## Basic usage

**1. Launch it against your current kube context:**
```bash
k9s
```
This opens the pod view for the current namespace using whatever context/cluster your `~/.kube/config` points at.

**2. Jump straight to a resource type or namespace:**
```bash
k9s -n production          # start in the "production" namespace
k9s --context staging      # start against a specific kube context
```
Inside the UI, typing `:deploy`, `:svc`, `:ns`, or `:events` (a "command mode", triggered by `:`) jumps directly to that resource type — no need to restart k9s.

**3. Drill into a pod and act on it:**
- Use arrow keys or `/` to filter and select a pod.
- `l` tails its logs, `d` describes it, `s` shells into it (`exec`), `Ctrl-k` kills it, `y` shows its raw YAML.
- `:pf` sets up a port-forward without leaving the terminal.

These are the interactive equivalents of `kubectl logs`, `kubectl describe pod`, `kubectl exec`, `kubectl delete pod`, and `kubectl port-forward`.

## Pitfalls to watch out for

- **It's a viewer/actuator, not a source of truth for automation.** Don't script around k9s output — it's built for humans watching a screen; use `kubectl`/client libraries for anything programmatic.
- **Destructive actions are one keypress away.** Deleting a pod, scaling a deployment to zero, or killing a resource takes just one or two keys once selected — it's easy to fat-finger an action against the wrong cluster if you're not paying attention to the context shown in the header.
- **Context matters — literally.** Because k9s makes cluster-switching so fast, it's easy to lose track of which cluster/namespace you're actually in, especially with multiple clusters open in different terminal tabs. Always check the top bar before running a destructive command.
- **RBAC still applies.** k9s can only show/do what your kubeconfig's credentials allow; a cluttered or broken UI (missing resources, permission errors) is often an RBAC issue, not a k9s bug.
- **Version/CRD drift.** For clusters with lots of custom resources (CRDs), k9s's built-in views may not render them as richly as the plain `kubectl get/describe`; you can extend it with custom resource definitions in its config, but out of the box some CRDs just show as raw YAML.
