# Changelog

All notable changes to shy are documented here. Entries follow
[Conventional Commits](https://www.conventionalcommits.org/) and the
file is maintained by [release-please](https://github.com/googleapis/release-please)
from `next`.

## [0.2.1](https://github.com/alfred-intelligence/shy/compare/v0.2.0...v0.2.1) (2026-05-21)


### Bug Fixes

* **release:** Trigger GoReleaser on release:created not push:tags ([44e3789](https://github.com/alfred-intelligence/shy/commit/44e37897230cff59e94daf9dc003b1f4884a0d6e))

## [0.2.0](https://github.com/alfred-intelligence/shy/compare/v0.1.0...v0.2.0) (2026-05-21)


### Features

* **authoring:** Semver, conventional commits, git state detection ([6403cbb](https://github.com/alfred-intelligence/shy/commit/6403cbb0463ddb0591830aba1aa4ab73a7f33d44))
* **collection:** Subscribe/list/update/unsubscribe with manifest discovery ([9f9d074](https://github.com/alfred-intelligence/shy/commit/9f9d07489e4e4893dbc8c12b1c8a6adaceca339e))
* **create:** Scaffold a new script and open it in \$EDITOR ([af52235](https://github.com/alfred-intelligence/shy/commit/af5223589e8f1acc04795ef1dc5944f157a44384))
* **examples:** Hello-world reference plugin ([263bcc2](https://github.com/alfred-intelligence/shy/commit/263bcc2fcbf667360133f83fa1f3b3e1d208ef18))
* **examples:** Shy-stdlib starter collection ([20e2c4d](https://github.com/alfred-intelligence/shy/commit/20e2c4dc8afaad2f7ad29089ef4601f794629d33))
* **override:** List, add, and remove system-seed overrides ([25539af](https://github.com/alfred-intelligence/shy/commit/25539af398d3f7f78e7533f9fb7805c919a8f194))
* Phase 1 skeleton — module, init.bash, install.sh, manifest parser ([c04485c](https://github.com/alfred-intelligence/shy/commit/c04485cb8962800c253ee34836d1658d648568fc))
* Phase 2 core commands — init, install, list, info, remove, update, alias, completion ([c1e5935](https://github.com/alfred-intelligence/shy/commit/c1e5935c85911e714bb714c9803461a0dfd78237))
* **plugin:** Discover and index installed plugins ([a73ffd0](https://github.com/alfred-intelligence/shy/commit/a73ffd0397d5326ff4d095ab0f61bcbf7d886148))
* **plugin:** Dispatch \`shy <command>\` to plugin entry scripts ([82b7187](https://github.com/alfred-intelligence/shy/commit/82b7187bc071727e2c80424c9b4255ec511a014e))
* **plugin:** List discovered plugins under \`shy --help\` ([faea81a](https://github.com/alfred-intelligence/shy/commit/faea81af3b880393cc3109ba698acfb87fe4aa84))
* **publish:** Three git states, Conventional Commits version inference ([726b31a](https://github.com/alfred-intelligence/shy/commit/726b31ad2a3664eaffafe58be784a3ef30ec210b))
* **release:** Package cobra-generated man-pages into .deb / .rpm ([8b7d910](https://github.com/alfred-intelligence/shy/commit/8b7d9104e7c02cba8c81958a2c683c5c92803416))
* **release:** Release-please + post-release sync + dependabot auto-merge ([0b5774d](https://github.com/alfred-intelligence/shy/commit/0b5774d18c81da888620b752a0f916e3b1703829))
* **system-reset:** Destructive cross-user reset behind --yes-i-know + RESET ([01c7348](https://github.com/alfred-intelligence/shy/commit/01c734861cbfc599a89b21b52bb05c559e32f648))


### Bug Fixes

* **ci:** Unblock 0.2.0 release — three CI failures ([7f5a3f6](https://github.com/alfred-intelligence/shy/commit/7f5a3f6b4a190a3e221f9f7bb570eda1a8194882))
* **install.sh:** Handle missing tools, network failure, races, partial installs ([4afef07](https://github.com/alfred-intelligence/shy/commit/4afef0706c0a8ffcd575e67eaf8f7113b1165c0b))
* **install:** Reject alias/completion names that escape SHY_HOME ([6258b30](https://github.com/alfred-intelligence/shy/commit/6258b3024c2906d26cbcb0b6deb7924e79100892))

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
