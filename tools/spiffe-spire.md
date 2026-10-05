# SPIFFE / SPIRE

SPIFFE (Secure Production Identity Framework For Everyone) is a CNCF-graduated standard for giving every workload a cryptographically verifiable identity, and SPIRE is its reference implementation — together they replace long-lived shared secrets and IP-based trust with short-lived, automatically rotated identity documents that services use to authenticate each other.

## Why teams adopt it

In most platforms, service-to-service trust is held together by static API keys, shared database passwords, cloud IAM keys copied into env vars, or network location ("it's in the VPC, so it's fine"). These secrets leak, rarely rotate, and don't survive crossing cluster, cloud, or on-prem boundaries. SPIFFE defines a uniform answer to "who is this workload?": a **SPIFFE ID** (a URI such as `spiffe://prod.example.com/ns/payments/sa/api`) carried in a short-lived **SVID** (SPIFFE Verifiable Identity Document), delivered as either an X.509 certificate or a JWT. SPIRE does the hard part: it **attests** workloads (proves what they are using platform evidence such as the Kubernetes service account, pod labels, AWS instance identity document, or Unix UID) and then issues and rotates SVIDs without any human handling a secret.

Teams typically adopt it when:
- They want mutual TLS between services without running a bespoke PKI or hand-managing certs (Istio, Envoy, and Linkerd can all consume SPIFFE identities).
- They span multiple clusters, clouds, or VMs-plus-Kubernetes and need one identity scheme across them, including **federation** between separate trust domains.
- They want to eliminate static credentials for workload-to-workload and workload-to-cloud access (e.g., exchanging a JWT-SVID for AWS/GCP credentials via OIDC federation, or authenticating to Vault without a stored token).
- They are building a zero-trust posture and need authorization policy (OPA, OpenFGA, Envoy RBAC) keyed on a strong identity rather than an IP.
- Compliance or security reviews demand short certificate lifetimes and an auditable issuance trail.

## Basic usage

**1. Install SPIRE on Kubernetes (server plus per-node agent) with Helm:**
```bash
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened/
helm repo update
helm install spire-crds spiffe/spire-crds -n spire-mgmt --create-namespace
helm install spire spiffe/spire -n spire-mgmt \
  --set global.spire.trustDomain=prod.example.com
```
The server is the CA and registry; an agent DaemonSet runs on each node and exposes the **Workload API** over a Unix socket, where workloads fetch their SVIDs.

**2. Register a workload using the spire-controller-manager CRD:**
```yaml
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: payments-api
spec:
  spiffeIDTemplate: "spiffe://prod.example.com/ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}"
  podSelector:
    matchLabels:
      app: payments-api
  namespaceSelector:
    matchLabels:
      kubernetes.io/metadata.name: payments
```
Any matching pod is automatically attested and issued an identity. (Without the controller, the equivalent is `spire-server entry create -spiffeID ... -parentID ... -selector k8s:ns:payments`.)

**3. Fetch an identity from inside a workload:**
```bash
# Mount the agent socket (CSI driver or hostPath) at /spiffe-workload-api/spire-agent.sock
spire-agent api fetch x509 -socketPath /spiffe-workload-api/spire-agent.sock
```
In real code use a SPIFFE library (go-spiffe, java-spiffe, py-spiffe) so certificates rotate in place:
```go
source, _ := workloadapi.NewX509Source(ctx)
defer source.Close()
tlsCfg := tlsconfig.MTLSServerConfig(source, source,
    tlsconfig.AuthorizeID(spiffeid.RequireFromString("spiffe://prod.example.com/ns/web/sa/frontend")))
```
This builds an mTLS server that only accepts the specific frontend identity, with no cert files on disk.

## Common pitfalls

- **Attestation is your real security boundary.** A SPIFFE ID is only as trustworthy as the selectors that grant it. Overly broad selectors (a whole namespace, or just a label an app owner can set) let unintended workloads mint privileged identities. Review registration entries like IAM policies.
- **The SPIRE server is critical infrastructure.** If it's down, new workloads can't get identities and existing SVIDs eventually expire. Run it HA with a shared datastore (Postgres/MySQL, not the default SQLite), and plan its upstream CA (UpstreamAuthority plugin for Vault/cert-manager/AWS PCA) rather than relying on a self-signed root in production.
- **Short TTLs expose sloppy clients.** Default SVID lifetimes are about an hour, rotated at half-life. Apps that load a cert once at startup will break; always use the Workload API streaming source or a sidecar/proxy that handles rotation.
- **Identity is not authorization.** SPIFFE tells you *who* is calling; you still need policy (Envoy RBAC, OPA, OpenFGA) deciding what that identity may do.
- **Trust domain design is hard to change later.** Pick the trust-domain name and ID path convention (`/ns/.../sa/...` vs. team/service) deliberately, and set up federation or bundle distribution before you need cross-cluster calls.
- **Operational surface area.** Socket mounting (use the SPIFFE CSI driver), agent upgrades, and clock skew all cause confusing failures. Monitor agent health and SVID expiry from day one.
