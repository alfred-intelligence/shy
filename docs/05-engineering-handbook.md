# shy — Engineering Handbook

Operational rules for the shy repository. Covers licence, branch
strategy, commit conventions, PR process, release process, and
maintenance routine. Decisions are concrete; rationale is brief.

Where this document and `04-agent-instructions.md` overlap, this
document is authoritative for *what happens at the repo level*;
`04` is authoritative for *what an AI agent should and shouldn't
propose*.

## Licence

**MPL-2.0** (Mozilla Public License 2.0).

File-level copyleft: modifications to MPL-licensed files must be
distributed under MPL-2.0. Non-MPL files in the same project
(plugins, integrations, downstream packaging) are not affected.

Concretely:

- The `shy` binary's source code is MPL-2.0.
- A plugin published as `@operator/some-plugin` is its own
  repository with its own licence. The author chooses.
- A collection (`@operator/some-collection`) is its own repository
  with its own licence; references between collections do not
  propagate MPL-2.0.

The `LICENSE` file at the repo root contains the full MPL-2.0
text. Every Go source file has a short `// SPDX-License-
Identifier: MPL-2.0` header for tooling.

**Documentation licence: CC-BY-SA 4.0.**

The whitepaper, design documents, and other documentation in the
repository (markdown under `docs/`, design files under
`shy-design/` or equivalent) are licensed under Creative Commons
Attribution-ShareAlike 4.0 International (CC-BY-SA 4.0). This is
symmetric with MPL-2.0's copyleft principle: work derived from
shy's documentation must remain under the same licence and credit
the source.

The `LICENSE-docs` file at the repo root contains the CC-BY-SA 4.0
text or links to https://creativecommons.org/licenses/by-sa/4.0/.
A short notice at the top of `docs/` (and equivalent design
directories) clarifies which licence applies.

## Branch strategy

**Four-branch time-sequenced model.**

Each branch represents a position on the time-axis of the project:

- **`main`** — stable. Reflects the latest tagged release. Default
  branch shown on GitHub. Read-only in practice: only the post-
  release automation pushes to it (and operator overrides for
  emergencies).
- **`next`** — active development for the *next* upcoming release.
  PRs target `next`. Release-please runs on `next` and maintains a
  release-PR with the pending changelog and version bump.
- **`after`** — experimental work for releases *after* the next
  one. Long-horizon features that aren't ready for the upcoming
  release land here. Code here may not stabilise for months. No
  release-please; no automatic version bumping. Cherry-picked into
  `next` when ready.
- **`before`** — backport branch for fixes that need to land in
  earlier stable releases (e.g., a CVE fix that must be applied to
  `v1.0.x` after `v1.1` has shipped). Release-please runs on
  `before` and tags `v<major>.<minor>.<patch+1>` releases off it.
  Inactive when no older releases need maintenance.

The branches don't all need to exist from day one:

- **`main`** + **`next`** exist from project start (Phase 1 in
  long-horizon).
- **`after`** is created when v2-workspace-arbete (or any v2+
  feature work that runs in parallel with v1.x maintenance)
  begins. Practically: at the start of Phase 1, since shy v2 work
  may begin before shy v1.0 ships.
- **`before`** is created only when v1.0 has shipped *and* a
  bugfix needs to be backported. It is short-lived per backport
  effort and can be deleted between backports if desired.

Lifecycle of a change (normal path):

1. Feature branch (`feat/<scope>` or `fix/<scope>`) branches from
   `next`.
2. PR opened against `next` with Conventional Commits in the
   commit messages.
3. CI runs. On green and approval (path-dependent), PR merges to
   `next`.
4. Release-please observes the merged commits and updates its
   release-PR with a new version bump and changelog entry.
5. When the operator merges the release-PR, release-please tags a
   new release, GoReleaser builds artifacts, and a post-release
   workflow merges `next` → `main` automatically.
6. `main` now reflects the new release; `next` continues
   development.

Lifecycle of long-horizon work (using `after`):

1. Feature branch (`feat/<scope>`) branches from `after`.
2. PR opened against `after`. CI runs but is less strict
   (acceptance tests are advisory, not blocking).
3. Work iterates on `after` over weeks or months.
4. When ready for release, the operator cherry-picks (or merges
   with care) the work from `after` into `next`. Release-please
   then handles versioning normally.
5. `after` continues with its own diverged history; periodic
   rebases against `next` may be needed to avoid drift.

Lifecycle of a backport (using `before`):

1. A fix is needed for an older stable release (e.g., a CVE
   discovered in `v1.0.2` after `v1.1.0` has shipped).
2. Operator creates or revives `before` branched from the relevant
   stable tag (`git checkout -b before v1.0.2`).
3. Cherry-picks the fix commit from `next` (where the fix was
   originally landed).
4. PR opened against `before` with the cherry-picked fix.
5. CI runs (full CI + acceptance) against the cherry-picked
   version.
6. Release-please on `before` generates a patch release
   (`v1.0.3`).
7. Operator merges the release-PR; release-please tags and
   releases.
8. `before` can be deleted or kept dormant until the next backport
   need.

**Cross-branch cherry-pick policy.** When a fix is needed in
multiple branches, operator discipline determines flow. Typical
patterns:

- Bug in `next` that should also be in `before`: cherry-pick to
  `before` after fix lands in `next`.
- Feature in `after` that should also land in `next`: cherry-pick
  (or careful merge) when ready. Avoid merging entire `after`
  into `next` — that pulls in unfinished work.
- Documentation fix: usually only `next`. Backporting docs to
  `before` is allowed but optional.

This is operator discipline, not automation. Document the
decisions in PR descriptions for audit trail.

## Pre-1.0 phase (v0.x) operator privileges

During v0.x development (before v1.0 is tagged), strict branch
protection is **not** enforced for the operator. This is a
practical concession to the bootstrapping phase: the operator
needs flexibility to push design documents, scaffolding, and CI
configuration directly to `main` without ceremony.

Concretely during v0.x:

- The operator may push documentation and scaffolding directly to
  `main`. This is allowed because no formal release exists yet and
  the `next` → `main` synchronisation that protects releases hasn't
  begun.
- Branch protection rules described in this document apply to
  external contributors and to merge-via-PR enforcement; they do
  not block operator direct-push during v0.x.
- The `next` branch becomes the dev target as soon as `v0.2-pre`
  (or whatever pre-release exists) is in place and release-please
  starts maintaining a release-PR there.

**From v1.0 onward**, strict protection is enforced. The operator
loses direct-push privileges on `main`. All changes to `main` flow
via the post-release-sync automation. The operator can still push
docs directly to `next` (via path-based exemption — see below) but
not to `main`.

This is explicit operator privilege during v0.x, not a permanent
override. The intent is to support the bootstrapping phase without
fighting branch protection over every initial commit.

## Path-based branch protection

GitHub's branch protection rules are configured per branch with
required status checks and required reviews. The operator is the
sole reviewer in `solo+contrib`; reviews are self-approved when
operator opens the PR.

**`main` protection (strict):**

- No direct pushes from anyone except the post-release automation
  (GitHub Actions bot using a workflow token).
- All changes via PR from `next` only.
- Required status checks: `ci`, `release-please` validation.

**`next` protection (hybrid path-based):**

PRs targeting `next` require passing CI for changes to
**production paths**:

```
cli/**
init/**
install.sh
.goreleaser.yaml
release-please-config.json
.release-please-manifest.json
go.mod
go.sum
```

Direct push to `next` is permitted for **non-production paths**:

```
*.md
docs/**
.github/workflows/**
.github/dependabot.yml
.github/labels.json
.github/PULL_REQUEST_TEMPLATE.md
.github/ISSUE_TEMPLATE/**
.editorconfig
.gitignore
```

This means:

- Code changes always go through PR + CI.
- Docs, CI workflows themselves, and meta-files can be pushed
  directly by the operator for trivia.
- External contributors always use PR regardless of path (they
  don't have push permission).

**`after` protection (relaxed):**

PRs targeting `after` require CI but acceptance tests are
*advisory* — they run and report but don't block merge. The intent
is experimental velocity: try things, observe behavior, iterate.

Required status checks: `go-test`, `go-lint`, `shell-lint`.
Acceptance tests run for visibility but failure does not block.

Direct push is permitted for all paths on `after` — it is the
operator's experimental playground.

**`before` protection (strict):**

PRs targeting `before` require passing CI **and** acceptance tests
across the OS matrix. The whole point of `before` is releasing
backport fixes that must be at least as well-tested as the
original release.

Required status checks: full CI + full acceptance matrix.

Direct push is **not** permitted on `before` even for the
operator. All backports go via PR for traceability — when a CVE
is being backported, the audit trail must be complete.

**Note on `after` and `before` non-existence.** When these branches
don't exist (which is the default state until they're needed), no
protection rules need to be configured. The bootstrap
`branch-protection.json` includes rules for all four branches; the
operator applies the rules selectively via `gh api` when each
branch is created.

The `branch-protection.json` in bootstrap implements all four
configurations.

## Commit conventions

**Conventional Commits.** Format:

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

Types:

| Type | Bumps version | Notes |
|---|---|---|
| `feat` | minor | New user-visible capability |
| `fix` | patch | Bug fix |
| `feat!` or `BREAKING CHANGE:` in footer | major | Breaking change |
| `docs` | no | Documentation only |
| `chore` | no | Tooling, deps, repo hygiene |
| `refactor` | no | Code restructure, no behaviour change |
| `test` | no | Test-only changes |
| `ci` | no | CI/CD changes |
| `perf` | patch | Performance fix that's also a fix |
| `style` | no | Formatting, no behaviour change |

Scope is optional but encouraged for projects with clear sub-areas
(`feat(install):`, `fix(parser):`, `docs(whitepaper):`).

**Rules:**

- Subject ≤ 72 characters, present tense, imperative mood ("add",
  not "added" or "adds").
- Subject describes **effect**, not implementation. "Add alias
  subcommand" is right; "implement alias using cobra" is wrong.
- Body, when present, explains *why*. Reviewer should not have to
  guess motivation.
- One sentence in the subject is enough for most commits; reserve
  body for non-trivial reasoning.

This matches `04-agent-instructions.md` § Code style.

## PR process

**Size.** Small enough that a reviewer can read it in one sitting.
Hard rule: never combine refactor + feature in one PR. The
reviewer can't tell what's behavior change versus tidying.

**Description.** Use the PR template (`bootstrap/.github/
PULL_REQUEST_TEMPLATE.md`). Mandatory sections:

- What changed (effect)
- Why (motivation)
- How to verify (concrete steps for the reviewer)
- Linked issues (`Closes #N`)

**Review.** In `solo+contrib`, operator self-approves operator's
PRs. External PRs require operator's explicit approval. CI must
pass before merge regardless of who opened the PR.

**Merge strategy.** **Squash merge** for feature PRs (cleaner
history; release-please reads squashed commits). **Rebase merge**
for trivial fix PRs where preserving the commit chain helps. **No
merge commits** on `next` (linear history is required for release-
please to function correctly).

**Stale PRs.** If a PR sits without progress for 60 days, the
operator either picks it up, closes it with a "stale" label, or
explicitly extends the timeline. The "stale" label is for triage,
not punishment.

## Release process

**Automated via release-please + GoReleaser.**

Step by step:

1. Conventional Commits accumulate on `next`.
2. Release-please workflow runs on every push to `next`. It
   maintains a release-PR titled `chore: release <version>` with
   the proposed CHANGELOG entry and version bump.
3. Operator reviews the release-PR. Adjusts CHANGELOG wording or
   version manually if needed (release-please respects manual
   edits to the release-PR).
4. Operator merges the release-PR.
5. Release-please workflow detects the merge, tags the release
   (`v<version>`), and pushes the tag.
6. GoReleaser workflow triggers on the tag. Builds binaries for
   linux/darwin × amd64/arm64, builds `.deb` and `.rpm` packages,
   generates man-pages via `cobra.GenManTree()`, signs SHA256
   checksums.
7. Release published to GitHub Releases with all assets.
8. Post-release workflow (or release-please itself) merges `next`
   → `main` automatically, ensuring `main` reflects the release.

**Versioning.** Strict SemVer. Major for breaking changes (rare
post-1.0), minor for features, patch for fixes. Release-please
computes the bump from commit types; the operator overrides only
in exceptional cases.

**Manual override.** To force a specific version, the operator
edits the release-PR or pushes a commit with
`Release-As: <version>` in the body. Release-please honours both.

**Pre-1.0 versioning.** During v0.x development, breaking changes
are allowed without a major bump (this is standard SemVer
pre-1.0). After v1.0, all breaking changes require a major bump.

## Maintenance

**Dependabot** scans dependencies and opens PRs for updates.
Configuration is hybrid:

| Ecosystem | Strategy |
|---|---|
| `gomod` (Go dependencies) | Auto-merge on minor/patch when CI green |
| `github-actions` (workflow versions) | Auto-merge on minor/patch when CI green |
| Major version bumps (any ecosystem) | Manual review always |
| Other configurations or scripts | Manual review always |

Auto-merge uses the `auto-merge` workflow that watches Dependabot
PRs and enables auto-merge after CI success. The operator can
disable auto-merge per PR by removing the label.

**Security advisories** flow through:

1. Reporter opens a private advisory via
   `https://github.com/alfred-intelligence/shy/security/advisories/new`.
2. Operator receives notification.
3. Operator and reporter coordinate fix in a private fork (if
   needed) or branch.
4. Fix is published as a regular release with `[security]` tag in
   the relevant item's manifest (if shy itself is affected) or
   noted in CHANGELOG.
5. Advisory is published with CVE link if assigned.

**Issue triage** runs weekly when the project is private (v0.x)
and on demand once public. Operator reviews:

- New bug reports — reproduce or request reproduction
- Feature requests — file against the long-horizon roadmap
- Design questions — engage in discussion, no obligation to
  decide immediately

Labels (defined in `bootstrap/.github/labels.json`):

- `bug`, `feature`, `design-question`, `documentation`, `security`
- `priority:high`, `priority:medium`, `priority:low`
- `status:triage`, `status:in-progress`, `status:blocked`,
  `status:stale`
- `good-first-issue`, `help-wanted`
- `area:cli`, `area:init.bash`, `area:install.sh`, `area:plugins`,
  `area:collections`, `area:ci`

## Sudo policy

shy itself requires sudo for **only two commands**:

- `sudo shy system-install` — writes the `/etc/skel/.shy/` seed for
  new users.
- `sudo shy system-uninstall` — removes the seed.

Every other shy command is user-level. The binary actively refuses
to run as root for commands that would write to user-area paths
(e.g., `shy init`). Running `sudo shy init` prints an error
pointing to `shy system-install` instead.

This is enforced at the cobra subcommand level: each subcommand
declares whether it requires root, refuses root, or is agnostic.
The default is "refuses root" — explicit opt-in for the few
commands that need it.

**Sudo for binary installation is separate.** When the operator
installs the binary via `sudo apt install shy` or
`sudo curl | bash`, the sudo is required by the package manager
or the install script, not by shy itself. shy's own sudo
requirements are limited to system-install/uninstall.

## Repository hygiene

**Default branch.** `main` (GitHub default; reflects stable).

**Branch deletion.** Feature branches deleted automatically on
PR merge. The operator does not maintain old feature branches.

**Forks.** External contributors fork the repo and PR back. No
special handling.

**CI minutes.** GitHub Actions on private repos uses metered
minutes. Workflows are tuned for efficiency: matrix tests run only
when production paths change; docs-only changes skip Go tests.

**Tags.** Created exclusively by release-please. No manual tags.
If a tag is needed for testing (`v0.1.0-draft`), it is created
once and never reused.
