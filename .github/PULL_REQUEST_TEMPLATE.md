<!--
Thanks for opening a PR! Please fill in the sections below.
PRs target the `next` branch, not `main`.
-->

## What changed

<!-- One or two sentences describing the effect of this change. -->

## Why

<!-- Motivation. Link any related issue: Closes #N -->

## How to verify

<!-- Concrete steps a reviewer can run to verify the change. -->

```bash
# example
go test -race ./...
./dist/shy ...
```

## Checklist

- [ ] Conventional Commits used in commit messages
- [ ] Tests added or updated for any behaviour change
- [ ] `go vet ./...`, `go test -race ./...`, and `shellcheck` pass locally
- [ ] PR targets `next`, not `main`
