# shy — CI/CD Plan

Concrete workflow specifications for the shy repository's CI/CD.
Each workflow has a clear purpose, a defined trigger, and explicit
status checks that gate merges into protected branches.

The actual workflow YAML files live in
`bootstrap/.github/workflows/`. This document describes *what
each workflow does and why*. The YAML is the source of truth for
*how*.

## Workflow overview

| Workflow | File | Trigger | Purpose |
|---|---|---|---|
| CI | `ci.yml` | push to `next`/`after`/`before`, pull_request | Validates code quality, tests, and plugin-dispatch performance |
| Acceptance | `acceptance.yml` | push to `next`/`before`, pull_request, weekly schedule | End-to-end installation and behavior tests across OS matrix (advisory on `after`) |
| Performance bench | `perf-bench.yml` | push to `next`, pull_request to `next` | Measures latency per-operation against per-operation thresholds |
| Release (next) | `release.yml` | push of `v*` tag from `next` | Builds and publishes regular release artifacts |
| Release Please (next) | `release-please-next.yml` | push to `next` | Maintains release-PR with version bump and CHANGELOG for upcoming release |
| Release Please (before) | `release-please-before.yml` | push to `before` | Maintains release-PR with patch version bump for backport release |
| Auto-merge Dependabot | `auto-merge-dependabot.yml` | dependabot PR opened | Auto-merges minor/patch updates for permitted ecosystems |
| Post-release sync | `post-release-sync.yml` | release published from `next` | Merges `next` → `main` to keep stable branch in sync |

No deploy workflows. shy is distributed via GitHub Releases (for
`curl | bash` install) and via OS packages (`.deb`/`.rpm` built by
GoReleaser and uploaded to the release). No external infrastructure
to deploy to.

## CI workflow

**File:** `.github/workflows/ci.yml`

**Trigger:** every push to `next`; every pull_request targeting
`next` or `main`.

**Jobs:**

### `go-test`

- **Runs on:** `ubuntu-latest`
- **Steps:**
  1. Checkout
  2. Setup Go (version from `go.mod`)
  3. `go vet ./...`
  4. `go test -race -coverprofile=coverage.out ./...`
  5. Upload coverage as artifact (optional)
- **Required for merge:** yes (on PRs touching `cli/**`, `go.mod`,
  `go.sum`)

### `go-lint`

- **Runs on:** `ubuntu-latest`
- **Steps:**
  1. Checkout
  2. Setup Go
  3. Run `golangci-lint` with `.golangci.yml` config
- **Required for merge:** yes (on PRs touching `cli/**`)

### `shell-lint`

- **Runs on:** `ubuntu-latest`
- **Steps:**
  1. Checkout
  2. Install `shellcheck` (`apt-get install shellcheck`)
  3. Run `shellcheck install.sh init/init.bash`
- **Required for merge:** yes (on PRs touching `install.sh`,
  `init/**`)

### `goreleaser-check`

- **Runs on:** `ubuntu-latest`
- **Steps:**
  1. Checkout
  2. Run `goreleaser check`
- **Required for merge:** yes (on PRs touching `.goreleaser.yaml`)

**Path-based skip.** Each job has a `paths-ignore` or `paths`
filter so docs-only PRs skip Go/shell jobs entirely. This keeps CI
fast for trivial changes.

## Performance benchmark workflow

**File:** `.github/workflows/perf-bench.yml`

**Trigger:** every push to `next`; every pull_request targeting
`next`.

**Purpose:** Measure latency per-operation against per-operation
thresholds. Granular metrics catch performance regression in
specific subsystems rather than masking it in a single aggregate
number.

**Per-operation thresholds:**

| Operation | Threshold | UX zone |
|---|---|---|
| Tab-completion roundtrip (`shy <tab>`) | <50ms | Doherty instant |
| Native subcommand (`shy list`, `shy info`) | <100ms | Nielsen direct response |
| Plugin-dispatch overhead (v1, no sandbox) | <50ms | Doherty instant |
| Plugin-dispatch overhead (v2, with sandbox) | <100ms | Nielsen direct response |
| `init.bash` sourcing at shell startup | <200ms | Acceptable for workspace startup |
| `shy install <local-path>` | <500ms | I/O-bound, generous |
| `shy publish <name>` (no push) | <1000ms | Involves git operations |

**Aggregate threshold:** 200ms for the median across all measured
operations as a smoke-test sanity check. If aggregate exceeds
200ms but no individual operation exceeds its specific threshold,
investigate why the distribution shifted.

**Implementation:**

- A `bench/` directory in the Go source contains separate
  benchmark functions per operation.
- The workflow runs `go test -bench=. -benchtime=20x ./bench/...`
  to get 20 measurements per operation.
- A small parser reads the output, computes median and 95th
  percentile per operation, compares against thresholds defined
  in `bench/thresholds.json`.
- Failure output names the specific operation and the measured
  vs. expected: `tab-completion: 78ms (threshold 50ms)`.
- Trend reports written to a benchmark-result artifact stored on
  the workflow run; allows tracking drift over time.

**Required for merge:** yes (on PRs touching `cli/internal/cmd/**`,
`cli/internal/manifest/**`, `cli/internal/install/**`).

**Threshold maintenance:** Thresholds live in `bench/thresholds.
json` and can be edited via PR. Loosening a threshold requires
operator review with explanation in the PR description (why is
this slower now? is it justified?). Tightening a threshold is
always welcome.

## Acceptance workflow

**File:** `.github/workflows/acceptance.yml`

**Trigger:** every push to `next` and `before`; every pull_request
targeting `next` or `before`; weekly schedule (Sunday 06:00 UTC)
to catch upstream OS image drift. Also runs on `after` but in
**advisory mode** — results reported but not blocking.

**Purpose:** End-to-end installation and behavior validation
across a matrix of operating systems. Acts as the automated
counterpart to the manual fresh-VM acceptance test in
`02-long-horizon.md` Phase 9. Catches "works on my machine"
bugs and OS-specific path or shell issues that unit tests miss.

**Matrix:**

| OS | Container image | Shell | Notes |
|---|---|---|---|
| Ubuntu 22.04 | `ubuntu:22.04` | bash 5.1 | Primary target |
| Ubuntu 24.04 | `ubuntu:24.04` | bash 5.2 | Current LTS |
| Debian 12 | `debian:12` | bash 5.2 | Stable target |
| Fedora 40 | `fedora:40` | bash 5.2 | RPM target |
| macOS (host runner) | n/a — runs directly on `macos-latest` | bash 3.2 (system) + zsh 5.x | macOS bash compatibility (intentionally exercises the 3.2 fallback path) |

The container-based jobs run on `ubuntu-latest` and use `docker run
--rm` per test invocation, ensuring a fresh state every time.
The macOS job runs directly on `macos-latest` because Docker on
macOS runners is significantly slower and the goal is to test
actual macOS behavior.

**Steps per matrix entry:**

1. Build shy binary for the matrix OS (cross-compile from
   `ubuntu-latest`, except macOS which builds natively)
2. Start fresh container (or use macOS host)
3. Install dependencies: `curl`, `git`, `tar`, `sha256sum`
4. Run **happy-path acceptance script** (see below)
5. Run **negative-path acceptance script** (see below)
6. Capture logs; upload as workflow artifact if any step fails

**Happy-path script:**

```bash
# Provided binary is in /tmp/shy
mv /tmp/shy /usr/local/bin/shy
shy init
shy alias 'll=ls -alh'
shy completion add gh    # if gh installed; skip otherwise
shy list --sources       # expect: 1 alias from local source

# Source the init.bash and verify
. ~/.shy/init.bash
ll                       # expect: detailed directory listing

shy --version            # expect: version string
shy <tab>                # expect: subcommand completions (via expect/manual)
```

**Negative-path script:**

```bash
# Expect: exit code != 0 with informative message
sudo shy init                                 # refuses with hint
shy publish nonexistent-item                  # "no such item" error
shy install /tmp/malformed-manifest/          # parse error with location
shy collection subscribe github:nonexistent/repo  # graceful network-failure error
SHY_HOME=/proc/nonexistent shy init           # filesystem error, handled
shy install @user/repo --version invalid      # semver validation error
```

Each negative-path test asserts both that exit code is non-zero
*and* that stderr contains a recognizable diagnostic phrase. Tests
that pass silently on error are themselves failures (false
positives).

**Required for merge:** yes — `acceptance/ubuntu-22.04`,
`acceptance/ubuntu-24.04`, `acceptance/debian-12`,
`acceptance/fedora-40`, `acceptance/macos` are required status
checks on `next`. Weekly scheduled runs do not block merges but
notify the operator on failure.

**Runtime budget:** ~5 min per matrix entry × 5 entries = 25 min
total. Runs in parallel, so wall-clock ≈ 5-6 min per CI cycle.

**Adding a new OS to the matrix:** edit the matrix block in
`acceptance.yml`, add a happy-path/negative-path verification on a
local environment of that OS first, then enable in CI. New entries
default to "not required for merge" until the operator confirms
they pass consistently for a week.

## Release workflow

**File:** `.github/workflows/release.yml`

**Trigger:** push of a tag matching `v*` (created by release-
please when its release-PR is merged).

**Steps:**

1. Checkout (with full history; `fetch-depth: 0`)
2. Setup Go (version from `go.mod`)
3. Run `cobra.GenManTree()` via `go run ./cli/cmd gen-man /tmp/
   shy-man/` to generate man-pages
4. Run `goreleaser release --clean`
5. GoReleaser:
   - Builds binaries for `linux/amd64`, `linux/arm64`,
     `darwin/amd64`, `darwin/arm64`
   - Bundles binaries into `shy_<version>_<os>_<arch>.tar.gz`
   - Generates per-asset SHA256 files
   - Builds `.deb` and `.rpm` packages (nfpms) with man-pages
   - Uploads all assets to the GitHub Release for the tag
6. The release published with the changelog generated by release-
   please

**Permissions:** `contents: write` (to publish release), `id-
token: write` (if signing is added later).

## Release Please workflows

Two separate release-please workflows, one per release-producing
branch. Both use the `googleapis/release-please-action@v4` action
with different configuration.

### Release Please for `next` (regular releases)

**File:** `.github/workflows/release-please-next.yml`

**Trigger:** every push to `next`.

**Steps:**

1. Run `googleapis/release-please-action@v4` with
   `release-please-config.json` and `.release-please-manifest.
   json`.
2. The action reads commit history since the last release tag
   reachable from `next`, classifies commits by type (Conventional
   Commits), computes the next version, and updates or creates a
   release-PR on `next`.
3. When the release-PR is merged, the action creates the tag
   (which triggers the Release workflow).

**Config files:**

- `release-please-config.json` — package configuration (single Go
  package, no monorepo)
- `.release-please-manifest.json` — current version tracked (e.g.
  `{"": "0.1.0"}`)

**Permissions:** `contents: write`, `pull-requests: write`.

### Release Please for `before` (backport releases)

**File:** `.github/workflows/release-please-before.yml`

**Trigger:** every push to `before`.

**Steps:**

1. Run `googleapis/release-please-action@v4` with
   `release-please-config.json` (same config) but operating against
   the `before` branch.
2. The action reads commit history since the last release tag
   reachable from `before` (typically a `v1.0.x` tag). Computes a
   patch bump (`v1.0.x+1`) from the cherry-picked fix commits.
3. Updates or creates a release-PR on `before`.
4. When the release-PR is merged, the action tags and triggers the
   Release workflow against the `before` branch.

The config is intentionally the same JSON file — the difference is
the branch context the action runs in. Release-please's algorithm
naturally produces patch releases from cherry-picked fix commits
on an older line.

**Permissions:** same as the `next` workflow.

**When `before` is inactive.** When no backport work is in flight,
the `before` branch may not exist or may be dormant. The workflow
still resides in `.github/workflows/` and is harmless when no
pushes occur. No cleanup needed between backport efforts.

## Auto-merge Dependabot workflow

**File:** `.github/workflows/auto-merge-dependabot.yml`

**Trigger:** Dependabot opens or updates a PR.

**Logic:**

1. Check if PR is from Dependabot (`github.actor ==
   'dependabot[bot]'`).
2. Fetch Dependabot metadata via
   `dependabot/fetch-metadata@v2`.
3. Filter to permitted ecosystems and bump types:
   - `gomod` ecosystem AND (`patch` OR `minor` bump) → eligible
   - `github-actions` ecosystem AND (`patch` OR `minor` bump) →
     eligible
   - All others → not eligible; skip auto-merge
4. For eligible PRs, enable auto-merge via `gh pr merge --auto
   --squash`.
5. Auto-merge waits for CI to pass before completing the merge;
   if CI fails, the PR remains open for the operator to
   investigate.

**Permissions:** `contents: write`, `pull-requests: write`.

## Post-release sync workflow

**File:** `.github/workflows/post-release-sync.yml`

**Trigger:** release published (release-please published a new
release).

**Steps:**

1. Checkout `main`
2. Fast-forward merge `next` → `main` (the release-please workflow
   tagged `next` HEAD; merging `next` to `main` ensures `main` is
   identical to the released state)
3. Push `main`

**Why:** `main` is the stable branch shown by default on GitHub.
Without this sync, `main` would lag behind `next` and the README
or files seen by visitors would be stale relative to the released
version.

**Permissions:** `contents: write`.

## Branch protection enforcement

The CI workflow's job names are the **required status checks**
configured in `branch-protection.json`. The protection ruleset
differs per branch.

**For `next` (strict, hybrid path-based):**

- `go-test` (when paths-changed match production paths)
- `go-lint` (when paths-changed match `cli/**`)
- `shell-lint` (when paths-changed match shell files)
- `goreleaser-check` (when paths-changed match GoReleaser config)
- `perf-bench/*` (per-operation thresholds, when paths-changed
  affect benchmarked subsystems)
- `acceptance/ubuntu-22.04`, `acceptance/ubuntu-24.04`,
  `acceptance/debian-12`, `acceptance/fedora-40`, `acceptance/macos`
  (always required on PRs touching production code)

**For `main` (strict, automation-only):**

- All of the above
- `release-please` PR from `next` is the only entry point; manual
  PRs rejected.

**For `after` (relaxed, experimental):**

- `go-test`, `go-lint`, `shell-lint` required (basic sanity)
- `goreleaser-check` advisory (still runs but doesn't block)
- `perf-bench/*` advisory (still runs but doesn't block; expect
  regression during experimental work)
- `acceptance/*` advisory (runs but doesn't block)

Direct push permitted on `after` for the operator. External
contributors still use PRs but with relaxed requirements.

**For `before` (strict, backport-focused):**

- Full required-status-checks like `next`, PLUS full acceptance
  matrix as **required** (not advisory).
- No direct push, even for operator. All backports via PR for
  audit trail.
- Required-for-merge: full CI, full acceptance, perf-bench, and
  manual approval from operator.

**Note on `after` and `before` non-existence.** When these
branches don't exist (default state at v0.x start), no protection
rules need to be configured. The bootstrap `branch-protection.
json` includes rules for all four branches; the operator applies
each ruleset when the corresponding branch is created.

## Security scanning

**Dependabot vulnerability alerts.** GitHub's built-in scanner
runs on every dependency. Critical alerts trigger advisory emails
to the operator.

**CodeQL.** Optional, deferred to v1.x. The repository is small
and Go-only; CodeQL gives diminishing returns at the v1.0 stage.
Add when external contribution volume warrants automated security
review.

**Container scanning, SAST tools, etc.** Out of scope. shy is a
CLI binary, not a service. The threat model documented in
`01-whitepaper.md` does not motivate further scanning beyond
Dependabot.

## Local development parity

The CI checks should be reproducible locally:

```bash
go vet ./...
go test -race ./...
golangci-lint run --config .golangci.yml ./...
shellcheck install.sh init/init.bash
goreleaser check
goreleaser build --snapshot --clean   # to test release builds
```

A `Makefile` or `justfile` can wrap these for convenience but is
not required. The operator can run them ad hoc.

## CI minutes budget

For a private repo on the GitHub Pro plan, included minutes are
3,000/month. Estimated burn:

- CI on every push to `next`/`after`/`before` and every PR:
  ~3 min × ~60 events/month = 180 min
- Acceptance (5 matrix entries × ~5 min each, parallel): ~5 min
  wall-clock × ~50 events/month = 250 billed minutes
- Weekly scheduled acceptance run: ~5 min × 4 = 20 min
- Performance benchmark: ~2 min × ~50 events/month = 100 min
- Release workflow: ~10 min × ~4 releases/month = 40 min
- Release-please (next + before): ~30 sec × ~60 events/month
  = 30 min
- Auto-merge Dependabot: negligible
- Post-release sync: ~30 sec × ~4 releases/month = 2 min

**Total ~620 min/month** — well within the 3,000 min/month
budget. Acceptance matrix is the largest single consumer; if
budget tightens:
- Reduce matrix to Ubuntu 22.04 + Fedora 40 + macOS during regular
  CI; run full matrix only on weekly schedule
- Skip acceptance on `after` (it's advisory anyway)
- Skip perf-bench on docs-only PRs

Once the repository becomes public at v1.0, CI minutes for public
repos are unlimited and this section becomes informational only.

## Observability

**Build logs.** GitHub Actions retains 90 days of logs by default.
This is sufficient for post-mortem investigations.

**Release artifacts.** GitHub Releases retains artifacts
permanently. No external storage needed.

**Metrics.** Not collected. No telemetry from shy itself, no
analytics on releases (download counts are visible in GitHub
Releases UI; that's enough).

## Failure modes

**CI fails on PR.** Operator investigates locally with the same
commands. If transient, re-run; if real, fix in a new commit on
the PR branch.

**Release workflow fails after tag.** This is the worst case
because the tag exists but the release doesn't. Recovery:

1. Delete the tag locally and remotely: `git tag -d v<version> &&
   git push --delete origin v<version>`
2. Revert the release-please commit on `next`
3. Investigate the failure; fix forward
4. Allow release-please to retry on next push

**Release-please fails to update its PR.** Usually due to merge
conflicts in CHANGELOG. Operator resolves manually or closes the
PR and re-runs the workflow.

**Auto-merge merges something it shouldn't.** Rare but possible if
CI is misconfigured. Mitigation: the post-merge state is fully
reversible (`git revert`), and Dependabot bumps are isolated
commits — they revert cleanly.

**Post-release sync fails.** `main` stays stale. Operator manually
merges `next` → `main` and pushes.
