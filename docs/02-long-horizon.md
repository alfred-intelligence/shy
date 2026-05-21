# shy — Long Horizon (Roadmap)

The roadmap is sequenced by capability dependencies, not by calendar
time. This is an AI-executed plan: the operator drives Claude through
each phase, with most code generation taking minutes rather than
weeks. Time estimates are therefore omitted — they would be
misleading either way (too generous for AI-paced work, too tight for
solo human pace).

What matters is the **order** and the **risk profile** of each phase.
Hit the dependencies in sequence; budget extra attention to the
high-risk phases.

The acceptance bar for v1.0 is concrete: a fresh machine can install
`shy`, install a stdlib collection, initialise the user environment,
create a new script via `$EDITOR`, publish it as a manifest, and load
a plugin invoked through the `shy` binary.

## Overview

| Phase | Name | Content |
|-------|------|---------|
| 1 | Skeleton | Repo, CI, `install.sh`, `init.bash`, manifest parser stub |
| 2 | Core commands | `init`, `install`, `remove`, `list`, `alias`, `completion add`, `update`, `info` |
| 3 | Collections | `collection subscribe/update/list/unsubscribe`, manifest discovery, external refs |
| 4 | Authoring | `create`, `publish`, `$EDITOR` integration, manifest generation |
| 5 | Plugin architecture | Plugin dispatch, `help` integration, cache rebuild on install |
| 6 | System install and diagnostics | `system-install/uninstall`, `doctor` |
| 7 | Distribution polish | GoReleaser pipeline, `.deb`/`.rpm`, man-pages, `install.sh` SHA verify |
| 8 | Public v0.9 | README polish, first stdlib collection released |
| 9 | v1.0 | Acceptance test passes end-to-end on a fresh VM; tag v1.0 |
| 10 | Ecosystem | First plugins (auto-completions, gh-clone, auto-clone); v1.x continues |

## Phase 1 — Skeleton

**Goal:** Repository exists at `github.com/alfred-intelligence/shy` with the
bare bones in place. Nothing functional yet; the structure is
correct.

**Done when:**

- Repository created; LICENSE (MPL-2.0) and README in place
- `go.mod` initialised; minimal `cli/cmd/main.go` that prints version
- `install.sh` at repo root: detects OS/arch, fetches binary from
  GitHub Releases (even if the release is a draft), verifies SHA256,
  unpacks to `$HOME/.shy/bin/shy`
- `init/init.bash` template that sources four directories from
  `$HOME/.shy/` with per-file error swallowing and `_`-prefix skip
- Glamour and Cobra added to `go.mod`
- GoReleaser config (`.goreleaser.yaml`) targeting linux/amd64,
  linux/arm64, darwin/amd64, darwin/arm64; produces
  `shy_<version>_<os>_<arch>.tar.gz` plus `.sha256` checksum and
  `.deb`/`.rpm` packages
- GitHub Actions workflow: `go vet`, `go test`, `shellcheck` on
  `install.sh` and `init.bash`
- First draft release `v0.1.0-draft` pushed; `install.sh` can fetch
  and unpack it

**Dependencies:** None — this is the foundation.

**Risk profile:** Moderate. GoReleaser asset naming must be locked
correctly now and never changed. The asset name template is the
permanent contract with `install.sh`.

## Phase 2 — Core commands

**Goal:** The binary handles single-machine operations: install
local files as snippets, alias generation, completion installation,
remove, update, list with type filtering, info rendering.

**Done when:**

- `shy init` — creates `$HOME/.shy/{scripts,aliases,completions,
  plugins,overrides.d}/`, writes `init.bash` if missing, adds source
  line to `~/.bashrc` if not present (interactive prompt; skip with
  `--no-bashrc`), auto-installs `shy`'s own bash-completion via
  `shy completion bash`, reports summary. Refuses if run as root
  (points to `shy system-install` for that use case).
- `shy install <path-or-url>` — installs from a local path,
  `file://` URL, `https://` URL, or git reference (`@user/repo`).
  Validates manifest. Resolves item types. Pins to current HEAD
  commit by default; `--track-main` to follow rolling.
- `shy alias <name>='<value>'` — imperative helper: writes
  `$HOME/.shy/aliases/<name>`
- `shy completion add <tool>` — imperative helper: runs `<tool>
  completion bash` (or known variants), writes output to
  `$HOME/.shy/completions/<tool>`
- `shy list [--type=script|alias|completion|plugin|override]
  [--sources]` — shows installed items with colour-coded output;
  `--sources` shows where each item came from
- `shy info <namespace>/<name>` — renders README.md with glamour;
  `--raw` for raw markdown
- `shy remove <namespace>/<name>` — removes the named item
- `shy update [<name>]` — re-fetches manifest from source, reinstalls
  if version differs; refuses items without `[source]`
- `shy-reload` alias installed by `shy init` for re-sourcing
  `init.bash` without opening a new shell
- Unit tests for manifest parser, item type validation, path
  resolution, namespace resolution

**Dependencies:** Phase 1 complete.

**Risk profile:** High. TOML parser edge cases, namespace
resolution rules, and the `[source]`-handling all interact. Most
foundation for everything else.

## Phase 3 — Collections

**Goal:** Subscription-based distribution works. A user can
subscribe to a collection on GitHub, all referenced items install
under the right namespace, external script references are fetched,
conflicts are flagged.

**Done when:**

- `shy collection subscribe github:user/name[@ref]` — clones the
  collection repo to `$HOME/.shy/collections/<name>/`, reads
  `manifest.toml`, installs every item under the appropriate
  namespace, recursively resolves `[[dependencies]]` with skip-if-
  already-installed for cycle prevention
- `shy collection list` — shows subscribed collections with version
  info and ref/commit
- `shy collection update [<name>]` — defaults to `--dry-run`; shows
  diffs that would be applied. `--apply` performs the upgrade.
  Conflict resolution: 1–2 alias/completion diffs prompt
  one-by-one; 3+ trigger bulk-summary prompt with `accept all from
  new` / `keep all current` / `prompt one-by-one`.
- `shy collection unsubscribe <name>` — removes the collection
  clone and uninstalls items that came from it (unless also owned
  by another subscription)
- Manifest discovery: when a collection's root `manifest.toml`
  lacks `entry` and `[[items]]`, items are inferred from
  sub-directories containing their own `manifest.toml`
- `SHY_ON_CONFLICT` environment variable handling for non-
  interactive contexts (`prefer-existing`, `prefer-new`, `skip`,
  `fail`; default `fail`)

**Dependencies:** Phase 2 complete (single-item installation works).

**Risk profile:** High. External reference resolution, dependency
graphs, and conflict UX. Where the design's social model collides
with the security model.

## Phase 4 — Authoring

**Goal:** The operator can create new scripts and publish them as
shareable artefacts.

**Done when:**

- `shy create <name>` — creates
  `$HOME/.shy/scripts/<hostname>/<name>/`, generates a `.sh`
  skeleton with `#!/usr/bin/env bash` and a placeholder `echo "OK"`,
  generates a README.md stub (heading + placeholder paragraph),
  opens the `.sh` in `$EDITOR`
- `shy publish <name>` — handles three git states with appropriate
  severity:
  - **State 1 (no git):** runs `git init`, makes initial commit,
    prompts for manifest fields, writes manifest, moves to publish
    namespace
  - **State 2 (already a repo at script root):** notes existing
    repo, reports working-tree state, infers version from
    Conventional Commits, proceeds
  - **State 3 (inside parent repo):** aborts with informative
    message; exit code 1
- All publish states require `git config --global user.name`;
  refuses with clear error if missing
- Conventional Commits parsing for version inference: `feat:` →
  minor, `fix:` → patch, `feat!:` / `BREAKING CHANGE:` → major,
  others ignored
- `shy publish <name> --to-github` uses `gh` if installed to create
  the GitHub repo and push; without `gh`, fails informatively
- `shy publish <name> --version <semver>` overrides inference
- Manifest validation runs on publish; warnings about missing
  recommended fields shown but do not block
- README.md stub-vs-content check: warn but don't block if README
  is missing or just the default stub

**Dependencies:** Phase 3 complete (manifests are well-understood
by the binary).

**Risk profile:** Moderate. `$EDITOR` integration on systems
without one set; git context probing; namespace migration on
publish; Conventional Commits parsing edge cases.

## Phase 5 — Plugin architecture

**Goal:** Items declared as plugins are dispatchable through `shy
<command>`. The binary's surface area grows through plugins, not
just native code.

**Done when:**

- `shy <command>` first checks native subcommands; if none matches,
  walks plugin manifests for matching `command` field
- Plugin invocation execs the plugin's entry script with remaining
  arguments and the operator's environment
- `shy help` lists native subcommands and discovered plugins
  separately, with descriptions from each plugin's manifest
- Plugin discovery cached in `$HOME/.shy/cache.json`; rebuilt
  automatically on `install`, `remove`, `update`, or when the cache
  is missing
- A reference plugin shipped in the repository under
  `examples/plugins/hello-world/` demonstrates the contract
- Integration test: install reference plugin, run `shy hello-world`,
  observe expected output
- Plugin dispatch overhead measured empirically in CI; threshold
  set at <100ms per `shy <plugin-command>` invocation on standard
  GitHub Actions runners. Threshold violation fails the build.

**Dependencies:** Phase 4 complete (manifests can declare plugins).

**Risk profile:** High. The plugin model is the design's
distinguishing feature; if dispatch is wrong or discovery is slow,
the whole "off-ramp for feature creep" promise fails.

## Phase 6 — System install and diagnostics

**Goal:** Sysadmin can seed `/etc/skel/` for new users; operator
has diagnostics to verify their setup.

**Done when:**

- `sudo shy system-install` — seeds `/etc/skel/.shy/` with a
  default init.bash and empty shy.toml; appends the source line to
  `/etc/skel/.bashrc` (idempotent); writes
  `.shy/.system-installed` marker for audit and uninstall. Refuses
  if not run as root.
- `sudo shy system-uninstall` — reverses system-install. Removes
  `/etc/skel/.shy/`, strips the source line from
  `/etc/skel/.bashrc` (leaving the rest intact), removes the
  marker. Does **not** touch existing users' `$HOME/.shy/`.
- `shy doctor` — diagnostic command. Reports: binary version,
  init.bash status, `$HOME/.shy/` structure, installed items
  (count by type), known issues (broken symlinks, missing source
  line in `.bashrc`, etc.). No sudo required.
- `sudo shy init` refuses with hint pointing to `shy system-
  install` for the `/etc/skel/`-seeding use case.

**Dependencies:** Phase 2 complete (`shy init` exists).

**Risk profile:** Low. `system-install` and `system-uninstall` are
single-machine, idempotent, and well-bounded by the `/etc/skel/`
contract. `shy doctor` is read-only.

## Phase 7 — Distribution polish

**Goal:** `install.sh` is robust, distribution packages exist with
man-pages, and the release pipeline is reproducible.

**Done when:**

- `install.sh` handles all common edge cases: existing partial
  install, missing `curl`, missing `tar`, unsupported OS/arch
  (actionable message), network failure during download (clean
  rollback)
- `install.sh` supports both user-level install (default,
  `$HOME/.local/bin/shy`, no sudo) and system-wide install
  (`--system` flag, `/usr/local/bin/shy`, requires sudo)
- `install.sh` interactively prompts to add the source line to
  `.bashrc`; supports `--no-bashrc` flag to skip
- GoReleaser produces `.deb` and `.rpm` packages containing
  **only** the binary, man-pages, and shy's own bash-completion
  file. No user content, no seed data.
- `shy gen-man /tmp/man` produces man-pages for every subcommand
- Man-pages packaged into `.deb`/`.rpm` via GoReleaser nfpms
  configuration; installed to `/usr/share/man/man1/`
- `man shy`, `man shy-install`, etc. work after `apt install shy`
- SHA256SUMS file in releases; `install.sh` verifies before
  unpacking
- Lock-file mechanism for `install.sh` to prevent races if invoked
  twice in parallel
- Homebrew formula in a separate tap (`alfred-intelligence/tap`)
  installing the same binary; `brew install shy` works on macOS

**Dependencies:** Phase 6 complete.

**Risk profile:** Moderate. Distribution package conventions vary
subtly between Debian and Red Hat families. Homebrew formulas in
taps have their own conventions to learn.

## Phase 8 — Public v0.9

**Goal:** Repository is public. README is polished. A first stdlib
collection exists.

**Done when:**

- `github.com/alfred-intelligence/shy` is public
- README explains the model in under 200 lines, with copy-paste
  install commands
- `LICENSE`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`
  in repo root
- A first stdlib collection (`github.com/alfred-intelligence/shy-stdlib`)
  exists with 5–10 broadly useful scripts
- Public install from a fresh VM works end-to-end
- One demo (text-based or short video) walks through the v1.0
  acceptance test
- Posted to one relevant venue (HN, Lobsters, self-hosted
  subreddit) for early signal

**Dependencies:** Phase 7 complete.

**Risk profile:** Low technically; high socially. Initial reception
shapes priorities.

## Phase 9 — v1.0

**Goal:** Tag v1.0. Acceptance test passes end-to-end on a fresh
VM.

**Done when:**

- Fresh VM (Ubuntu 22.04 or similar) provisioned from scratch
- Acceptance sequence works without intervention:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/alfred-intelligence/shy/main/install.sh | bash
  sudo shy install @alfred-intelligence/shy-stdlib
  shy init
  shy create my-first-script   # opens $EDITOR; user types echo "OK"; saves
  shy publish my-first-script  # generates manifest.toml
  shy-reload                   # picks up the new script
  my-first-script              # outputs "OK"
  ```
- A reference plugin (e.g. `@alfred-intelligence/shy-gh-clone`) installs and
  invokes as `shy gh-clone <repo>`
- All Phase 1–8 done-criteria still hold
- `CHANGELOG.md` summarises what changed since v0.9
- Tag `v1.0.0` pushed; GoReleaser produces release artefacts
- Container-based acceptance test (see `06-ci-cd-plan.md`,
  `acceptance.yml`) passes across the OS compatibility matrix.
  This is the automated counterpart to the manual fresh-VM test —
  ensures the same sequence works on every supported distribution
  without operator intervention.

**Dependencies:** Phase 8 feedback addressed.

**Risk profile:** Low. By this point the design has been exercised
end-to-end; v1.0 is a tagging exercise.

## Phase 10 — Ecosystem (ongoing)

**Goal:** Plugins and stdlib content grow organically. No fixed
end point.

### v1.x — plugins on the v1 base

- **`@alfred-intelligence/shy-audit` is the priority plugin.** Static analysis
  of installed scripts and plugins against suspicious patterns
  (eval of untrusted input, network calls to undeclared hosts,
  reads of sensitive paths, subprocess spawns to undeclared
  binaries). Reads `[capabilities]` declarations from manifests
  and flags gaps against actual code. Trigger to build it: when the
  operator starts losing track of subscribed collections and their
  trust state.
- **Plugins shipped early:**
  - `@alfred-intelligence/shy-auto-completions` — weekly scanner that detects
    new tools on `$PATH` and installs their completions
  - `@alfred-intelligence/shy-gh-clone` — clone GitHub repos with default org
  - `@alfred-intelligence/shy-auto-clone` — clone subscribed collections to a
    configurable local directory for editing
- **Stdlib expansion** with broadly useful scripts contributed
  through PRs to `alfred-intelligence/shy-stdlib`
- **Documentation polish**: hosted docs at `shy.alfred-intelligence.io`
- **Additional distribution targets** (AUR, Homebrew, NixOS) as
  community contributions

### v2 — workspace shy

The v2 vision is shy as a terminal workspace, not just a CLI. The
core shifts from "manage your snippets" to "run your terminal
session", with snippets remaining the substrate.

- **Plugin sandboxing via bubblewrap/firejail.** Built when
  `shy audit` reveals consistent gaps between declared
  `[capabilities]` and actual plugin behaviour. Enforces sandboxed
  exec for plugins; sourced scripts remain unsandboxed by
  architecture.
- **Terminal workspace controller** with tmux backend and named
  windows: bash, local AI (Ollama), Claude Code, git-repo browser,
  optionally IDE-style editor pane.
- **Named persistent sessions** with isolated state per session
  (AI history, Claude Code ID, bash history, layout).
- **Workspace-sandbox** as outer defence layer; plugin-sandbox
  remains inner. Scripts sourced inside the workspace become
  sandboxed via the outer layer — architectural fix for v1's
  "scripts cannot be sandboxed" limitation.
- **Error console window.** Mini-DSL filtering, colour-coded
  output by level and source, resource watchdog for memory and CPU.
- **`[security]` tag CVE verification.** Currently v1 trusts
  upstream `[security]` claims at face value. v2 will require
  valid CVE references that shy verifies against NVD or
  GitHub Advisory Database. Severity levels become enforced.
- **Zsh and fish support.** Runtime sourcing layer gets zsh/fish
  variants of `init.bash`.

### Out of shy-core, into independent plugin scope

The following items were previously sketched as v2 features of
shy-core. They have been moved out of shy's roadmap entirely and
into independent plugin scope. They may be developed under the
`alfred-intelligence` organisation as standalone projects that
integrate with shy via the plugin model and extensible manifest
parser.

- **`@alfred-intelligence/shy-kebab-conformance`** — cross-ecosystem
  variable-namespace convention enforcement. Reads
  `[kebab-conformance]` sections from item manifests and verifies
  semantic compatibility between installed items. Couples with
  kebab-it's librarian agent. Trigger: when shy's ecosystem grows
  large enough that variable-name collisions across collections
  from unknown authors become a real problem.
- **`@alfred-intelligence/shy-sign`** — GPG signing for plugin and
  collection releases, modelled on kebab-it-core's signing flow.
  Trigger: when the user base warrants a trust layer beyond
  gitops + default-pinning.

These are not on shy's critical path. If neither plugin is ever
built, shy-core works exactly as designed.

## Dependencies between phases

```
Phase 1 (skeleton)
    ↓
Phase 2 (core commands)
    ↓
Phase 3 (collections)
    ↓
Phase 4 (authoring)
    ↓
Phase 5 (plugins) ─────────┐
    ↓                      │
Phase 6 (override/reset)   │
    ↓                      │
Phase 7 (distribution) ←───┘
    ↓
Phase 8 (public v0.9)
    ↓
Phase 9 (v1.0)
    ↓
Phase 10 (ecosystem, ongoing)
```

Phases 2–4 are strictly sequential because each builds on the
previous's manifest semantics. Phase 5 (plugins) can technically
begin in parallel with Phase 4 (authoring), but a solo operator
serialises.

## High-risk phases

Three phases are worth extra attention because they introduce
genuinely new complexity rather than mechanically applying earlier
patterns:

- **Phase 2 (core commands)** — first time the manifest schema is
  exercised against real installations. Namespace resolution,
  TOML edge cases, item-type validation all interact.
- **Phase 3 (collections)** — introduces the network, dependency
  graph, and cross-source conflict resolution. The conflict-prompt
  UX is novel and will need iteration on real subscriptions.
- **Phase 5 (plugins)** — the distinguishing feature. Dispatch
  semantics, help integration, and cache invalidation must work
  cleanly or the off-ramp promise fails.

Other phases are mostly mechanical execution of patterns
established in these three.

## Signals to delay or reconsider

Stop and rethink if:

- **Phase 2** reveals that the manifest schema is unwieldy in
  practice. Revise `01-whitepaper.md` rather than working around
  it.
- **Phase 3** shows that conflicts between subscribed collections
  are common and the prompt UX is hostile. Reconsider whether
  conflict resolution should be more rule-based.
- **Phase 5** finds that plugin dispatch overhead is too slow to
  feel snappy (>100ms per `shy <command>`). Reconsider caching
  strategy or move to a daemon-based dispatcher (would be a major
  design change).
- **Phase 8 feedback** is uniformly negative about a core premise
  ("the manifest is too complex", "TOML is the wrong format",
  "user > system is the wrong default"). Reconsider before tagging
  v1.0.

## Non-goals

These are deliberately not on any roadmap. They are not "maybe
later" — they are "no, not in `shy`":

- Themes and prompt-builders (use `oh-my-bash` or a dedicated tool)
- Full dotfile management (use `chezmoi` or `yadm`)
- Cross-shell support beyond bash at v1 (zsh support is reserved
  for v1.x via the cobra completion already in place)
- Administrative enforcement on shared hosts (user > system always)
- Encrypted content (snippets are public-by-design)
- Realtime sync across machines (collections-via-git is asynchronous
  by nature, which is a feature)
