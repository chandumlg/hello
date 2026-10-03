# Cosign (Sigstore)

Cosign is the signing and verification tool from the Sigstore project that lets you cryptographically sign container images and other build artifacts — and verify them at deploy time — without having to run your own PKI or manage long-lived signing keys.

## Why teams adopt it

Scanners like Trivy tell you what is *inside* an artifact; Cosign tells you *who built it and whether it was tampered with*. Platform and security engineers adopt it when "we pulled `myapp:latest` from the registry" is no longer an acceptable provenance story.

Common adoption triggers:
- Meeting supply-chain requirements (SLSA, SSDF, customer or regulator demands) that require signed artifacts and verifiable build provenance.
- Enforcing "only images built by our CI may run in production" via an admission controller (Kyverno, OPA Gatekeeper, or Sigstore's policy-controller) that verifies signatures before a pod is admitted.
- Eliminating signing-key management: **keyless signing** uses short-lived certificates from Fulcio, issued against an OIDC identity (e.g., a GitHub Actions workflow), and records the signature in the Rekor transparency log. No private key to rotate, leak, or store in CI secrets.
- Attaching attestations (SBOMs, vulnerability scan results, SLSA provenance) to an image so downstream consumers can verify them, stored as OCI artifacts alongside the image in the same registry.

## Basic usage

**1. Key-based signing (simplest way to try it):**
```bash
brew install cosign

cosign generate-key-pair            # creates cosign.key / cosign.pub

# Always sign by digest, not by mutable tag
cosign sign --key cosign.key registry.example.com/myapp@sha256:3f1c...

cosign verify --key cosign.pub registry.example.com/myapp@sha256:3f1c...
```

**2. Keyless signing in CI (GitHub Actions):**
```yaml
permissions:
  id-token: write      # lets the job request an OIDC token
  packages: write
steps:
  - uses: sigstore/cosign-installer@v3
  - run: cosign sign --yes ghcr.io/acme/myapp@${{ steps.build.outputs.digest }}
```
Verify by pinning the *identity* that is allowed to sign, not a key:
```bash
cosign verify ghcr.io/acme/myapp@sha256:3f1c... \
  --certificate-identity-regexp 'https://github.com/acme/myapp/\.github/workflows/release\.yml@refs/tags/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

**3. Attach and verify an SBOM attestation:**
```bash
trivy image --format cyclonedx -o sbom.cdx.json registry.example.com/myapp@sha256:3f1c...

cosign attest --yes --type cyclonedx --predicate sbom.cdx.json \
  registry.example.com/myapp@sha256:3f1c...

cosign verify-attestation --type cyclonedx \
  --certificate-identity-regexp '...' --certificate-oidc-issuer '...' \
  registry.example.com/myapp@sha256:3f1c...
```

## Pitfalls and things to watch

- **Signing a tag instead of a digest.** Tags are mutable; signing `myapp:1.4.0` and later re-pushing the tag leaves you with a signature that doesn't match what's running. Sign and deploy by digest.
- **A loose identity check defeats keyless verification.** Verifying with `.*` for identity or issuer proves only that *someone* signed it. Pin the exact repo, workflow file, and ref.
- **Signing is not verifying.** Many teams sign in CI and never enforce anything. The value arrives only when an admission controller or deploy pipeline *rejects* unsigned or wrongly-signed images.
- **Keyless uses public infrastructure by default.** Signing identities and image digests are written to the public Rekor log. For private or air-gapped environments, run private Fulcio/Rekor instances or use key-based/KMS-backed signing (`--key awskms://...`, `gcpkms://...`, `hashivault://...`).
- **Registry and version quirks.** Signatures are stored as extra OCI artifacts in the registry (tag `sha256-<digest>.sig`); some registries, garbage collectors or retention policies delete them. Cosign v2/v3 also changed defaults (e.g., transparency-log and bundle formats), so pin the CLI version in CI and in verifiers.
- **Signatures don't mean "safe".** A signed image can still contain CVEs or malicious code if your build was compromised. Pair signing with scanning, hermetic builds, and provenance attestations.
