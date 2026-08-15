# HashiCorp Vault

## What it is

Vault is a tool for centrally storing, generating, and tightly controlling access to secrets — API keys, database credentials, TLS certificates, encryption keys — so that plaintext secrets never end up hardcoded in config files, environment variables, or source control.

## Why teams adopt it

Every platform hits the same wall eventually: secrets sprawled across `.env` files, CI variables, Kubernetes ConfigMaps, and Slack messages, with no audit trail and no easy way to rotate a compromised key without a fire drill. Vault centralizes that problem and solves several adjacent ones at once:

- **Static secret storage** — a KV store for credentials that services fetch at runtime instead of baking into images or config.
- **Dynamic secrets** — Vault can generate short-lived, on-demand credentials for backends like Postgres, MySQL, AWS IAM, or SSH, so a leaked credential expires in minutes instead of living forever.
- **Encryption as a service (Transit engine)** — apps send plaintext to Vault and get ciphertext back, without ever handling or storing encryption keys themselves.
- **PKI/certificate issuance** — Vault can act as an internal CA, issuing short-lived TLS certs for mTLS between services.
- **Fine-grained access control and audit logging** — every read of every secret is policy-gated and logged, which matters a lot for SOC2/compliance work.

Teams typically reach for Vault once they have more than a handful of services, more than one environment (dev/staging/prod), and a real need to rotate credentials without redeploying everything — i.e., roughly where "just use environment variables" stops being sustainable. It's a natural fit for platform and security engineers building the substrate other teams' services run on.

## Basic usage

**1. Run a dev server and write/read a static secret**

```bash
# Starts an in-memory, unsealed dev server (never do this in production)
vault server -dev

# In another shell
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='<root-token-printed-on-startup>'

vault kv put secret/myapp/db username=admin password=s3cr3t
vault kv get secret/myapp/db
```

**2. Generate dynamic, short-lived database credentials**

```bash
vault secrets enable database

vault write database/config/my-postgres \
  plugin_name=postgresql-database-plugin \
  connection_url="postgresql://{{username}}:{{password}}@localhost:5432/mydb" \
  allowed_roles="readonly" \
  username="vaultadmin" password="vaultpass"

vault write database/roles/readonly \
  db_name=my-postgres \
  creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
  default_ttl="1h" max_ttl="24h"

# Each call mints a brand-new username/password pair, valid for 1 hour
vault read database/creds/readonly
```

**3. Encrypt data without ever handling the key (Transit engine)**

```bash
vault secrets enable transit
vault write -f transit/keys/my-key

vault write transit/encrypt/my-key plaintext=$(base64 <<< "sensitive data")
# -> returns vault:v1:<ciphertext>

vault write transit/decrypt/my-key ciphertext="vault:v1:<ciphertext>"
# -> returns the original base64 plaintext
```

## Pitfalls to watch for

- **Dev mode is a trap.** `vault server -dev` auto-unseals, uses in-memory storage, and hands you a root token — great for a five-minute demo, dangerous if it quietly becomes someone's "temporary" production instance.
- **Unsealing is a real operational burden.** A production Vault uses Shamir key shares to unseal after every restart (or auto-unseal via a cloud KMS). Teams that skip setting up auto-unseal get paged at 3am when a node restarts and stays sealed.
- **Root tokens and overly broad policies.** It's easy to hand every service a token with access to everything under `secret/*`. Write narrow policies per-service/per-path from day one — retrofitting least-privilege later across dozens of integrations is painful.
- **Static secrets don't rotate themselves.** The KV engine is just storage; nothing forces you to rotate a static credential. The value of Vault comes from leaning on dynamic secrets and TTLs wherever the backend supports it, not just moving secrets from `.env` into KV and calling it done.
- **Lease expiration surprises.** Dynamic secrets and tokens expire on a TTL. Clients need to handle renewal (or re-fetch) explicitly — a service that caches a dynamic DB credential past its lease will start failing to connect with no obvious error pointing back at Vault.
- **Vault becomes a single point of failure.** If every service needs Vault to boot (to fetch its DB password, etc.), a Vault outage becomes an everything outage. Production deployments need HA (Raft or Consul storage backend) and clients that fail gracefully, not hard-crash, on a Vault hiccup.
