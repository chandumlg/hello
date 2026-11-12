# Renovate

Renovate is an open-source dependency update bot (maintained by Mend) that scans your repositories for every pinned version it can find, such as packages, Docker image tags, Helm charts, Terraform providers and GitHub Actions, and opens pull requests to bump them, so staying current becomes a stream of small reviewable changes instead of a quarterly upgrade project.

## Why teams adopt it

Dependency drift is cheap to ignore and expensive to pay off. A service that is two years behind has to cross several major versions at once, usually under pressure from a CVE. Manual upgrades don't scale past a handful of repos, and nobody owns the versions hiding in Dockerfiles, CI workflows and Helm values.

Renovate handles this with three pieces:

- **Managers** know how to find dependencies in a file type (`npm`, `gomod`, `pip_requirements`, `dockerfile`, `helm-values`, `terraform`, `github-actions`, and many more).
- **Datasources** know where to look up new versions (npm registry, PyPI, Docker registries, GitHub releases and tags, Helm repos).
- **Versioning schemes** know how to compare them (semver, PEP 440, Docker tag conventions, and others).

Each run extracts dependencies, looks up newer versions, and creates or rebases one branch and PR per update or per group, with release notes included in the PR body. Policy lives in a `renovate.json` in the repo, and it can extend shared presets, so a platform team can keep one org-wide config that every repo inherits.

Compared to Dependabot, Renovate's draw is breadth and control: more ecosystems, grouping rules, schedules, automerge policy, regex-based custom managers for versions no built-in manager understands, a Dependency Dashboard issue that lists everything pending, and support for GitHub, GitLab, Bitbucket, Azure DevOps and Gitea/Forgejo.

Common adoption triggers:
- You run many repos or a monorepo and want one consistent update policy rather than per-repo habits.
- A security review found base images and CI actions pinned to tags from years ago.
- You want low-risk updates (patches, dev dependencies, lockfile refreshes) merged automatically once CI passes, with humans reviewing only majors.
- You're on a forge other than GitHub, or self-hosted, where Dependabot isn't an option.

You can run it as the hosted Mend Renovate GitHub App, or self-host the CLI or Docker image on a cron job in your own CI.

## Basic usage

**1. Onboard a repo with a config file.** After you install the app (or point a self-hosted run at the repo), Renovate opens a "Configure Renovate" PR. Merging it enables updates. A reasonable starting `renovate.json`:

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["config:recommended"],
  "timezone": "America/Los_Angeles",
  "schedule": ["before 6am on monday"],
  "minimumReleaseAge": "3 days",
  "packageRules": [
    {
      "description": "Automerge patch and minor updates of dev dependencies",
      "matchDepTypes": ["devDependencies"],
      "matchUpdateTypes": ["minor", "patch"],
      "automerge": true
    },
    {
      "description": "One PR for all AWS SDK packages",
      "matchPackageNames": ["@aws-sdk/**"],
      "groupName": "aws-sdk"
    },
    {
      "description": "Majors need a human to approve them on the dashboard first",
      "matchUpdateTypes": ["major"],
      "dependencyDashboardApproval": true
    }
  ]
}
```

`packageRules` are evaluated in order and later matches override earlier ones, so put broad rules first and exceptions after them.

**2. Self-host it and preview what it would do.** The CLI ships on npm and as a Docker image. A dry run prints the planned branches and PRs without creating anything:

```bash
export RENOVATE_TOKEN=<forge token with repo access>

# Preview only
LOG_LEVEL=debug npx renovate --platform=github --dry-run=full my-org/my-service

# Real run, via Docker, suitable for a scheduled CI job
docker run --rm -e RENOVATE_TOKEN -e LOG_LEVEL=info \
  renovate/renovate --platform=github my-org/my-service

# Check a config file before committing it
npx --package renovate -- renovate-config-validator renovate.json
```

Give the run a GitHub token even when your code lives on another forge (`RENOVATE_GITHUB_COM_TOKEN`), since release notes are fetched from github.com and unauthenticated requests get rate limited quickly.

**3. Track versions no built-in manager understands.** Tool versions in a Dockerfile `ARG`, a Makefile or a shell script can be annotated with a comment and picked up by a regex custom manager:

```dockerfile
# renovate: datasource=github-releases depName=kubernetes-sigs/kustomize extractVersion=^kustomize/v(?<version>.+)$
ARG KUSTOMIZE_VERSION=5.4.3
```

```json
{
  "customManagers": [
    {
      "customType": "regex",
      "managerFilePatterns": ["/(^|/)Dockerfile$/"],
      "matchStrings": [
        "# renovate: datasource=(?<datasource>\\S+) depName=(?<depName>\\S+)( extractVersion=(?<extractVersion>\\S+))?\\s+ARG \\S+_VERSION=(?<currentValue>\\S+)"
      ]
    }
  ]
}
```

The built-in preset `customManagers:dockerfileVersions` covers the common form of this pattern, so check the presets before writing your own regex. Older Renovate versions call the file pattern field `fileMatch`.

## Pitfalls and things to watch out for

- **PR floods.** The first run on a neglected repo can propose dozens of updates. Defaults throttle this (`prHourlyLimit` and `prConcurrentLimit`), but the real fix is grouping, a schedule, and using the Dependency Dashboard to release updates deliberately. Teams that skip this step usually end up ignoring the bot.
- **Automerge is only as safe as your tests.** Renovate merges when required checks pass. With thin CI, or a repo with no required checks, that means merging unverified code. Start automerge with dev dependencies and lockfile maintenance, and require branch protection.
- **Supply-chain exposure.** Fast automatic updates pull in a compromised release quickly. Set `minimumReleaseAge` so new versions sit for a few days before a PR appears, and pin GitHub Actions and Docker images to digests (`helpers:pinGitHubActionDigests`, `docker:pinDigests`) so a moved tag can't change what you run.
- **Private registries fail quietly.** Without `hostRules` credentials, lookups for private packages fail and those dependencies just don't get updates. Check the run logs or the dashboard's warnings rather than assuming silence means up to date.
- **Rebase churn eats CI minutes.** By default Renovate rebases its branches when the base branch moves or conflicts appear. On a busy repo with many open Renovate PRs, each merge can trigger a wave of CI runs. Tune `rebaseWhen` (for example `conflicted`) and keep the open PR count low.
- **Don't push to Renovate's branches casually.** Once someone else commits to a `renovate/*` branch, Renovate stops updating it until you ask it to rebase via the PR checkbox, which discards those commits.
- **Config drift across repos.** Copy-pasted `renovate.json` files diverge. Put shared policy in a preset repo (`"extends": ["github>my-org/renovate-config"]`) and keep per-repo files to the exceptions.
- **Version scheme mismatches.** Docker tags and non-semver projects are sometimes parsed wrongly, giving odd proposals like a "upgrade" to a differently-suffixed tag. Fix it per package with `versioning` or `allowedVersions` rather than disabling the manager.
- **Self-hosting is a real service.** You own the schedule, the token (a GitHub App is better than a personal token for rate limits and identity), the cache, and upgrades of Renovate itself, whose major versions regularly rename config options. Config migration PRs help, but read the release notes.
- **License.** Renovate is AGPL-3.0. Running it unmodified is fine for most organizations, but check with legal before embedding or modifying it in a product.
