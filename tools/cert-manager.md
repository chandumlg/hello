# cert-manager

**cert-manager is a Kubernetes-native controller that automates the issuance, renewal, and rotation of X.509 certificates from sources like Let's Encrypt, Vault, and private CAs, so TLS stops being a manual, expiry-driven chore.**

## Primary use cases and when to adopt it

- **Ingress/Gateway TLS**: automatically obtain and renew public certificates (ACME / Let's Encrypt) for hostnames on Ingress or Gateway API resources.
- **Internal PKI**: issue short-lived certs from an internal CA, HashiCorp Vault, AWS Private CA, or Google CAS for service-to-service mTLS.
- **Service mesh and workload identity**: via `csi-driver` and `istio-csr`, issue per-pod certificates to meshes such as Istio.
- **Webhook and operator certs**: many operators (including admission webhooks) rely on cert-manager for their serving certs.

Adopt it when you run more than a handful of certificates, have been bitten by an expired cert, want short-lived certs (days, not years), or are standardizing platform-level TLS across many teams/clusters. If you only have one ingress and a managed load balancer cert, a cloud-provided cert may be simpler.

Core model: an **Issuer/ClusterIssuer** defines *where* certs come from; a **Certificate** declares *what* you want; cert-manager creates CertificateRequests/Orders and stores the result in a **Secret** that it renews before expiry (default: at 2/3 of lifetime).

## Getting started

**1. Install (Helm):**

```bash
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true
kubectl -n cert-manager rollout status deploy/cert-manager
```

**2. Create a Let's Encrypt ClusterIssuer (start with staging!):**

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: platform@example.com
    privateKeySecretRef:
      name: letsencrypt-staging-account-key
    solvers:
      - http01:
          ingress:
            ingressClassName: nginx
```

**3. Request a certificate, either explicitly or by annotating an Ingress:**

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: app-tls
  namespace: apps
spec:
  secretName: app-tls
  dnsNames: [app.example.com]
  issuerRef:
    name: letsencrypt-staging
    kind: ClusterIssuer
```

```bash
kubectl -n apps get certificate app-tls          # READY should become True
kubectl -n apps describe certificate app-tls     # events, failures
cmctl status certificate app-tls -n apps         # richer diagnostics (cmctl CLI)
```

For an Ingress, annotate with `cert-manager.io/cluster-issuer: letsencrypt-staging` and add a `tls:` block; cert-manager creates the Certificate for you.

## Common pitfalls

- **Let's Encrypt rate limits**: debug against the staging endpoint first; switch to production only once the flow works.
- **Challenge type mismatch**: HTTP-01 needs the port 80 path reachable from the internet and cannot issue wildcards; wildcards require DNS-01 with correct DNS provider credentials/RBAC. Private clusters usually need DNS-01.
- **Stuck Orders/Challenges**: when a Certificate isn't Ready, inspect the chain `Certificate → CertificateRequest → Order → Challenge`; the real error is usually on the Challenge.
- **ClusterIssuer secret namespace**: ClusterIssuer secrets are read from cert-manager's "cluster resource namespace" (default `cert-manager`), not the Certificate's namespace.
- **Apps don't reload renewed certs**: cert-manager updates the Secret, but pods must pick it up (mounted volumes update, env vars don't). Use reloaders or apps that watch files.
- **Private key rotation**: default `rotationPolicy` is `Never`; set `rotationPolicy: Always` if you want a fresh key on each renewal.
- **CRD/upgrade handling**: manage CRDs deliberately (`crds.enabled`, `crds.keep`), and read release notes; deleting the CRDs deletes all Certificates.
- **Monitor expiry anyway**: scrape cert-manager's Prometheus metrics (e.g. `certmanager_certificate_expiration_timestamp_seconds`) and alert on certs nearing expiry; automation can silently fail.
