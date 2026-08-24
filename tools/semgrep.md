# Semgrep

Semgrep is an open-source static analysis engine that lets you write lightweight, pattern-based rules — that look almost like the code itself — to find bugs, security vulnerabilities, and enforce code standards across large codebases without the ceremony of a full AST-parsing framework.

## Why teams adopt it

Staff and security engineers reach for Semgrep when they need to catch a class of problem across an entire codebase (or every codebase in an org) faster than code review can, and faster than standing up a heavyweight SAST (static application security testing) tool would allow. Unlike traditional linters that are tied to one language's ecosystem and syntax rules, or commercial SAST tools that require significant setup and produce results that are slow to iterate on, Semgrep rules are written in a small YAML DSL where the pattern to match looks like real source code with metavariables (`$X`, `$FUNC`) standing in for the parts you don't care about. That makes it approachable enough for an application security team — or any senior engineer — to write a custom rule in minutes, rather than needing a compiler background.

Common adoption triggers:
- Security teams needing to enforce guardrails across many repos/languages (e.g., "no `eval()` on user input," "no hardcoded AWS keys," "always use the parameterized query API, never string-concatenated SQL").
- Platform/staff engineers rolling out an internal API migration and wanting to find every call site of a deprecated function, or block new usages in CI while a migration is in progress.
- Replacing ad-hoc `grep`/`ripgrep` scripts in CI with something that understands code structure — so a rule matches `foo(bar)` and `foo( bar )` and `foo(\n  bar\n)` alike, and ignores matches inside comments or strings.
- Wanting curated, community-maintained rulesets (Semgrep Registry) for OWASP Top 10-style findings across a dozen+ languages without writing rules from scratch.
- Needing SAST results fast in a PR/CI feedback loop — Semgrep scans are typically seconds to a couple of minutes even on large monorepos, unlike slower whole-program analyzers.

## Basic usage

**1. Install and run a scan with a community ruleset:**
```bash
# Install (pipx recommended, also available via brew/docker)
pipx install semgrep

# Scan the current directory with a broad, well-vetted default ruleset
semgrep --config auto .
```

**2. Write a custom rule to ban a dangerous pattern:**
```yaml
# rules/no-eval.yaml
rules:
  - id: no-eval-on-request-data
    languages: [python]
    severity: ERROR
    message: >
      Avoid eval() on data derived from user input — this is a code
      injection risk. Use ast.literal_eval() or a proper parser instead.
    patterns:
      - pattern: eval(...)
    metadata:
      category: security
      cwe: "CWE-95"

# Run it
semgrep --config rules/no-eval.yaml .
```

**3. Wire it into CI to fail builds and comment on PRs:**
```bash
# In a GitHub Actions / GitLab CI job
semgrep ci --config auto
# `semgrep ci` auto-detects the diff, only reports new findings
# introduced by the PR by default, and can post inline PR comments
# when run via Semgrep AppSec Platform or a CI integration.
```

## Pitfalls to watch out for

- **Pattern matching isn't full data-flow analysis by default.** Basic `pattern:` rules match syntax shapes, not taint propagation — for tracking "user input reaches a dangerous sink through several function calls," you need Semgrep's `mode: taint` rules, which are more powerful but also more work to write correctly.
- **False positives pile up fast with broad rulesets.** Running `--config auto` or large registry rulesets on a big, older codebase can produce hundreds of findings on day one; teams that don't triage/suppress (`# nosemgrep` comments) or start with a narrow, high-confidence rule set often abandon the tool after the first noisy run.
- **Language/framework coverage varies.** Semgrep supports many languages, but depth of official/registry rule coverage differs a lot by language and framework — don't assume a ruleset as mature for, say, Kotlin or Elixir as it is for JavaScript, Python, Go, or Java.
- **It's a point-in-time scanner, not a runtime guard.** Semgrep only sees what's in the source — it won't catch vulnerabilities introduced via dependencies (that's Semgrep Supply Chain / SCA territory) or issues that only manifest at runtime (that's DAST territory); treat it as one layer, not the whole AppSec program.
- **Rule precision requires iteration.** A first-draft rule that looks right often over- or under-matches (e.g., missing that an argument can be a method call, not just a variable); use `semgrep --test` with example good/bad snippets to validate rules before rolling them out org-wide.
