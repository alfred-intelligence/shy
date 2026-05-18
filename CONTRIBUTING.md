# Contributing to shy

Thank you for considering a contribution.

shy is a `solo+contrib` project: the operator is the primary user and
the design authority, but external pull requests are welcome.

## Before you start

- Read [`docs/01-whitepaper.md`](docs/01-whitepaper.md) for the
  product intent and architecture.
- Read [`docs/04-agent-instructions.md`](docs/04-agent-instructions.md)
  if your change touches anything in the "Approval boundaries"
  section — those changes need operator sign-off before code.
- For substantial features, open an issue first to confirm scope.

## Where to send PRs

The development branch is `next`. PRs target `next`, not `main`.
`main` reflects the latest tagged release and is updated by
automation after each release.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/), please:

```
<type>(<scope>): <subject>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`,
`ci`, `perf`. The subject describes effect, not implementation
("add alias subcommand", not "implement alias using cobra").

release-please reads these to compute the next version, so accurate
types matter.

## Local checks

```bash
go vet ./...
go test -race ./...
shellcheck install.sh init/init.bash
goreleaser check
```

CI runs the same set on every PR; passing locally first saves a
round-trip.

## Style

- **Go**: `gofmt` clean. Wrap errors with `fmt.Errorf("...: %w", err)`.
  No `panic()` outside genuinely unreachable code. Tests live next
  to the code they test.
- **Bash**: shellcheck clean. `set -euo pipefail`. Functions
  declared `name() { ... }` (not `function name`); locals declared
  `local`.
- **Comments**: one sentence, max one-and-a-half. Explain WHY when
  it's non-obvious; let the code speak for WHAT.

## What's in scope

Bias toward plugins. Anything that can live as a plugin probably
should — the binary's surface area is deliberately small. See
`docs/04-agent-instructions.md` § "Plugins absorb feature pressure".

The non-goals list in [`docs/02-long-horizon.md`](docs/02-long-horizon.md)
is real, not "maybe later". Themes, full dotfile management,
multi-tenant administrative control, encrypted content, and
realtime cross-machine sync belong in other tools.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).

## Security

Please report vulnerabilities privately as described in
[`SECURITY.md`](SECURITY.md) — not as public issues.
