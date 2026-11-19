# Teleport

Teleport is an identity-aware access proxy. Engineers and machines use it to reach SSH servers, Kubernetes clusters, databases, internal web apps and Windows desktops with short-lived certificates tied to their SSO identity, instead of static keys, kubeconfigs and shared database passwords. Every session is recorded and audited in one place.

## Why teams adopt it

Infrastructure access usually piles up over time: SSH keys in `authorized_keys` that nobody rotates, a bastion host, VPN profiles, kubeconfigs with long-lived tokens, a shared `readonly` Postgres password in a wiki, and offboarding that means "grep for their key everywhere." Teleport replaces all of that with one model:

- **Certificate-based auth.** Teleport runs its own SSH and X.509 certificate authorities. When you `tsh login` through your IdP (Okta, Entra ID, Google, GitHub, any OIDC/SAML provider), you get certificates that expire in hours. There's nothing long-lived to leak, rotate or revoke. Remove someone from the IdP group and their access is gone at the next expiry.
- **One proxy for every protocol.** The **Proxy Service** is the only thing exposed to the internet, and it multiplexes SSH, Kubernetes API, Postgres/MySQL/Mongo/Redis wire protocols, HTTP and RDP over a single TLS port (443). Resources sit behind **agents** that dial *out* to the proxy (reverse tunnels), so private subnets need no inbound firewall rules.
- **Label-based RBAC.** Resources carry labels (`env=prod`, `team=payments`), and roles grant access by label selector. New nodes or databases inherit the right access automatically when they register.
- **Just-in-time access.** Users hold low-privilege roles by default and file **Access Requests** for elevated roles ("prod-admin for 2h, reason: INC-4812"). Requests are approved in Slack, PagerDuty, Jira or the CLI, and the grant expires on its own.
- **Audit and session recording.** SSH and `kubectl exec` sessions are recorded and can be replayed. Database queries and every login, request and approval go to a structured audit log you can ship to a SIEM.
- **Machine ID (`tbot`).** CI jobs, cron jobs and services get the same short-lived certificates, so you don't need static deploy keys or service-account tokens either.

The architecture has three parts. The **Auth Service** is the CA and the policy and audit store. The **Proxy Service** is the public entry point. **Agents** run the SSH, Kubernetes, DB, App and Desktop services next to the resources they protect.

Common adoption triggers:
- SOC 2, ISO 27001, PCI or HIPAA audits ask "who accessed production, when, and what did they do?" and the honest answer is "we'd have to check bastion logs."
- Offboarding or a laptop compromise showed you can't enumerate every credential a person holds.
- You run many Kubernetes clusters and databases across clouds or VPCs and are tired of per-cluster IAM mapping and VPN sprawl.
- You want standing prod access set to zero, with elevation only on request.

## Basic usage

**1. Log in and reach resources as a user:**
```bash
# SSO login through the proxy. This issues short-lived SSH and X.509 certs.
tsh login --proxy=teleport.example.com --auth=okta
tsh status                      # shows your roles, logins and cert expiry

# SSH
tsh ls env=staging              # list nodes filtered by label
tsh ssh ubuntu@api-staging-1    # works for nodes in private subnets too
tsh ssh ubuntu@env=staging "uptime"   # run on every matching node

# Kubernetes: tsh writes a kubeconfig context that uses your Teleport cert
tsh kube ls
tsh kube login prod-eu
kubectl get pods -n payments

# Databases: no shared password; the DB user is mapped from your role
tsh db ls
tsh db connect --db-user=reader --db-name=orders orders-prod   # opens psql
tsh proxy db --tunnel --port 5433 orders-prod                  # local tunnel for GUI tools
```

**2. Define a role with label-based access and just-in-time elevation:**
```yaml
# roles.yaml
kind: role
version: v7
metadata:
  name: payments-dev
spec:
  options:
    max_session_ttl: 8h
  allow:
    logins: [ubuntu]
    node_labels:
      team: payments
      env: [dev, staging]
    kubernetes_groups: [payments-viewers]
    kubernetes_labels:
      env: [dev, staging]
    kubernetes_resources:
      - kind: pod
        namespace: payments
        name: "*"
    db_labels:
      team: payments
    db_users: [reader]
    db_names: [orders]
    # Holders of this role may *request* prod access, but don't have it.
    request:
      roles: [payments-prod-admin]
      thresholds:
        - approve: 1
          deny: 1
---
kind: role
version: v7
metadata:
  name: payments-prod-admin
spec:
  options:
    max_session_ttl: 2h
  allow:
    logins: [ubuntu]
    node_labels:
      team: payments
      env: prod
    db_labels:
      team: payments
      env: prod
    db_users: [writer]
```
```bash
tctl create -f roles.yaml

# Users get roles from IdP group claims via the SSO connector's mappings.
# A local user is fine for a first test:
tctl users add alice --roles=payments-dev --logins=ubuntu

# Later, during an incident, Alice elevates:
tsh request create --roles=payments-prod-admin --reason="INC-4812: stuck migration"
# An approver runs:  tctl requests approve <request-id>   (or clicks in Slack)
tsh login --request-id=<request-id>   # certs now include prod access for up to 2h
```

**3. Give a CI pipeline short-lived credentials with Machine ID:**
```yaml
# tbot.yaml: runs inside a GitHub Actions job and joins with the job's OIDC token
version: v2
proxy_server: teleport.example.com:443
onboarding:
  join_method: github
  token: gha-deploy-bot     # a join token resource bound to repo + branch claims
storage:
  type: memory
outputs:
  - type: kubernetes
    destination:
      type: directory
      path: ./tbot-out
    kubernetes_cluster: prod-eu
```
```bash
tbot start -c tbot.yaml --oneshot
KUBECONFIG=./tbot-out/kubeconfig.yaml kubectl rollout status deploy/api -n payments
```
The `github` join method trusts GitHub's OIDC token for one specific repository and workflow, so the pipeline stores no Teleport secret at all. Similar join methods exist for AWS IAM, GCP, Azure, GitLab, CircleCI and Kubernetes service accounts.

## Common pitfalls

- **The proxy and auth tier is now critical-path infrastructure.** If the Auth Service or Proxy is down, nobody can log into anything, including during the incident you're trying to fix. Run Auth with an HA backend (DynamoDB, Firestore, or etcd/Postgres for self-hosted) and multiple proxy replicas. Keep a documented, tested break-glass path, such as a sealed emergency SSH key or cloud console access, that doesn't depend on Teleport.
- **Version skew.** Agents and `tsh`/`tctl` clients must stay within one major version of the cluster, and you upgrade the Auth Service first. If fleets of agents are pinned to old AMIs or Helm values, they disconnect after a cluster upgrade. Turn on managed agent updates, or tie agent versions to your upgrade runbook.
- **Overly broad label selectors.** `node_labels: {'*': '*'}` in a role meant for "everyone" quietly grants prod. Prefer explicit `env` and `team` labels, and treat role YAML as code: keep it in Git, review it in PRs, apply it with the Terraform provider or Kubernetes operator, and test it with `tctl` before applying. Labels are only as trustworthy as the agent that sets them, so use cloud-sourced labels (AWS tags, GCP labels) where you can.
- **Default role sprawl.** Out of the box, `access` and `editor` are permissive. Create narrow roles early, map IdP groups to them in the SSO connector, and don't give engineers `editor`, which can change RBAC itself.
- **Certificate TTL trade-offs.** A long `max_session_ttl` (24h or more) gives up most of the value of short-lived credentials. A very short TTL means constant re-login and broken long-running `kubectl port-forward` or DB sessions. Choosing 8–12h for normal roles and 1–2h for elevated ones is a common middle ground. Also enable per-session MFA for prod roles (`require_session_mfa`), so a stolen cert file alone isn't enough.
- **Session recording storage and privacy.** Recordings of busy SSH and `kubectl exec` sessions add up fast. Set a retention and lifecycle policy on the recordings bucket. Recordings can also capture secrets that are typed or echoed in a shell, so restrict who can replay sessions.
- **Database access isn't a query firewall.** Teleport authenticates the connection and maps it to a database user. Fine-grained permissions still come from database grants, so keep `db_users` mapped to properly scoped database roles. Also check whether your DB engine supports automatic user provisioning before you rely on it.
- **Licensing.** Since v16, Teleport Community Edition binaries carry a license that limits commercial use by larger organizations, and features like Access Request workflows with external approvers, Okta/Entra sync and device trust are Enterprise-only. Check what you actually need before you build processes around a feature.
- **Exposing the proxy wrong.** TLS-routing everything over 443 is convenient, but L7 load balancers that terminate TLS break it. Put the proxy behind an L4/NLB in TLS passthrough mode, or configure ALPN routing explicitly.
