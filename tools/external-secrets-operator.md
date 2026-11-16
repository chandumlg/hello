# External Secrets Operator

External Secrets Operator (ESO) is a Kubernetes operator that reads secrets from an external secret manager (AWS Secrets Manager, HashiCorp Vault, Google Secret Manager, Azure Key Vault and many others) and keeps them synchronized into native Kubernetes `Secret` objects, so the secret manager stays the source of truth and nothing sensitive lives in Git.

## Why teams adopt it

Every team on Kubernetes has to answer the question "how does the database password get into the pod?" The usual answers each have a cost:

- **`kubectl create secret` by hand**: nothing is reproducible, and a rebuilt cluster has no secrets.
- **Encrypted secrets in Git (SOPS, Sealed Secrets)**: reproducible, but Git becomes a second copy of every secret, rotation means a commit, and the secrets your cloud services already generate (RDS passwords, API keys managed by another team) have to be copied in by someone.
- **Apps call the secret manager directly**: no copy in the cluster, but every service needs an SDK, credentials and retry logic, and third-party Helm charts that expect a `Secret` can't use it.

ESO takes a different position: the secret manager you already have is the source of truth, and the cluster holds a cached, automatically refreshed copy. What you commit to Git is a *reference* ("the key `prod/payments/db` in AWS Secrets Manager"), which is safe to review in a pull request. The operator resolves the reference, writes a normal `Secret`, and re-reads the source on an interval. Workloads consume the `Secret` as usual, with no knowledge of where it came from.

The core resources:
- **SecretStore**: a namespaced description of *how* to reach a backend: which provider, which region or URL, and which credentials to authenticate with.
- **ClusterSecretStore**: the same, cluster-scoped, so that many namespaces can share one backend configuration.
- **ExternalSecret**: *what* to fetch from a store and how to shape it into a `Secret` (which keys, what names, what template).
- **ClusterExternalSecret**: stamps the same ExternalSecret into every namespace that matches a selector. Useful for image pull secrets and shared CA bundles.
- **PushSecret**: the reverse direction. It writes a Kubernetes `Secret` out to the external manager, for example to publish a certificate that was generated in the cluster.
- **Generators**: produce values on demand rather than fetching them, such as random passwords or short-lived ECR and GCR registry tokens.

Common adoption triggers:
- You're moving to GitOps (Argo CD, Flux) and need a way to declare secrets in Git without putting their values there.
- The security team wants one audited place for secrets, with rotation, and the cluster keeps drifting from it.
- You run many clusters and want each to pull the same secrets from a central manager instead of copying them at provisioning time.
- You're consolidating: some teams use Vault, some use the cloud provider's manager, and you want one consistent interface in the cluster.
- Third-party charts need a `Secret` to exist, and you want it to be created automatically.

ESO is not the only option in this space. The Secrets Store CSI Driver mounts secrets as files directly into pods without creating a `Secret` object (unless you turn syncing on), which keeps them out of etcd but only works for file mounts and ties the secret's lifetime to the pod. The Vault Agent injector and Vault Secrets Operator are Vault-specific. ESO is the usual choice when you want real `Secret` objects and a backend-neutral API.

## Basic usage

**1. Install the operator with Helm:**
```bash
helm repo add external-secrets https://charts.external-secrets.io
helm repo update

helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets \
  --create-namespace

# Three deployments should become ready: the controller, the webhook and the cert controller.
kubectl get pods -n external-secrets
```

**2. Define a SecretStore that authenticates to AWS Secrets Manager using the namespace's own IAM role:**
```yaml
# The ServiceAccount carries the IAM role (IRSA). The role's policy should allow
# secretsmanager:GetSecretValue and DescribeSecret only on this team's secrets.
apiVersion: v1
kind: ServiceAccount
metadata:
  name: eso-payments
  namespace: payments
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/eso-payments
---
apiVersion: external-secrets.io/v1   # v1beta1 on older ESO releases
kind: SecretStore
metadata:
  name: aws-secrets
  namespace: payments
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: eso-payments
```

```bash
kubectl apply -f secretstore.yaml

# STATUS should be Valid and READY should be True before you go further.
kubectl get secretstore -n payments
```

**3. Declare an ExternalSecret that pulls two fields from one remote secret and renders a connection string:**
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: payments-db
  namespace: payments
spec:
  refreshInterval: 1h              # how often to re-read the source
  secretStoreRef:
    kind: SecretStore
    name: aws-secrets
  target:
    name: payments-db              # the Kubernetes Secret to create
    creationPolicy: Owner          # deleting the ExternalSecret deletes the Secret
    template:
      data:
        # Values fetched below are available to the template by their secretKey.
        DATABASE_URL: "postgres://{{ .username }}:{{ .password }}@db.internal:5432/payments"
  data:
    - secretKey: username
      remoteRef:
        key: prod/payments/db      # the secret's name in AWS Secrets Manager
        property: username         # a field inside its JSON value
    - secretKey: password
      remoteRef:
        key: prod/payments/db
        property: password
```

```bash
kubectl apply -f externalsecret.yaml

# STATUS SecretSynced means the Secret exists and matches the source.
kubectl get externalsecret payments-db -n payments

# If it isn't syncing, the reason is in the events and conditions.
kubectl describe externalsecret payments-db -n payments

# Force a refresh now instead of waiting for the interval.
kubectl annotate externalsecret payments-db -n payments \
  force-sync=$(date +%s) --overwrite
```

To copy every field of a JSON secret without listing them, replace `data` with `dataFrom`:
```yaml
  dataFrom:
    - extract:
        key: prod/payments/db      # each top-level JSON field becomes a key in the Secret
```

## Common pitfalls

- **The secret still ends up in etcd.** ESO moves the source of truth, not the risk of the copy. The resulting `Secret` is base64-encoded, not encrypted, and anyone with `get secrets` in the namespace can read it. You still need etcd encryption at rest and tight RBAC on Secrets. If the requirement is "secrets must never be stored in the cluster", ESO doesn't meet it; look at the CSI driver or direct access from the application.

- **A ClusterSecretStore is a shared credential.** Any namespace that can create an ExternalSecret can reference a ClusterSecretStore and ask for any key its credentials can read. With one broadly scoped store, the `dev-sandbox` namespace can fetch production database passwords. Either use namespaced SecretStores with per-team credentials, as in the example above, or restrict the cluster store with `spec.conditions` (namespace selectors) and scope its IAM or Vault policy to a path prefix. Treat permission to create ExternalSecrets as permission to read secrets.

- **Rotation doesn't reach running pods by itself.** When the source changes, ESO updates the `Secret` on its next refresh, which can be up to a full `refreshInterval` later. Mounted volumes pick up the change after the kubelet's sync delay, but environment variables and `subPath` mounts never update in a running pod. You need something to restart the workload (Stakater Reloader is the common choice, or a checksum annotation in your chart) or an app that re-reads the file. Plan rotations so the old credential stays valid for at least the refresh interval plus rollout time.

- **Refresh interval multiplied by secret count is an API bill and a rate limit.** Each ExternalSecret polls the backend on every refresh, and one with ten `data` entries may make ten calls. Thousands of ExternalSecrets at `refreshInterval: 1m` will cost real money on backends that charge per request and can get throttled, at which point syncs fail across the cluster. Use the longest interval you can tolerate, prefer `dataFrom.extract` on one JSON secret over many single-key lookups, and for values that never change set the interval to `0` so they're fetched once.

- **A backend outage doesn't delete secrets, but it does block new ones.** If the secret manager is unreachable, existing `Secret` objects stay in place and workloads keep running. New namespaces, new ExternalSecrets and anything that was deleted and recreated will wait. That matters most during disaster recovery, when you're rebuilding a cluster and the secret manager is in the affected region. Know which of your secrets are needed to bootstrap and where they're replicated.

- **Ownership and deletion policies have sharp defaults.** With `creationPolicy: Owner`, deleting the ExternalSecret garbage-collects the `Secret`, so a GitOps prune or a renamed manifest can remove a credential from under a running app. If the `Secret` already exists and is managed by something else, use `Merge` so ESO adds keys rather than taking it over. Separately, `deletionPolicy` controls what happens when the *remote* secret disappears: the default keeps the stale `Secret`, and `Delete` removes it.

- **Bootstrap order matters under GitOps.** The CRDs and the webhook must be ready before any SecretStore or ExternalSecret is applied, and the Secret must exist before the pod that mounts it starts. In Argo CD, put the operator in an earlier sync wave than its custom resources, and expect pods to sit in `CreateContainerConfigError` briefly on first deploy until the `Secret` appears. The operator's own credentials are the remaining "secret zero" problem, so use workload identity (IRSA, GKE Workload Identity, Azure Workload Identity, Vault's Kubernetes auth) rather than a static access key stored in a `Secret`.

- **Templates and `property` lookups fail quietly at the edges.** `property` expects the remote value to be JSON; a plain string, or a missing field, gives a sync error that's only visible in the ExternalSecret's status. A typo in a template key renders as an error or an empty value depending on the function. Alert on the operator's sync metrics (it exports Prometheus metrics for sync calls and errors, and a readiness condition per ExternalSecret) so a broken reference is noticed when it breaks, not when the pod next restarts.

- **API versions have moved.** The resources were promoted from `v1beta1` to `v1`, and newer releases stop serving the older version. Before upgrading the operator across several releases, read the upgrade notes and update your manifests first. `PushSecret` and the generators are still on alpha API versions, so expect their schemas to change.
