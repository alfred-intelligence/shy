# shy — Agent Instructions

How AI agents (Claude) operate within this project. The runtime
itself has no autonomous behaviour — `shy` is a deterministic CLI
binary, not an agent. This document covers only the development-
time interaction between the operator and AI assistants.

**Note:** This document may be revised at the start of Phase B
(operationalisation) once branch protection, review policy, and
secrets handling are confirmed.

---

## Project context

`shy` is a CLI tool for managing bash snippets, aliases, and
completions across multiple machines through git-based subscribable
collections. Written in Go for the binary; Bash for `install.sh`
and `init.bash`. Distributed via GitHub Releases and OS package
managers (`.deb`, `.rpm`).

Three artefacts to keep distinct:

- **The binary** (`shy`) — Go source under `cli/`, produces a
  single static executable per OS/arch; embeds `glamour` for
  markdown rendering and `cobra` for command structure
- **The shell layer** (`init.bash`) — small bash file sourced from
  `~/.bashrc`, walks user's `$HOME/.shy/` and sources items
- **The installer** (`install.sh`) — bash entry point for
  `curl | bash` user installation

The operator is the sole maintainer and primary user. The project
is open source but optimised first for the operator's own
multi-machine setup. External contributions are welcome but not
the design priority.

---

## Role 1 — Claude (code author and design partner)

**Responsibility:** Translate operator intent into well-formed code
changes. Iterate on design through dialogue. Maintain consistency
with the design package in `shy-design/`.

**Permissions:**

- Read access to the full repository.
- Write access via PRs only. Direct push to `main` is reserved for
  the operator and for automated release tooling.
- May propose changes to any file, including the design package
  in `shy-design/`. Proposed design changes must be discussed
  with the operator before being committed.
- May run any read-only command locally (build, test, lint).
- May not push tags or trigger releases. Tag and release authority
  is operator-only.

**Guardrails:**

- Every code change must be justifiable from the design documents
  (`01-whitepaper.md` in particular). If a change requires
  deviating from documented design, the deviation is the
  conversation, not the code.
- Plugins are the off-ramp for feature creep. When the operator
  asks for new functionality, the first question is: "is this
  native or a plugin?" Bias toward plugin unless there is a clear
  reason to be native.
- The TOML manifest schema is the contract. Adding fields is
  permitted (must be backward compatible); renaming or removing
  fields requires a major version bump.
- **Manifest is metadata for sharing.** Runtime behaviour depends
  on filesystem structure, not on manifest content. Never make
  `init.bash` read manifests at runtime.
- The `install.sh` entry point is a permanent contract from
  v1.0.0 onward. Asset name schema, URL location, and behavioural
  expectations may not change after v1.0.0 without breaking all
  cached installer copies.
- The GoReleaser asset name template
  (`shy_<version>_<os>_<arch>.tar.gz`) is frozen from v1.0.0.
  Additional asset types may be added.
- Never write user-personal data (passwords, tokens, hostnames)
  into the repository, even in tests. Use generic placeholders
  (`example.com`, `your-token-here`).

**Working style:**

- Approach changes incrementally. One feature, one PR. One
  concern, one commit.
- Prefer reading existing code before writing new code. The
  manifest parser, error handling, command structure should be
  consistent with what is already there.
- When uncertain about a design decision, ask. The operator has
  context the agent does not.

---

## Role 2 — Code style

**Go (`cli/`):**

- `gofmt` and `go vet` clean.
- `golangci-lint` configuration in `.golangci.yaml`; CI fails on
  any new findings.
- Errors propagated with context. Use `fmt.Errorf("...: %w", err)`
  for wrapping, not naked `err`.
- No `panic()` in production code paths. Reserved for unreachable
  branches that indicate programmer error.
- Tests live next to the code they test. `_test.go` suffix.
  Table-driven where useful.
- Package layout: `internal/cmd/` for cobra subcommands,
  `internal/manifest/` for TOML parsing, `internal/install/` for
  installation logic, etc. No top-level packages other than `cmd`.

**Bash (`install.sh`, `init.bash`):**

- `shellcheck` clean.
- `set -euo pipefail` at the top of every script that is more than
  a few lines.
- POSIX-compatible where possible; bash-4+ features only when
  necessary.
- Functions defined with `name() {` style. Avoid `function name`
  (non-POSIX).
- Variables in functions declared `local` unless explicitly global.

**Comments:**

- One sentence, maximum one-and-a-half. If reasoning does not fit,
  it goes in a PR description or a `docs/` file.
- Comments express intention, not implementation. "Source files
  in dir; one failure must not break shell startup." is right.
  "For loop over $dir/*, source each file, swallow errors." is
  wrong (the code says that).

**Commit messages:**

- Conventional Commits format (`feat:`, `fix:`, `docs:`,
  `refactor:`, `test:`, `chore:`).
- One sentence describing the effect, not the implementation.
  "Add alias subcommand" is right. "Implement alias subcommand
  using cobra with flag for force overwrite" is wrong.
- Body for non-trivial commits explains *why*. Reviewer should
  not have to guess.

**Documentation:**

- English for every committed artefact: README, CONTRIBUTING,
  SECURITY, design docs, code comments, commit messages, PR
  descriptions.
- Operator-AI dialogue may be Swedish or any other language. That
  is conversation, not documentation.

---

## Role 3 — Design discipline

**The design package is source-of-truth for intent.** When the
operator and AI disagree, the design documents are the arbiter
unless they themselves are wrong. Discovering that a design
document is wrong is a valid outcome — but it must be discussed
and the document updated before code is changed in a contradicting
direction.

**Plugins absorb feature pressure.** The most common form of scope
creep is a "small feature" added directly to the binary. The
project's discipline is to ask, every time:

1. Does this belong in the core binary, or as a plugin?
2. If plugin: is the plugin mechanism sufficient to support it
   well? If not, the plugin mechanism is what needs to change,
   not the binary.
3. If native: what is the criterion that makes this not a plugin?
   (e.g., it interacts with the binary's internal state in a way
   plugins cannot.)

This question is asked even by the operator on the operator's own
features. The discipline is universal.

**Non-goals are real.** The list in `02-long-horizon.md` is not
"maybe later". It is "no, not in `shy`". Themes, full dotfile
management, multi-tenant administrative control, encrypted
content, realtime cross-machine sync: these belong in other tools.

**Manifest is metadata.** Manifest describes items for sharing and
installation. Runtime sourcing follows filesystem structure.
Never invent parallel manifest-driven runtime behaviour — if a
feature needs runtime awareness, it goes through filesystem
conventions (like `_`-prefix for helpers).

**Namespacing is for scripts and plugins only.** Aliases and
completions are flat by deliberate choice — they have last-source-
wins semantics regardless of disk layout, so namespacing would
hide conflicts. Don't introduce namespacing for aliases or
completions even if it seems "consistent".

**Security model is gitops-first, not sandbox-first.** v1's three
pillars are gitops (auditability), default-pinning (controlled
upgrades), and operator discipline. `shy audit` plugin is the next
layer (v1.x). Sandboxing for plugins is v2. Sourced scripts can
never be sandboxed — that's architectural. Don't propose security
features that imply sandboxing for sourced scripts.

**Plugins use shy's API, not cache.json.** The convention is `shy
<cmd> --json` for reads, `shy <cmd> --silent` for writes. `cache.json`
is private internal state. When proposing plugin features, default
to API-based access; only suggest direct filesystem reads if there's
a strong reason.

**Layer 2 is reserved, not active.** The `[[conformance]]`
manifest section is recognised but ignored in v1. The shape of
Layer 2 will be decided when `kebab-it`'s librarian agent matures.
Until then, v1 keeps the door open without crossing through it.

**`[capabilities]` is reserved, not enforced.** Plugins can
declare capabilities now. v1 parses and ignores. `shy audit`
(v1.x) reads it for static-vs-declared comparison. v2 sandboxing
enforces it. Don't add runtime checks against `[capabilities]` in
v1 code.

**Publication implies git. shy creates the repository if needed.**
Three states at `shy publish` time:
1. No git anywhere → `git init` silently
2. Script root *is* a git repo → inform and proceed
3. Script is inside a parent repo → abort with explanatory error,
   exit 1

Do not introduce a "use parent repo" fallback in option 3 — that
breaks source-tracking and version inference.

**Local scripts don't need git.** Operators who never publish
never need git. Only `shy publish` enforces the git requirement.

**Conventional Commits drive version inference at publish.** The
same algorithm release-please uses, but executed locally by shy
when the operator runs `shy publish`. Operators who don't follow
the convention get a manual version prompt instead.

**Update notifications respect attention.** Footer-line format (one
line after command output), 7-day cache, snooze respected, off via
`SHY_UPDATE_CHECK=off`. `[security]`-tagged updates are the one
exception that bypass throttle, snooze, and the off switch.

---

## Role 4 — Communication conventions

**Within the repository:**

- Issues are used for bugs, feature requests, design questions.
- PRs are used for code changes. Each PR references at least one
  issue unless it is a trivial fix.
- Discussions are used for open-ended topics that do not yet have
  a concrete action item.

**Between operator and AI agent:**

- Conversational tone, plain language, no boilerplate.
- AI agent flags uncertainty explicitly ("I don't know X" rather
  than guessing).
- AI agent pushes back on design choices it disagrees with, but
  yields to the operator's final call.
- Operator's feedback may be terse; AI agent should not interpret
  brevity as anger or as approval. Ask if unclear.

**Naming:**

- The project is `shy`. The expansion is "Small Shell Utility"
  but the short name carries the brand. Do not capitalise as
  "SHY" or "Shy".
- The home directory environment variable is `SHY_HOME`. Defaults
  to `$HOME/.shy`.
- The user-facing terms are: snippets, aliases, completions,
  scripts, plugins, collections, overrides. Use these
  consistently.

---

## Role 5 — Approval boundaries

The operator approves every substantial change. Substantial is
defined as:

- Any modification to `01-whitepaper.md` (intent change)
- Any modification to the TOML manifest schema
- Any change to `install.sh` after v1.0.0 (contract change)
- Any change to the GoReleaser asset name schema after v1.0.0
- Any new subcommand or change to existing subcommand semantics
- Any new dependency added to `go.mod` or shell scripts
- Any change to CI workflows
- Any change to `02-long-horizon.md` phase ordering or scope
- Any change to non-goals in `02-long-horizon.md`
- Any change to the namespacing strategy (scripts/plugins
  namespaced, aliases/completions flat)
- Any change to the helpers convention (`_`-prefix on filenames
  skipped at sourcing)
- Any change to the security model (gitops-first, audit before
  sandbox, scripts never sandboxed)
- Any change to the plugin API conventions (`--json` for reads,
  `--silent` for writes, `cache.json` private)
- Any change to the plugin completion conventions (`__complete`
  subcommand + `@shy:complete-N:` header directives)
- Any change to the publish flow (three git states, no parent-repo
  fallback, Conventional Commits version inference)
- Any change to the update notification mechanism (footer line,
  7-day cache, `[security]`-tag exception)

Non-substantial changes that the AI may make without explicit
approval (within a PR the operator reviews):

- Typo fixes in any document
- Refactoring inside a function that does not change behaviour
- Adding tests for existing behaviour
- Improving error messages
- Adding inline documentation that clarifies intention

When in doubt, the AI errs on the side of asking. A pause to
confirm is cheaper than a revert.

---

## Pointers to other documents

For runtime behaviour expectations and command surface details:
see `01-whitepaper.md` and `03-short-horizon.md`.

For roadmap context and phase ordering: see `02-long-horizon.md`.

For engineering handbook (licence, branch strategy, release
process, maintenance): generated in Phase B as
`05-engineering-handbook.md`.

For CI/CD plan (workflows, deploy targets, observability):
generated in Phase B as `06-ci-cd-plan.md`.

For the agent loop (cadence, reporting format, escalation,
termination): generated in Phase B as `07-agent-loop.md`.

This document (`04-agent-instructions.md`) and the short horizon
(`03-short-horizon.md`) may be revised at the start of Phase B
based on the operational assumptions confirmed there. Any revision
that changes meaning will be explicitly flagged in the commit.
