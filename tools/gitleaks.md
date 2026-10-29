# Gitleaks

Gitleaks is an open-source secret scanner that detects hardcoded credentials (API keys, tokens, private keys, passwords) in git history, working trees, and arbitrary files, so leaked secrets are caught before — or soon after — they land in a repository.

## Why teams adopt it

A secret committed to git is effectively permanent: even after you delete the line, it lives in history, forks, CI caches, and clones. Security and platform engineers adopt Gitleaks to shift detection as far left as possible (a pre-commit hook), back it with a CI gate, and periodically sweep full history for things that slipped through.

Common adoption triggers:
- A leaked cloud key or token incident, and the follow-up question "what else is lying around in our repos?"
- Compliance requirements (SOC 2, PCI) asking for evidence of automated secret detection.
- Platform teams standardizing a baseline guardrail across hundreds of repos without per-repo setup.
- Onboarding a newly acquired or inherited codebase and needing a fast audit of its full history.

It is a single static Go binary with no server, which makes it easy to run in pre-commit, GitHub Actions, GitLab CI, or a scheduled org-wide job. Detection uses a TOML rule set (regex + keywords + Shannon entropy + optional allowlists), with a large built-in default ruleset covering AWS, GCP, GitHub, Slack, Stripe, private key blocks, and many more.

## Getting started

**1. Scan a repo, including its full history**

```bash
brew install gitleaks            # or: docker run ghcr.io/gitleaks/gitleaks, or go install
gitleaks git -v .                # scan every commit reachable in the repo
gitleaks dir ./some/path         # scan files on disk, no git required
```

Findings print the rule ID, file, line, commit, author, and the (redacted with `--redact`) secret. Exit code is `1` when leaks are found, so it works as a gate directly.

**2. Block secrets before they're committed**

```bash
gitleaks git --pre-commit --staged --redact -v
```

Wire that into a `.pre-commit-config.yaml` (repo `https://github.com/gitleaks/gitleaks`, hook id `gitleaks`) or a plain `.git/hooks/pre-commit`. Only staged changes are scanned, so it stays fast.

**3. Customize rules and suppress known false positives**

```toml
# .gitleaks.toml
title = "org config"

[extend]
useDefault = true            # keep built-in rules, add ours on top

[[rules]]
id          = "internal-service-token"
description = "Internal service token"
regex       = '''svc_[a-z0-9]{32}'''
keywords    = ["svc_"]       # cheap prefilter; big speedup on large repos

[allowlist]
paths   = ['''(^|/)testdata/''', '''\.lock$''']
regexes = ['''EXAMPLE_KEY_DO_NOT_USE''']
```

For individual known-benign findings, add a trailing `# gitleaks:allow` comment on the line, or record the finding's fingerprint in a `.gitleaksignore` file. To adopt on a noisy legacy repo, generate a baseline (`--report-path baseline.json`) and run later scans with `--baseline-path baseline.json` so CI only fails on *new* leaks.

## Pitfalls and things to watch for

- **Detection is not remediation.** Once a secret has been pushed, treat it as compromised: rotate it first. Rewriting history (filter-repo/BFG) is secondary cleanup, since forks, caches, and PR refs may still hold it.
- **False positives and fatigue.** High-entropy strings (hashes, test fixtures, UUIDs) trip generic rules. Tune with path/regex allowlists and rule `keywords`, otherwise developers learn to bypass the hook (`--no-verify`).
- **Pre-commit is advisory.** Hooks are local and skippable; always enforce the same scan in CI as the real gate.
- **Full-history scans are slow on big monorepos.** In PR CI, scan only the new commit range (`--log-opts="origin/main..HEAD"`) and reserve full-history scans for a scheduled job.
- **Verification is not built in.** Gitleaks matches patterns; it doesn't check whether a key is still live. If you need "is this credential active?" triage, compare with TruffleHog, which verifies against provider APIs, and consider running both.
- **Only sees what you point it at.** It won't catch secrets in Slack, tickets, container image layers, or build logs unless you feed that content to `gitleaks stdin`/`dir`. Pair it with image scanning (e.g. Trivy) and GitHub's native secret scanning/push protection for defense in depth.
- **Version/CLI drift.** Commands changed in v8.19+ (`gitleaks git` / `gitleaks dir` replaced `detect`/`protect`); pin the version in CI and check old blog snippets before copying them.
