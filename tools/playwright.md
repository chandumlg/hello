# Playwright

## What it is

Playwright is a browser automation framework (from Microsoft) for **reliable
end-to-end testing across Chromium, Firefox, and WebKit from a single API** —
it solves the problem that real user flows (login, checkout, multi-tab
workflows) are hard to test with unit tests alone, and older browser
automation tools (Selenium in particular) were slow and notoriously flaky
because they didn't wait properly for pages to be ready.

## Primary use cases

Playwright's core idea is **auto-waiting and actionability checks** baked
into every action: instead of a test author sprinkling `sleep(2)` calls to
hope an element has loaded, Playwright waits until an element is visible,
enabled, and stable before clicking or typing into it, and fails fast with a
clear error if it never gets there.

Teams reach for it when:

- **They need real cross-browser E2E coverage** — one test suite that
  actually drives Chromium, Firefox, and WebKit (Safari's engine), which
  matters for anything customer-facing.
- **Existing Selenium suites are flaky and slow**, and the team wants to
  replace ad-hoc waits and retries with a framework that handles timing
  correctly by default.
- **They want tests that also double as debugging tools** — Playwright's
  trace viewer records a full timeline (DOM snapshots, network, console) for
  every test run, which turns "it failed in CI and I can't reproduce it"
  into a scrubbable recording.
- **API + UI testing need to live together** — Playwright can make raw HTTP
  requests in the same test context as browser actions (e.g., seed data via
  an API call, then verify it in the UI), avoiding a second HTTP client.
- **Multi-context scenarios** — testing two logged-in users interacting (chat,
  collaborative editing) by spinning up isolated browser contexts in one test.

It's overkill for pure unit or component logic that doesn't touch a real DOM
— use a unit test framework there and save Playwright for flows that need an
actual browser.

## Basic usage

**1. Install and scaffold a project:**

```bash
npm init playwright@latest
```

This installs the `@playwright/test` runner, downloads browser binaries, and
generates a sample `playwright.config.ts` plus an example test.

**2. Write a test:**

```ts
import { test, expect } from '@playwright/test';

test('user can log in', async ({ page }) => {
  await page.goto('https://example.com/login');
  await page.getByLabel('Email').fill('user@example.com');
  await page.getByLabel('Password').fill('correct-horse');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page.getByText('Welcome back')).toBeVisible();
});
```

`getByRole`/`getByLabel` locate elements the way a user would (by visible
text or ARIA role) rather than brittle CSS selectors, and `expect(...)` calls
auto-retry until the assertion passes or times out.

**3. Run tests and inspect failures:**

```bash
npx playwright test                # run headless across configured browsers
npx playwright test --ui           # interactive UI mode, step through actions
npx playwright show-trace trace.zip  # replay a failed run's full trace
```

**4. Generate a test by recording actions (useful for onboarding a team):**

```bash
npx playwright codegen https://example.com
```

This opens a browser and emits Playwright code as you click around — a fast
way to get a starting selector-correct draft before hand-tuning it.

## Pitfalls to watch for

- **Selector choice determines flakiness.** Auto-waiting fixes *timing*
  flakiness, not *selector* flakiness — CSS selectors tied to layout
  (`div > div:nth-child(3)`) break the moment markup shifts. Prefer
  `getByRole`/`getByTestId` locators, which are both more stable and closer
  to how a real user finds things.
- **Test isolation matters at scale.** Each test should set up its own state
  (via API calls or fixtures) rather than depending on execution order or
  data left over from a previous test — parallel execution (Playwright's
  default) will otherwise produce intermittent, hard-to-reproduce failures.
- **Browser binaries are large and versioned.** CI images need the matching
  browser binaries for the installed `@playwright/test` version; upgrading
  the package without re-running `playwright install` (or rebuilding a
  pinned Docker image) causes version-mismatch failures.
- **Network mocking is powerful but easy to overuse.** `page.route()` can
  stub any request, which is great for testing error states, but mocking too
  much turns an "end-to-end" test into one that no longer exercises the real
  backend — reserve it for cases genuinely hard to trigger otherwise (rate
  limits, third-party outages).
- **Headless vs. headed timing can differ subtly**, especially around
  animations and CSS transitions — a suite that's green headless can
  occasionally reveal timing issues when run headed for debugging; the trace
  viewer is the reliable way to diagnose these rather than guessing.
