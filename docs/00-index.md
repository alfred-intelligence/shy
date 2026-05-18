# shy — Design Package

This folder contains the full design package for `shy` (Small Shell
Utility), a CLI tool for managing bash snippets, aliases, and
completions with cross-machine portability through subscribable
collections.

## Documents

| File | Purpose | Audience |
|---|---|---|
| [`01-whitepaper.md`](01-whitepaper.md) | Product vision, problem, solution, architecture, design rationale | Everyone; prospective adopters |
| [`02-long-horizon.md`](02-long-horizon.md) | Phased roadmap across ~5 months; milestones and dependencies | Maintainer and contributors |
| [`03-short-horizon.md`](03-short-horizon.md) | Detailed step-by-step plan for v0.1 → v0.5; concrete commands | Implementers |
| [`04-agent-instructions.md`](04-agent-instructions.md) | How Claude and other AI agents work within this project | AI agents + operator |

## How to use this package

- **Starting fresh** — read `01` then `02` for orientation, then jump to `03` for the next concrete actions.
- **Returning after a break** — read `03` to locate where work stopped, then consult `01` for principles if a design question arises.
- **Reviewing an AI-authored change** — cross-check against `04` to confirm the agent operated within its guardrails.

## Maintenance

Documents are versioned with the project. Any non-trivial update to
`01-whitepaper.md` (product scope, architecture, security model)
should propagate to `02` and `03` within the same commit. Minor
corrections (typos, clarifications) can stand alone.

The design package is source-of-truth for **intent**. The actual code
and `README.md` in the `shy` repository are source-of-truth for
**behaviour**. When they diverge, file an issue and resolve
deliberately.

## Status

v0.1 — Phase A (product design) complete. Architecture, manifest
schema, command surface, plugin model, security boundaries, and
acceptance criteria for v1.0 are defined.

Documents `03` and `04` carry a header reminding the reader that they
may be revised at the start of Phase B (operationalisation) once the
repository exists and the silent operational assumptions captured in
`01-whitepaper.md` (Assumptions for Phase B) have been confirmed.

No repository pushed yet. Phase B begins when `github.com/GeGGe01/shy`
exists and the operator confirms in writing.
