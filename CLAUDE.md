# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository status

This repository currently contains **only design documents** under `docs/`. There is no source code, no build, no tests. Phase A (product design) is complete; Phase B (operationalisation) begins when `github.com/alfred-intelligence/shy` exists and the operator confirms.

Once code lands, the target tree is rooted at `cli/` (Go) plus `init/init.bash` and `install.sh` at the repo root. The toolchain is Go (binary), Bash (`init.bash`, `install.sh`), Cobra (CLI), Glamour (markdown), `pelletier/go-toml/v2` (manifests), GoReleaser, and release-please.

## Design package — entry points

Read in this order when orienting:

1. `docs/01-whitepaper.md` — product intent, architecture, manifest schema, security model. **Source of truth for intent.** Any change here cascades to `02` and `03`.
2. `docs/04-agent-instructions.md` — guardrails, approval boundaries, what counts as "substantial" enough to require operator approval. Read this before proposing non-trivial changes.
3. `docs/03-short-horizon.md` — concrete step-by-step plan for Phases 1–2 (skeleton → core commands). When implementing, work from a numbered step here.
4. `docs/02-long-horizon.md` — phase roadmap and non-goals. Phases are sequenced by capability dependencies; non-goals are real ("no, not in `shy`").
5. `docs/05-engineering-handbook.md`, `docs/06-ci-cd-plan.md`, `docs/07-agent-loop.md` — operational rules, CI workflows, agent interaction patterns.

## Load-bearing invariants

These come up constantly and are easy to violate by accident. All are derived from `docs/01-whitepaper.md` and `docs/04-agent-instructions.md`.

- **Manifest is metadata, not configuration.** `init.bash` walks the filesystem and sources `.sh` files; it **never** parses `manifest.toml` at runtime. The CLI reads manifests for `list`, `info`, `update`, `publish`. Runtime behaviour follows directory structure, not manifest content.
- **Namespacing applies to scripts and plugins only.** Aliases and completions are flat by deliberate choice — last-sourced wins regardless of disk layout, so namespacing would hide conflicts. Surface conflicts at install/subscribe time instead.
- **User layer is always above system layer at runtime.** `/usr/share/shy/` is a seed only; `shy init` copies from it skip-on-conflict. Overrides live in `$HOME/.shy/overrides.d/` (user-owned), never in `/etc/`.
- **`_`-prefixed files are private helpers.** Skipped by `init.bash`. Convention is independent of the manifest.
- **`install.sh` and the GoReleaser asset name (`shy_<version>_<os>_<arch>.tar.gz`) are permanent contracts from v1.0.0.** Additive changes only after that point.
- **`cache.json` is private.** Plugins read state via `shy <cmd> --json` and write via `shy <cmd> --silent`. Never expose `cache.json` as a plugin API.
- **`[capabilities]` and `[[conformance]]` are reserved in v1** — parsed and ignored. Don't enforce them in v1 code; the `shy audit` plugin (v1.x) consumes `[capabilities]` for static analysis, v2 sandboxing enforces it.
- **`shy publish` requires the script to be its own git repo root.** Three states (no git → `git init`; already a repo → proceed; inside parent repo → abort with exit 1). No "use parent repo" fallback.
- **Sourced scripts cannot be sandboxed in any version of shy.** Architectural limit. Operator discipline is the only security mechanism for sourced scripts. Don't propose sandboxing for them.
- **Plugins are the off-ramp for feature creep.** When asked for new functionality, the first question is "native or plugin?". Bias toward plugin unless there's a concrete reason it must be native.

## Approval boundaries

The operator approves substantial changes before they land. `docs/04-agent-instructions.md` § Role 5 lists what counts as substantial — including any manifest schema change, any new subcommand, any CI workflow change, any change to namespacing/security/publish/update-notification semantics. When in doubt, ask before changing.

Non-substantial (proceed within a PR the operator reviews): typo fixes, refactors that preserve behaviour, tests for existing behaviour, error-message improvements, intention-clarifying comments.

## Style (once code exists)

- **Go:** `gofmt`, `go vet`, `golangci-lint` clean. Errors wrap with `fmt.Errorf("...: %w", err)`. No `panic()` in production paths. Tests live next to code (`_test.go`). Package layout: `cli/internal/cmd/` (cobra subcommands), `cli/internal/manifest/`, `cli/internal/install/`, etc.
- **Bash:** `shellcheck` clean. `set -euo pipefail` at top. `name() {` not `function name`. `local` for function-scoped vars.
- **Comments:** one sentence, max one-and-a-half. Intention, not implementation. Longer reasoning goes in PR descriptions or `docs/`.
- **Commits:** Conventional Commits (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `perf`, `style`). Subject describes effect, not implementation. Body explains *why* for non-trivial changes.
- **Documentation language:** English everywhere committed. Operator dialogue may be Swedish.
- **Naming:** the project is `shy` (lowercase, never "SHY" or "Shy"). `SHY_HOME` defaults to `$HOME/.shy`. User-facing terms: snippets, aliases, completions, scripts, plugins, collections, overrides.

## Branch and release flow (when repo is live)

Stable/next pattern: `main` reflects the latest tagged release; `next` is the active development branch and the PR target. Release-please runs on `next` and maintains a release-PR; merging it tags a release, GoReleaser builds artefacts, and post-release automation merges `next` → `main`. Hotfixes go through `next` like everything else.
