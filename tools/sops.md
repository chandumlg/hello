# SOPS (Secrets OPerationS)

## What it is and the problem it solves

SOPS is a CLI editor of encrypted files that encrypts only the *values* of YAML, JSON, ENV, INI, or binary files (keys stay in plaintext), so secrets can live in Git with readable, reviewable diffs while key management is delegated to a KMS (AWS KMS, GCP KMS, Azure Key Vault, HashiCorp Vault) or to age/PGP.

## Primary use cases and when to adopt it

- **GitOps secrets**: Flux has native SOPS decryption; Argo CD supports it via plugins/helm-secrets/KSOPS. Secrets are versioned and reviewed like any other manifest.
- **Terraform/Helm/Ansible inputs**: encrypted `values.yaml` or `secrets.tfvars.json` committed beside the code that uses them.
- **Small-to-mid teams without a secrets server**: no Vault cluster to run; access control is "who can use this KMS key / hold this age key."
- **Multi-environment config**: different recipients per path (prod keys only for the prod platform group) via `.sops.yaml` creation rules.

Adopt it when you want secrets-as-code with an audit trail and don't need dynamic credentials, leasing, or fine-grained runtime policy (that's Vault territory). It complements Vault rather than replaces it.

## Getting started

Install (`brew install sops age`) and generate an age key:

```bash
age-keygen -o ~/.config/sops/age/keys.txt
# prints: Public key: age1abc...
```

Configure per-path rules once in `.sops.yaml` at the repo root:

```yaml
creation_rules:
  - path_regex: ^secrets/prod/.*\.yaml$
    encrypted_regex: ^(data|stringData|password|token)$   # only encrypt these keys
    kms: arn:aws:kms:us-east-1:111122223333:key/abcd-...
    age: age1abc...                                       # break-glass recipient
  - path_regex: ^secrets/dev/.*\.yaml$
    age: age1devteam...
```

Create/edit and use:

```bash
sops secrets/dev/app.yaml          # opens $EDITOR on decrypted content; re-encrypts on save
sops -e -i secrets/dev/app.yaml    # encrypt an existing plaintext file in place
sops -d secrets/dev/app.yaml       # decrypt to stdout
sops exec-env secrets/dev/app.env './run-server.sh'   # inject as env vars without writing plaintext to disk
sops updatekeys secrets/prod/app.yaml                 # apply recipient changes from .sops.yaml
```

Kubernetes + Flux sketch: set `spec.decryption: {provider: sops, secretRef: {name: sops-age}}` on the `Kustomization`, and keep the age private key in a cluster Secret.

## Common pitfalls

- **Encrypting the whole file with no `encrypted_regex`** in k8s manifests breaks `kind`/`metadata`, making Kustomize/Helm unable to read them. Use `encrypted_regex: ^(data|stringData)$`.
- **Committed plaintext before encrypting**: it stays in Git history. Rotate the secret; encrypting afterwards doesn't un-leak it. Add a pre-commit check (e.g. verify a `sops:` block exists).
- **Removing a recipient doesn't revoke past access**: anyone who had the key and cloned the repo can decrypt old versions. Rotate the data key *and* the underlying secrets (`sops rotate -i`).
- **Merge conflicts**: files carry a MAC and per-value ciphertext; concurrent edits conflict noisily. Resolve by decrypting both sides (or re-doing the change via `sops`), never hand-editing ciphertext. A tampered file fails MAC verification.
- **Key availability in CI/cluster**: losing the age key or KMS access means losing the secrets. Keep a break-glass recipient and back up age keys securely.
- **Metadata leaks**: key names, structure, and file paths are plaintext. Don't put sensitive information in key names.
- **Not a runtime secrets manager**: no rotation, leasing, or per-read audit. Decrypted values in k8s Secrets are only base64 at rest unless etcd encryption is enabled.
