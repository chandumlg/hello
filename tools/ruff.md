# Ruff

**Ruff is an extremely fast Python linter and code formatter, written in Rust, that replaces Flake8, isort, pyupgrade, Black and dozens of plugins with a single tool and a single config file.**

## Primary use cases and when to adopt it

- **Monorepo / CI speed:** Ruff lints a large codebase in well under a second (10-100x faster than Flake8/Pylint), so it can run on every save, in pre-commit, and in CI without being a bottleneck.
- **Toolchain consolidation:** One `pyproject.toml` section replaces `.flake8`, `isort.cfg`, `pyupgrade`, `pydocstyle`, `bandit`-style rules and Black. Fewer pinned dev dependencies, fewer version conflicts.
- **Large-scale migrations:** Auto-fixes (`--fix`) apply safe rewrites across thousands of files: upgrading to modern syntax (`UP` rules), sorting imports (`I`), removing unused imports (`F401`), banning deprecated APIs (`TID`, `TCH`).
- **Platform / enforcement teams:** Rules are selectable by prefix, with per-file overrides, so a platform team can ship a shared baseline config (via `extend`) and let services tighten it.

Adopt it when lint/format time shows up in your CI profile, when you maintain several overlapping linters, or when you want a uniform baseline across many repos. Teams generally switch incrementally: start with the linter alone, then the formatter.

## Getting started

1. **Install and run (works well with uv):**
   ```bash
   uv tool install ruff        # or: pip install ruff
   ruff check .                # lint
   ruff check . --fix          # apply safe autofixes
   ruff format .               # Black-compatible formatter
   ```

2. **Configure in `pyproject.toml`:**
   ```toml
   [tool.ruff]
   line-length = 100
   target-version = "py312"

   [tool.ruff.lint]
   select = ["E", "F", "I", "UP", "B", "SIM", "S"]   # pycodestyle, pyflakes, isort, pyupgrade, bugbear, simplify, security
   ignore = ["E501"]

   [tool.ruff.lint.per-file-ignores]
   "tests/**" = ["S101"]      # allow assert in tests
   ```

3. **Wire into pre-commit and CI:**
   ```yaml
   # .pre-commit-config.yaml
   - repo: https://github.com/astral-sh/ruff-pre-commit
     rev: v0.8.0   # pin to a current release
     hooks:
       - id: ruff
         args: [--fix]
       - id: ruff-format
   ```
   In CI use `ruff check --output-format=github .` and `ruff format --check .` for inline PR annotations and a non-mutating format gate.

Handy extras: `ruff check --select UP --diff` previews a rule's changes before applying; `ruff rule F401` explains any rule; `ruff check --add-noqa` baselines existing violations when enabling a new rule on legacy code.

## Common pitfalls

- **Version pinning:** Ruff ships new rules and changes preview behavior often. Pin the version (in `uv.lock`/pre-commit `rev`) and upgrade deliberately, or CI will break on unrelated PRs.
- **`select = ["ALL"]` is a trap:** It turns on every rule including conflicting ones (e.g. docstring-style pairs) and new rules on upgrade. Prefer an explicit allowlist.
- **Not a type checker:** Ruff doesn't do type inference; keep mypy/pyright for that. Some rules approximate it but are not equivalent.
- **Formatter is not 100% Black:** It is ~99.9% compatible, but differences exist (notably in edge cases with docstrings and parentheses). Expect a one-time reformat diff; isolate it in its own commit and add it to `.git-blame-ignore-revs`.
- **Fix safety levels:** "Unsafe" fixes need `--unsafe-fixes` and can change behavior. Review the diff rather than blindly applying them.
- **Formatter/linter conflicts:** Rules like `E501`, `W191`, `COM812`, and `ISC001` can fight the formatter. Ruff warns about these; heed the warning and ignore them.
- **Plugin gaps:** Ruff reimplements popular plugins but not arbitrary third-party or custom Flake8 plugins. Audit any bespoke checks before removing Flake8.
