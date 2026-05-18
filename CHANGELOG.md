# Changelog

All notable changes to shy are documented here. Entries follow
[Conventional Commits](https://www.conventionalcommits.org/) and the
file is maintained by [release-please](https://github.com/googleapis/release-please)
from `next`.

## [0.1.0] — unreleased

Initial scaffold and the full v0.1 feature surface from the design
package in [`docs/`](docs/). Highlights:

### Features

- **Binary, runtime, and installer.** Go binary (cobra + glamour),
  `init.bash` filesystem-walking runtime layer, and a hardened
  `curl|bash` installer with SHA256 verification, lockfile, and
  atomic binary replacement.
- **Core commands.** `init`, `install`, `list`, `info`, `remove`,
  `update`, `alias`, `completion add`, plus per-shell completion
  emitters (bash/zsh/fish/powershell). Path layout, host-derived
  namespacing, and a schema-versioned `cache.json`.
- **Collections.** `collection subscribe`, `update` (dry-run by
  default), `unsubscribe`, owner-tagged item tracking, dependency
  recursion, and bare-manifest sub-directory discovery.
- **Authoring.** `create` scaffolds under the host namespace and
  opens `$EDITOR`. `publish` handles the three git states from the
  whitepaper (no git → init; self-root → proceed; inside parent →
  abort with explanatory error and exit 1), Conventional Commits
  version inference, `--to-github` via `gh`, namespace rename to
  `scripts/<user.name>/<name>/`.
- **Plugins.** Discover/index installed plugins from manifests,
  dispatch `shy <command>` to plugin scripts, surface plugins in
  `shy --help`, and rebuild the index on every install/remove/
  update/subscribe. Reference plugin under
  `examples/plugins/hello-world/`. Median dispatch overhead ~17ms
  on commodity hardware (CI gates at 100ms).
- **Override and reset.** `override list/add/remove` against
  `/usr/share/shy/overrides.d/`. `system-reset` wipes the host
  behind `--yes-i-know` plus a typed `RESET` confirmation.
- **Distribution.** GoReleaser for linux/darwin × amd64/arm64
  tarballs, `.deb`/`.rpm` with cobra-generated man-pages, per-asset
  `.sha256`. release-please on `next`, post-release sync to `main`,
  Dependabot auto-merge for patch/minor gomod and github-actions
  bumps.
- **Starter stdlib.** `examples/stdlib/` ships `mkcd`, `extract`,
  `serve`, `path-list`, `up`, and a few common aliases. Slated for
  extraction to `github.com/alfred-intelligence/shy-stdlib` at the
  v1.0 cut.

### Tests

- Unit coverage across `manifest`, `paths`, `cache`, `install`,
  `collection`, `authoring`, `plugin`, and the cmd integration layer.
- Container-based acceptance matrix (`acceptance.yml`) across
  Ubuntu 22.04/24.04, Debian 12, Fedora 40, plus a native macOS run.

### Fixes

- **install:** preserve the executable bit when copying script
  files so plugin entries dispatch without a `chmod` dance.
- **install:** reject alias/completion names that would resolve to
  a directory or escape `SHY_HOME` (`.`, `..`, anything containing
  `/`, anything starting with `-`).
- **authoring:** `DetectGit` resolves symlinks before walking up so
  a symlinked script under a parent repo is recognised as such.

### Docs

- CONTRIBUTING, SECURITY (private-advisory + v1 threat model),
  CODE_OF_CONDUCT, PR/issue templates.
- `CLAUDE.md` for future agent sessions: design entry points,
  load-bearing invariants, approval boundaries.

[0.1.0]: https://github.com/alfred-intelligence/shy/releases/tag/v0.1.0
