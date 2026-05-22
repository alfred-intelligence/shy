# shy — Whitepaper

> A CLI tool for managing bash snippets, aliases, and completions
> across multiple machines through git-based subscribable
> collections. The premise: your `.bashrc` should be a single
> sourcing line, not a thousand-line monolith.

## Problem

Bash users who spend time on the command line accumulate snippets,
aliases, and completions over the years. They live as ad-hoc edits
to `~/.bashrc`. Three problems compound over time:

- **Cross-machine portability is broken.** A useful function on one
  machine gets re-typed (and slightly mis-typed) on the next. There
  is no canonical version. Backup is `git clone dotfiles && ln -s`,
  which is fragile and unselective.
- **Binary-coupled snippets pollute the file.** Functions that only
  make sense when a specific binary is installed (kubectl helpers,
  Docker wrappers, GitHub CLI shortcuts) live unconditionally in
  `.bashrc`, throwing errors on machines where the binary is absent.
- **Completions take half the file.** `kubectl completion bash`
  output alone is a thousand lines. With a handful of tools it
  becomes impossible to scan `.bashrc` visually.

Existing tools solve adjacent problems but not this one:

- **`oh-my-bash` / `bash-it`** are *frameworks* with central
  curation. They are not tool managers — they are opinionated
  bundles with themes, plugins, and conventions. You buy in or stay
  out.
- **`chezmoi` / `yadm` / `nix-home-manager`** manage entire dotfile
  setups. They are powerful but heavy: templating, encryption,
  profiles, host detection. For the narrower problem of "manage my
  bash bits", the ceremony is wasted.
- **`zinit` / `sheldon`** (zsh-only) come closest in spirit —
  modular plugin managers with multiple sources. There is no real
  counterpart for bash.

`shy` is a direct response. It does one thing: manage bash snippets,
aliases, and completions with cross-machine portability through
git-distributed collections. It does not template, encrypt, theme,
or own your entire shell. It is a tool manager — closer to `pip`
than `apt` in spirit.

## Target audience

- **Solo developers and engineers** with multiple machines (laptop,
  desktop, servers) who want a consistent shell environment without
  the friction of a full dotfile-management framework.
- **Homelab operators** managing several Linux hosts where each
  user has slightly different bash needs but a shared baseline.
- **The author specifically.** This project starts as a personal
  tool with the discipline to be useful to others, not the ambition
  to be popular.

It is **not** for:

- Teams enforcing shell conventions across many users — that wants
  centralised tooling with administrative controls; `shy` keeps the
  user always above the system, so admins cannot force changes onto
  users' shells.
- Operators who want themes, prompt-builders, and out-of-the-box
  feature bundles — that is `oh-my-bash`.
- People migrating their entire dotfile setup — `chezmoi` is the
  right tool for that.

## Prior art

shy stands on a decade of work by others who have explored adjacent
parts of this problem space. Several deserve explicit recognition,
both because they got the foundational ideas right and because
understanding their choices clarifies what shy adds.

**`basher` (Carlos Becker et al., basherpm/basher, since the early
2010s).** The closest neighbour. basher established the model that
shy inherits: install shell packages from git, store them in a
central per-user location, register their binaries on `PATH`, and
let an init hook activate the lot from `.bashrc`. It supports bash,
zsh, and fish; handles dependencies through `package.sh` metadata;
manages completions and man-pages; and has been quietly reliable
since long before this whitepaper was written. The fact that a
project this small has survived a decade of churn in the bash
ecosystem is not an accident — it is evidence that the core idea is
sound. basher's deliberate scope (a bash implementation that does
one thing well: installing scripts from git) is a feature, not a
limitation. Anyone who has used basher and found it sufficient is
not in shy's target audience, and that is fine.

**`bpkg` (`bpkg/bpkg`).** A second well-loved bash package manager
in the same lineage. bpkg uses `bpkg.json` for metadata, supports
both global and per-project installation modes, and emphasises
lightweight integration. The dual-mode (global vs per-project) is
an idea shy does not adopt — shy is per-user only — but the
philosophy of "lightweight, install from git, don't get in the way"
is shared.

**Adjacent tools, different problems.** `oh-my-bash` and `bash-it`
are frameworks: they bundle themes, prompts, and curated alias sets
behind a single install. They optimise for "decorate my shell";
shy optimises for "let me curate my own snippets and share them".
`chezmoi` and `yadm` manage entire dotfile repositories with
templating and per-host conditionals; they own the whole user
environment. shy intentionally owns only the small, sourceable
pieces.

**What shy adds.** The grounding idea — install shell things from
git via a per-user package manager — is not new, and shy does not
pretend otherwise. What is new is taking that idea further on
several axes simultaneously:

- A Go binary as dispatcher, allowing fast subcommand routing and
  enabling plugins to register as first-class commands (`shy
  gh-clone <repo>`) rather than just additions to `PATH`.
- Aliases and completions treated as first-class item types with
  their own lifecycle, not as side-effects of installing a binary.
- Collections as a meta-package primitive: one git repository can
  declare a curated set of plugins, scripts, and aliases that
  install together, with reference resolution and conflict
  handling.
- A `manifest.toml` deliberately scoped to metadata only, with the
  filesystem as runtime source of truth — making the system
  legible by direct inspection rather than configuration parsing.
- A per-user install model that owns nothing outside `$HOME/.shy/`,
  with an optional `/etc/skel/`-based seed mechanism that lets
  sysadmins activate shy for new users without taking control away
  from anyone.
- A `shy publish` flow that uses Conventional Commits to infer
  versioning and treats publication as the establishment of a
  distributable git repository, not just a copy operation.

None of these is a criticism of basher or bpkg. They are choices
that follow naturally once one decides to take the foundational
idea — install shell things from git — and run with it further than
its originators chose to. shy is not better than basher; shy is
broader, with the costs and complexity that breadth entails. Anyone
whose needs are met by basher should keep using basher. Anyone
whose needs have outgrown basher's scope, and who has been
assembling that scope themselves from tmux configurations, ad hoc
plugins, and dotfile sprawl, is the audience shy is built for.

## Solution

`shy` is one CLI binary plus a thin shell layer.

**1. The binary `shy`.** Written in Go, distributed as a static
single-file release. Handles all operations: install, remove,
update, list, create, publish, init, info, outdated, snooze,
system-install, system-uninstall, doctor, and plugin dispatch.
The binary embeds `glamour` for rendering README files inline and
`cobra` for subcommand structure, help text, man-page generation,
and shell-completion generation. It exposes a structured API to
plugins via
`--json` on read commands and `--silent` on write commands.

Two complementary installation channels:

- **User-level install** via `curl | bash`. Lands in
  `$HOME/.local/bin/shy` by default, or `/usr/local/bin/shy` if
  the operator opts for system-wide placement (requires sudo).
  Targets the user who ran the command. The installer script
  lives at a stable URL and remains backward compatible forever.
- **System package install** via distribution packages (`.deb`,
  `.rpm`) or Homebrew. Lands the binary in `/usr/bin/shy` (Linux
  packages) or `$(brew --prefix)/bin/shy` (Homebrew). Man-pages
  installed to `/usr/share/man/man1/`. Bash-completion installed
  to `/usr/share/bash-completion/completions/shy`.

System packages contain **only the binary plus man-pages and shy's
own bash-completion file** — no content, no seed data, no
user-area modification. Installing the package does not alter any
user's environment until they explicitly run `shy init`.

Both install methods can coexist; user-level installs take
precedence via `$PATH` ordering when both are present.

**Optional seed for new users (sysadmin opt-in).** A separate
command `sudo shy system-install` seeds `/etc/skel/.shy/` so that
users created afterward (via `useradd -m`) get a working shy
setup automatically. This is opt-in and never altered by package
installation. See § Seed model for new users below.

**2. The shell layer.** A small `init.bash` file sourced from
`~/.bashrc`. It sources `entry.sh` from every `installed/%<ns>/<name>/`
directory (scripts), and flat files from `helpers/aliases/` and
`helpers/completions/`. Override equivalents under `overrides.d/` are
sourced last to win any conflicts. One bad file does not break shell
startup — errors are reported per-file to stderr while sourcing continues.

**3. The manifest format (TOML).** Every package — whether a
single-script repo or a multi-item collection — uses the same
schema. The binary interprets the structure semantically. **Manifest
is metadata for sharing and distribution only.** The runtime layer
(init.bash) sources files based on filesystem structure and ignores
manifests entirely.

**4. Collections as the distribution unit.** A collection is a git
repository with a `manifest.toml` at the root. Other users can
subscribe to a collection (`shy collection subscribe github:user/name`),
which installs everything declared in the manifest including
external references to other repos. Cross-machine sync emerges
naturally: subscribe the same collection on every machine.

The operator's mental model:

1. `curl | bash` to install `shy` once on a new machine.
2. `shy init` to set up `$HOME/.shy/` and the `.bashrc` integration.
3. `shy install <thing>` to add a snippet, alias, completion, or
   plugin.
4. `shy collection subscribe github:user/name` to inherit a curated
   set.
5. `shy create <name>` to draft a new script (opens `$EDITOR`).
6. `shy publish <name>` to generate a `manifest.toml`, initialise
   git if needed, and publish.

`shy` modifies `~/.bashrc` exactly once during the first `shy init`,
adding a single source line. No other state lives outside of
`$HOME/.shy/`.

## Architecture

### File system layout

```
$HOME/.shy/                      ← Per-user, owned by user, chmod 700
├── init.bash                    ← Sourced from ~/.bashrc
├── installed/                   ← All installed items; directory prefix encodes type
│   ├── %<namespace>/<name>/     ← % = script — entry.sh sourced at shell start
│   ├── @<namespace>/<name>/     ← @ = plugin — exec'd on `shy <command>`
│   └── #<name>/                 ← # = collection clone (raw git checkout)
├── helpers/                     ← Flat files, all sourced
│   ├── aliases/
│   └── completions/
├── overrides.d/                 ← Sourced last; re-defines items from user layer
│   ├── installed/%<namespace>/<name>/
│   └── helpers/
│       ├── aliases/
│       └── completions/
├── bin/                         ← shy binary + any plugin launchers
└── cache.json                   ← Internal runtime cache; not a plugin API

/etc/skel/.shy/                  ← Seed for *new* users (optional)
├── init.bash                    ← Default init script template
└── shy.toml                     ← Default config (empty)
```

**Directory prefix convention** — a single symbol as the first character of the
namespace directory communicates the type without a separate layer:

| Prefix | Type | Sourced at start? |
|--------|------|-------------------|
| `%` | script | yes — `entry.sh` sourced into the shell |
| `@` | plugin | no — exec'd on demand |
| `#` | collection | no — raw clone, content installed elsewhere |

**Entry point** — every script and plugin directory contains an `entry.sh` file
as its canonical entry point. Helper files within the same directory may have
any name; only `entry.sh` is sourced/exec'd by shy itself.

Two locations, both per-user-scoped at runtime:

- **`$HOME/.shy/`** is the operator's canonical area. Everything
  shy reads or writes during runtime lives here. The operator
  owns it; shy never modifies it outside of explicit operator
  commands (`shy init`, `shy install`, etc.).
- **`/etc/skel/.shy/`** is a *template*, not a runtime location.
  Linux's `useradd -m` mechanism copies its contents to new users'
  home directories at account creation time. shy itself never
  reads from `/etc/skel/` at runtime. The seed exists only so
  newly created users get a working shy environment without
  having to run `shy init` manually.

There is no `/usr/share/shy/` or similar system-content area. Shy
binaries from `.deb`/`.rpm`/Homebrew install only the executable,
man-pages, and shy's own bash-completion file — never any user
content.

### Namespacing strategy

**Scripts and plugins are namespaced; aliases and completions are
not.**

A script lives at `$HOME/.shy/installed/%<namespace>/<name>/entry.sh`.
The `<namespace>` comes from:

- **Published items**: `<namespace>` is the author handle from the
  manifest's `[source].repo` (e.g., `alice` from
  `alice/git-autofetch`).
- **Locally created items**: `<namespace>` is the safe-name of
  `$HOSTNAME`. Safe-name is lowercase, `a-z0-9-` only, `.local`
  suffix stripped. `MacBook-Pro-2.local` becomes `macbook-pro-2`.
- **Manually placed items without manifest**: same as local — safe-
  name of `$HOSTNAME` for the namespace.

This means two collections can publish scripts with the same name
without filesystem collision: `scripts/alice/git-autofetch/` and
`scripts/bob/git-autofetch/` coexist as distinct installations.

**Aliases and completions are not namespaced — by design.** An
alias is one thing at a time (the last-defined `ll` wins; you
cannot have two `ll` aliases active simultaneously). Namespacing
them at the filesystem level would not change runtime semantics —
last-sourced still wins — so it would create a false sense of
safety while hiding the conflict. Better to surface the conflict at
install/subscribe time with the diff-prompt flow (1–2 conflicts get
one-by-one prompts; 3+ get a bulk-summary prompt with `accept all
from new` / `keep all current` / `prompt one-by-one`).

### How `shy init` works

`shy init` initialises a user's shy area. It:

1. Creates `$HOME/.shy/` with `chmod 700` (protects against other
   users on shared hosts).
2. Creates the standard subdirectory structure under `$HOME/.shy/`
   with permissions inherited from the umask.
3. Writes a default `init.bash` to `$HOME/.shy/init.bash` that
   walks the subdirectories and sources files in the right order.
4. Auto-installs shy's own bash-completion to
   `$HOME/.shy/completions/shy` via `shy completion bash >
   $HOME/.shy/completions/shy`.
5. Writes the source line to `~/.bashrc` if not already present
   (with an interactive prompt if `--no-bashrc` is not specified).

Re-running `shy init` is idempotent: existing files are not
overwritten unless `--force` is passed.

`sudo shy init` refuses with a clear error pointing to
`shy system-install`:

```
shy init creates per-user state. Don't run as root.

To seed /etc/skel/ for new users on this machine, use:
  sudo shy system-install
```

### Seed model for new users

`sudo shy system-install` (sysadmin opt-in) seeds `/etc/skel/` so
new users get shy active by default. The command:

1. Creates `/etc/skel/.shy/` with a minimal init.bash and empty
   shy.toml.
2. Appends `source $HOME/.shy/init.bash` to `/etc/skel/.bashrc`
   (or creates `.bashrc` if absent), idempotent — won't duplicate
   on re-run.
3. Writes a marker file (`.shy/.system-installed`) for audit and
   later uninstall.

After `system-install`, every new user created via `useradd -m`
gets `/etc/skel/`'s contents copied to their home, including the
`.shy/` seed and modified `.bashrc`. Their first shell start
activates shy.

**This does not affect existing users.** Their `$HOME/` is
unchanged. They install shy for themselves via `shy init` if they
want it.

`sudo shy system-uninstall` reverses the seed:

1. Removes `/etc/skel/.shy/`.
2. Removes the `source $HOME/.shy/init.bash` line from
   `/etc/skel/.bashrc` (leaves the rest of the file intact).
3. Removes the marker file.

It does **not** touch existing users' `$HOME/.shy/`. Those are
their property to manage.

### Runtime layer order

At every shell start, `init.bash` sources files in two layers:

1. **User layer** — `$HOME/.shy/{scripts,aliases,completions}/`.
   The user's canonical content.
2. **Overrides** — `$HOME/.shy/overrides.d/{scripts,aliases,completions}/`.
   Sourced after the user layer; later wins by standard bash
   semantics.

The override layer exists for user-level customisation of
installed content. If the operator installs a collection but wants
to modify one of its scripts, the modification goes in
`overrides.d/` rather than directly editing the installed file —
otherwise `shy update` of the collection would overwrite the
local change.

### Helpers convention

Files in script and plugin directories whose names start with `_`
are skipped by `init.bash`. This is the convention for private
helper files that the entry script sources internally but which
should not be globally sourced into the user's shell.

```
scripts/alice/git-autofetch/
├── git-autofetch.sh         ← entry, sourced by init.bash
├── _is-git-repo.sh          ← helper, skipped by init.bash
└── _fetch-async.sh          ← helper, skipped by init.bash
```

The entry script sources its helpers manually. The convention is
opt-in and entirely independent of the manifest. "Private" here is
a programming-encapsulation convention, not a security mechanism —
`_`-prefixed files are still readable and inspectable.

### Item types — sourced vs dispatched

shy distinguishes four item types in the manifest: `script`,
`alias`, `completion`, and `plugin`. At the manifest and CLI level
these are separate categories with their own ergonomics and
conflict-resolution rules. **At runtime there are only two
categories: sourced and dispatched.**

| Item type | Runtime category | Mechanism |
|---|---|---|
| `script` | Sourced | Loaded into the user's shell at startup; persists state (functions, env, CWD) |
| `alias` | Sourced | Single-line `alias x='y'` file loaded at startup; degenerate case of script |
| `completion` | Sourced | Completion-spec file loaded at startup; defines tab-completion behaviour for an external tool |
| `plugin` | Dispatched | Executed as subprocess when `shy <command>` is invoked; no shell-state side effects |

**The mental model:** *does this code need to run in your shell, or
does it just need to run?* If it modifies shell state (defines a
function, sets an environment variable, changes directory, adjusts
`PROMPT_COMMAND`), it must be sourced — that is what `script`,
`alias`, and `completion` are for. If it performs a discrete action
that begins and ends as a subprocess, it is a plugin.

Aliases and completions exist as separate item types from scripts
purely for UX and distribution reasons:

- `[aliases] gst = "git status -sb"` is more ergonomic than a full
  `[[items]]` block with `type = "alias"` and `value`.
- shy can auto-generate completions via `<tool> completion bash`
  without the operator authoring them by hand.
- Conflict resolution targets aliases by name (one `ll` wins) and
  completions by tool name; the targeting differs from script
  conflicts, which collide on shell-function names.

Operationally — disk layout (`scripts/`, `aliases/`,
`completions/`), CLI filters (`shy list --type=alias`), and
init.bash's sourcing loop — they are kept apart. Conceptually they
are all the same thing: code that runs in your shell.

The architectural consequence is that **only plugins can be
sandboxed** (planned for v2 via bubblewrap/firejail). Sourced items
must execute in the operator's interactive shell to do their job;
sandboxing would break the sourcing semantics they rely on. This is
permanent across all versions of shy, not a v1 limitation.

### Manifest format

A single TOML schema covers single-script repos, multi-item
collections, and plugins. The binary infers the form from what is
present in the file plus what is present on disk in the repo
structure.

**Manifest is metadata, not configuration.** It is read by the CLI
for `list`, `info`, `update`, and `publish`. It is not read by
`init.bash` at runtime. Files on disk are the source of truth for
what gets sourced; the manifest describes those files for sharing
purposes only.

**The parser is extensible.** shy's TOML parser tolerates top-level
sections it does not recognise rather than rejecting them. Unknown
sections are preserved verbatim and returned to plugins via
`shy info <name> --json`, which exposes the full parsed manifest.
This lets plugins evolve their own metadata schemas without
shy-core coordinating every release — the same convention `npm`
uses for `package.json` (`"eslint"`, `"babel"`, and similar
plugin-specific keys are tolerated).

Plugin authors who need durable metadata place it under a section
named after their plugin (e.g. `[kebab-conformance]`) and read it
back via the JSON API. shy itself never interprets these sections.

Minimal single-script form:

```toml
name = "git-autofetch"
version = "1.0.0"
description = "Run git fetch in background when entering a repo"
license = "MIT"
type = "script"
entry = "./git-autofetch.sh"

[source]
repo = "alice/git-autofetch"

[requires]
bash = ">=4"
binaries = ["git"]
```

Minimal collection form:

```toml
name = "alice-default"
version = "2.5.0"
description = "My personal shy setup"

[source]
repo = "alfred-intelligence/shy-setup"
```

Explicit collection with inline items:

```toml
name = "alice-default"
version = "2.5.0"

[source]
repo = "alfred-intelligence/shy-setup"

[[items]]
name = "git-autofetch"
type = "script"
path = "./scripts/git-autofetch.sh"

[[items]]
name = "gh-clone"
type = "plugin"
command = "gh-clone"
path = "./plugins/gh-clone.sh"
description = "Clone a GitHub repo with default org"

[[items]]
name = "ll"
type = "alias"
value = "ls -alh --group-directories-first --color=auto"

[aliases]
la = "ls -A"
gst = "git status -sb"

[[completions]]
tool = "kubectl"
generate = "kubectl completion bash"

[[dependencies]]
source = "github:bob/git-helpers"
constraint = "^1.0"
type = "required"

[capabilities]
# Reserved for v2 sandboxing; parsed and ignored at runtime in v1.
# `shy audit` plugin (v1.x) reads this for static-vs-declared analysis.
network = ["github.com", "api.github.com"]
binaries = ["git", "gh"]
filesystem = ["$HOME/repos"]

[security]
# Optional; declares this release as a security update.
# v1: trust-based. v2: requires verified CVE reference (see v2 refactor).
fixes = "CVE-2026-12345"
severity = "high"
description = "Fix path traversal in cd-tree handler"
```

Manifest fields:

| Field | Required for | Notes |
|---|---|---|
| `name`, `version` | All packages | SemVer |
| `description`, `license` | Public packages | Free-form text; SPDX identifier |
| `type` | Single-item form | `"script"` (default), `"plugin"`, `"alias"`, `"completion"` |
| `entry` | Single-item scripts/plugins | Path to `.sh` relative to manifest |
| `command` | Plugin items | Subcommand exposed via `shy <command>` |
| `value` | Alias items | The aliased expansion |
| `generate` | Completion items | Command whose stdout is captured |
| `[source]` | Published packages | Determines installation namespace; missing means local |
| `[[items]]` | Multi-item collections | Each entry is an item with type-specific fields |
| `[aliases]` | Any | Inline name → value pairs |
| `[[completions]]` | Any | Equivalent to `[[items]] type="completion"` |
| `[[dependencies]]` | Any | External packages with `required`, `recommended`, `optional` |
| `[requires]` | Any | Runtime checks (bash version, binaries on PATH) |
| `[capabilities]` | Reserved | Declared capabilities for v2 sandboxing; parsed and ignored in v1; read by `shy audit` plugin |
| `[security]` | Updates | Marks an update as a security fix; bypasses update-check throttle and snooze |

Item-type validation is performed by the binary at install time.
`type = "plugin"` requires `command`; `type = "alias"` requires
`value`; `type = "completion"` requires `tool` and `generate`.

### Behaviour when manifest is missing

A script directory without a manifest is fully usable at runtime:
`init.bash` walks the directory and sources every `.sh` file (other
than those prefixed with `_`). The shell sees the script's
functions, aliases, and exports normally.

CLI commands behave as follows when manifest is missing:

- **`shy list`** shows the item with description `(no manifest)`
  and version `—`. Still visible, still removable.
- **`shy info <namespace>/<name>`** shows README.md if it exists;
  otherwise prints a hint that the script has no manifest and no
  README.
- **`shy remove <namespace>/<name>`** works identically.
- **`shy update <namespace>/<name>`** refuses with a clear message
  pointing to `shy publish`.
- **`shy publish <namespace>/<name>`** generates a manifest
  interactively (described in detail under "Publication" below).

The principle: manifest is for sharing across machines and users.
A locally placed script that never needs to leave the machine
never needs a manifest.

### Publication

`shy publish` initialises a script as a publishable git repository
and generates its manifest. The flow handles three git states with
different severity:

**State 1 — No git anywhere (no error, proceed silently):**

shy runs `git init`, makes an initial commit, prompts for manifest
details (description, license, version defaulting to `0.1.0`),
writes `manifest.toml`, and moves the directory to its namespace.

```
$ shy publish my-script
shy: initializing git in ~/.shy/scripts/macbook-pro-2/my-script/
shy: committed initial state.
shy: namespace will be alice (from git config user.name).
shy: prompting for manifest details...

Description: A wrapper around cd that fetches in git repos
License [MIT]: <enter>
Version [0.1.0]: <enter>

shy: written manifest.toml
shy: moved to ~/.shy/scripts/alice/my-script/
```

**State 2 — Script is its own git repo root (minor, inform and
continue):**

shy notes the existing repo, reports working-tree cleanliness and
commits-since-last-publish, then proceeds. Version inference uses
Conventional Commits from commit messages since the previous
manifest version.

```
$ shy publish my-script
shy: my-script is already a git repository.
shy:   working tree clean: yes
shy:   commits since last publish: 3 (1 feat, 1 fix, 1 chore)
shy: continuing with publish.
shy: suggested version: 1.0.0 → 1.1.0 (minor bump)
Accept? [Y/n/edit] _
```

If the working tree is dirty, the report includes a non-blocking
warning:

```
shy:   working tree clean: NO — 2 uncommitted files
shy:   commits since last publish: 3 (1 feat, 1 fix, 1 chore)
shy: continuing with publish (uncommitted changes will not be
shy: included in version inference).
```

**State 3 — Script is inside a parent git repo (major, abort with
info):**

shy refuses to publish a directory that lives under another git
repository's `.git/`. The error explains why and gives the
operator two paths forward.

```
$ shy publish my-script
shy: cannot publish my-script: directory is inside another git repository.

  Script directory:  ~/.shy/scripts/macbook-pro-2/my-script/
  Parent .git/ at:   ~/dotfiles/.git/

  Published scripts must live in their own git repository at the
  script's root. Publishing a subdirectory of another repository
  creates ambiguous source-tracking and version inference.

  To publish my-script as its own repository:
    1. Move the script directory outside of any existing repo
       (e.g., copy ~/.shy/scripts/macbook-pro-2/my-script/ to a
       new location)
    2. Then run: shy publish my-script

  To keep my-script local (no publish required):
    Local scripts work without a manifest. No action needed.

$ # exit code 1
```

**Exit codes** follow Unix convention: State 1 and 2 exit 0
(success or informational); State 3 exits 1 (operation aborted).

**Git user.name requirement.** All publication states require
`git config --global user.name` to be set. shy refuses to publish
without it and prints:

```
shy: cannot publish: git user.name is not set.
shy: run: git config --global user.name "<your-github-handle>"
```

**Push integration.** After publish completes locally, shy offers
two paths:

- `shy publish <name> --to-github` — uses `gh` if installed to
  create `github.com/<user.name>/<script-name>` and push the repo.
  Without `gh`, this option fails informatively.
- Manual `git remote add origin <url> && git push -u origin main`
  for operators using non-GitHub hosts or custom workflows.

**Version inference via Conventional Commits.** For State 2 and
subsequent publishes, shy parses commit messages since the last
`manifest.toml` version tag:

- `feat:` → minor bump (1.2.0 → 1.3.0)
- `fix:` → patch bump (1.2.0 → 1.2.1)
- `feat!:` or `BREAKING CHANGE:` → major bump (1.2.0 → 2.0.0)
- `chore:`, `docs:`, `style:` → no bump

This is the release-please algorithm executed locally. Operators
who don't follow Conventional Commits get the prompt to enter a
version manually. `shy publish --version 2.5.0` overrides both
inference and prompt.

### Plugin model

Plugins are scripts the binary dispatches as subcommands rather
than sourcing into the shell. A plugin manifest item declares a
`command` field; when the operator runs `shy <command>`, the
binary:

1. Looks up native subcommands first (install, init, list, etc.).
2. If not native, walks plugin manifests under
   `$HOME/.shy/plugins/<namespace>/<name>/manifest.toml` for an item
   with matching `command`.
3. Executes the matched entry script with the remaining arguments,
   inheriting the operator's environment.

Plugins are *the* extension point for v1. Features that might
otherwise be debated as native additions can be implemented as
plugins first. If a plugin proves widely useful and stable, it is
promoted to native in a later release; if it does not, it stays a
plugin and costs the binary nothing.

### Plugin API conventions

Plugins interact with shy's installed state through the binary's
JSON API, not by reading `cache.json` or walking filesystem
directly.

**Read API** (idempotent, no side effects):

```bash
shy list --json
shy list --type=plugin --json
shy info <namespace>/<name> --json
shy collection list --json
shy outdated --json
```

**Write API** (plugin-initiated state changes):

```bash
shy install <path> --silent
shy remove <namespace>/<name> --silent
```

**`cache.json` is private.** Plugins do not read or write it. The
schema can change between versions without breaking plugins that
use the API.

This API contract is the forward-compatibility surface for v2
sandboxing: when plugin sandboxing is added, plugins that already
follow the convention work unchanged; plugins that read filesystem
directly need fixes.

### Plugin completion conventions

Plugins exposing arguments via `shy <command> <tab>` declare
completions through one of two opt-in conventions. Both are
independent of the manifest.

**Convention 1 — `__complete` subcommand within the plugin:**

```bash
#!/usr/bin/env bash
if [[ "$1" == "__complete" ]]; then
    shift
    case "$#" in
        0) gh repo list --limit 50 --json nameWithOwner -q '.[].nameWithOwner' ;;
        1) git ls-remote --heads "https://github.com/$1.git" 2>/dev/null \
             | awk '{print $2}' | sed 's|refs/heads/||' ;;
    esac
    exit 0
fi
gh repo clone "$1" "$2"
```

shy invokes `<plugin-entry> __complete <prev-args>` with a 100ms
timeout. Stdout output, one per line, becomes the candidate list.

**Convention 2 — Header directives in the script:**

```bash
#!/usr/bin/env bash
# @shy:complete-1: shell:gh repo list --limit 50 --json nameWithOwner -q '.[].nameWithOwner'
# @shy:complete-2: shell:git branch -r | sed 's|.*origin/||'
# @shy:complete-3: static:main develop master

gh repo clone "$1" "$2"
```

shy parses these lines via simple regex. `<type>` is either
`shell:` (execute the value as a command) or `static:` (space-
separated literal list). Regex-extractable — deterministic for
non-AI tooling.

**Resolution order** at `shy <plugin> <tab>`:

1. Try `<plugin-entry> __complete <args>` with 100ms timeout.
2. Otherwise, parse `@shy:complete-N:` headers.
3. Otherwise, no completion.

### Update notifications

shy notifies the operator about available updates through a
non-intrusive footer-line model. Inspired by `gh`'s pattern, but
implemented natively via Go's net/http against the GitHub Releases
API — no dependency on `gh` or any external CLI.

**Normal update notifications:**

```
$ shy list
SCRIPTS:
  alice/git-autofetch       1.0.0    Run git fetch on cd
  bob/cool-helper         2.1.0    A useful helper

shy: 1 update available — run `shy outdated` to view.
```

Mechanics:

- Cache duration: **7 days** (configurable via
  `SHY_UPDATE_CHECK_INTERVAL=168h`). When the cache is older, the
  next shy command triggers a background check.
- Footer notification respects "seen" status. Once the operator
  runs `shy outdated` or `shy collection update`, the notification
  is suppressed until a newer version appears.
- Per-item snooze via `shy snooze <namespace>/<name> --for 30d` or
  `--until v3.0`.
- Globally disabled with `SHY_UPDATE_CHECK=off`.

**Security update notifications:**

Items whose latest version declares a `[security]` section in the
manifest bypass both the cache throttle and any snooze state. They
are displayed prominently:

```
$ shy list
SCRIPTS:
  alice/git-autofetch       1.0.0    Run git fetch on cd

⚠ security update available: alice/git-autofetch 1.0.0 → 1.0.1
  CVE-2026-12345 (high) — Fix path traversal in cd-tree handler
  Run `shy collection update --security` to apply.
```

`SHY_UPDATE_CHECK=off` does **not** suppress security
notifications — they are the only category of notification that
cannot be disabled by configuration. This trade-off (small user
friction, large security benefit) is acceptable for v1.

**v1 trust model**: shy trusts upstream `[security]` claims at
face value. Authors who lie about security severity to force
upgrades face only social consequences. This is intentional v1
simplicity, paired with a documented v2 refactor toward
deterministic verification.

**v2 refactor (planned):** `[security]` requires a verifiable CVE
reference (e.g., `fixes = "CVE-2026-12345"`). shy queries the NVD
or GitHub Advisory Database to confirm the CVE exists and is
publicly recorded. Severity levels become enforced (`critical`
ignores all throttle; `high` ignores snooze; `medium`/`low`
respect snooze). False CVE references downgrade severity
silently.

### Documentation conventions

Documentation is decoupled from the manifest. A script or plugin
that wants documentation places a `README.md` next to its
`manifest.toml` (the standard npm/pip/cargo convention).

```
scripts/alice/git-autofetch/
├── git-autofetch.sh
├── manifest.toml
└── README.md
```

For larger documentation that needs multiple files:

```
plugins/alice/big-plugin/
├── big-plugin.sh
├── manifest.toml
├── README.md           ← entry, short intro
└── docs/               ← optional, for longer content
    ├── USAGE.md
    ├── CONFIGURATION.md
    └── EXAMPLES.md
```

The CLI surfaces documentation through:

- **`shy info <namespace>/<name>`** — renders README.md in the
  terminal using the embedded `glamour` library.
- **`shy info <namespace>/<name> --raw`** — pipes raw markdown to
  stdout.
- **`shy create <name>`** generates a README.md stub alongside the
  `.sh` and `manifest.toml`.
- **`shy publish`** warns (does not block) if README is missing or
  is just the default stub.

### Help, manuals, and completion

Four surfaces, all generated by `cobra` and packaged through
GoReleaser.

| Surface | How | Where it lands |
|---|---|---|
| `shy --help` / `shy <cmd> --help` | `cobra` auto-generates from subcommand definitions | Built into binary |
| Man-pages (`man shy`, `man shy-install`) | `cobra.GenManTree()` runs at release time | Packaged in `.deb`/`.rpm` via GoReleaser nfpms, installed to `/usr/share/man/man1/` |
| `shy info <name>` for installed items | Glamour renders README.md from item directory | Built into binary |
| Bash-completion for `shy` itself | `shy completion bash` (cobra-generated) | Auto-installed to `$HOME/.shy/completions/shy` during `shy init`, via shy's own completion mechanism |
| Plugin argument completion | `<plugin> __complete` subcommand OR `@shy:complete-N` header directives | Resolved by shy at tab time |

### Distribution

`shy` is published in three places:

- **GitHub Releases** — pre-built binaries per OS/arch via
  GoReleaser (`shy_<version>_<os>_<arch>.tar.gz`). Consumed by
  `install.sh`.
- **`install.sh` at a stable URL** — `curl | bash` entry point.
  The GoReleaser asset name schema is frozen at v1.0.0.
- **Distribution packages** (`.deb`, `.rpm`) — built by GoReleaser
  for system-wide installation.

The shy binary itself uses [release-please](https://github.com/googleapis/release-please)
in CI to compute its own next version from Conventional Commits.
This is the same algorithm shy applies to user-authored scripts at
`shy publish` time — internal consistency.

## Design decisions and rationale

| Decision | Alternatives considered | Rationale |
|---|---|---|
| Go binary for CLI | Pure bash, Rust, Python | Subcommands, JSON/TOML handling, error types; Go is the author's primary language; static binaries with no runtime dependencies |
| Bash for `init.bash` and `install.sh` | Pure Go | `init.bash` must be sourced into the user's shell; `install.sh` runs before any binary exists |
| `curl \| bash` is user-only by default | Auto-detect via root | Predictability beats cleverness |
| Distro packages contain binary only | Pre-seed content in packages | Decouples binary distribution from content seeding; sysadmin opts into seeding explicitly |
| `/etc/skel/.shy/` for new user seed | Custom `/usr/share/shy/` content area | Reuses OS-native mechanism for new-user initialisation instead of duplicating it |
| Sudo required only for `system-install`/`system-uninstall` | Sudo for any system-affecting command | shy is per-user; only the `/etc/skel/` seed needs root |
| Overrides as user-owned | System-owned alternative | Single-layer model — overrides live alongside user content |
| `shy init` creates empty area | Copy seed from system location | No system content to seed from; user-area is self-contained |
| TOML manifest | JSON, YAML | Hand-editable, less syntactic noise |
| Manifest is metadata, not configuration | Manifest authoritative for runtime | Runtime depends on filesystem structure |
| Unified manifest schema (single + multi) | Two schemas | One mental model; binary infers form from content |
| Namespacing for scripts and plugins | Flat directories | Two collections can publish same name without collision |
| No namespacing for aliases and completions | Symmetry with scripts | Aliases are last-defined-wins regardless of disk layout |
| Helpers via `_`-prefix on filenames | Manifest `entry` authoritative | Convention is independent of manifest; opt-in |
| Documentation via README.md beside manifest | Field in manifest | Standard npm/pip/cargo convention |
| Glamour embedded for markdown rendering | Delegate to glow/mdcat/cat | Consistent UX without external dependency |
| Cobra for help, man-pages, completion | Hand-rolled | Standard pattern in Go CLIs |
| Plugin completion via `__complete` + header directives | Manifest fields | `__complete` is the git/kubectl/gh pattern; headers are regex-extractable for non-AI tooling |
| Plugin declared via manifest field | Filename convention | Explicit, robust against renaming, schema-validatable |
| Plugin API via `--json`/`--silent` flags | Direct filesystem access to `cache.json` | Stable contract; forward-compat with v2 sandboxing |
| `cache.json` is private | Plugin-readable internal state | Schema can evolve without breaking plugins |
| Collections subscribed via git | Custom sync protocol | Sync as emergent property of git distribution |
| GitHub primary distribution | Multi-host neutral | Optimised UX (`@user/repo` syntax, GoReleaser integration) |
| `install.sh` permanent contract | Reversioned installers | One URL anyone can curl forever |
| Plugin architecture in v1.0 | Plugin support in v1.x | Plugins are the off-ramp for feature creep |
| Extensible manifest parser | Strict schema rejecting unknown sections | Plugins evolve their own metadata schemas without shy-core releases; npm/cargo precedent |
| `[capabilities]` reserved in manifest, deferred to v2 | Enforce from v1, omit entirely | Audit plugin reads it for static-vs-declared analysis |
| Default-pinning at `shy install @user/repo` | Follow main branch | Security and reproducibility |
| `shy collection update --dry-run` default | Apply directly | Operators see diffs before they hit shell |
| `shy list --sources` | Implicit ownership tracking | Insight into where every item came from |
| Gitops as primary v1 security model | Sandbox-from-start, central trust authority | Existing strong foundation before sandboxing |
| Audit plugin before sandboxing | Sandbox first | Static analysis is immediately useful without sandbox infrastructure |
| `chmod 700` on `$HOME/.shy/` | Default permissions | Protects against other users on shared hosts |
| `shy publish` requires own git repo | Allow nested repos via parent detection | Renlighet; ambiguous source-tracking and version inference avoided |
| Implicit `git init` when none exists | Prompt operator first | "Publication implies git" — no ceremony for common case |
| Conventional Commits for version inference | Manual version always | Reuses release-please algorithm; falls back to prompt if no commits |
| `[security]` tag bypasses throttle | Same throttle as normal updates | Security fixes are the one category where tjat is the lesser evil |
| Update-check trust model in v1, CVE-verified in v2 | Verify from start | v1 simplicity; abuse not yet documented |
| 7-day update check cache | 24-hour (gh default) | Less invasive for personal-use cadence |
| Footer-line update notification | Box, banner, or explicit-only | Informative without visual noise |
| No GPG signing in v1 | Sign from start | User is trust root for own collections |
| MPL-2.0 for code, CC-BY-SA 4.0 for docs | Single licence for both | File-level copyleft for code, share-alike for documentation symmetry |

## Security model

`shy` v1 rests on **three pillars**: gitops, default-pinning, and
operator discipline.

### Pillar 1 — Gitops

Every snippet, alias, completion, and plugin is a file in git
somewhere. Everything is auditable, diff-able, and reversible.

### Pillar 2 — Default-pinning

`shy install @user/repo` pins to the current HEAD commit by
default. `shy collection update` defaults to `--dry-run`. Operators
see what would change before it touches their shell.

### Pillar 3 — Operator discipline

- Trust collections like apt repositories.
- Never store secrets in snippets.
- Run `shy list --sources` regularly.
- Prefer plugins over scripts for non-global functionality.
- Avoid `sudo NOPASSWD`.

### Filesystem permissions

`shy init` creates `$HOME/.shy/` with `chmod 700`. Blocks other
users on shared hosts from reading the operator's scripts or
aliases.

### Blast radius

A `.bashrc`-sourced snippet runs in the user's interactive shell
with full UID privileges. This is the same blast radius as any
code the user runs.

### Architectural limit: scripts cannot be sandboxed

Plugins are dispatched on demand and *can* be sandboxed (planned
for v2 via bubblewrap or firejail with `[capabilities]`-declared
permissions). Scripts cannot be — they must be sourced into the
operator's interactive shell to function. Operator discipline is
the *only* security mechanism for sourced scripts, in every version
of shy forever.

### Trigger for next-level mitigations

shy's primary security advice is "subscribe to collections you
trust personally". This is sufficient as long as the operator can
hold the list in their head and audit each entry.

**When the operator starts losing track** — too many subscribed
collections, too many transitive dependencies, too many scripts
from authors they've never personally vetted — that is the trigger
for the next layer: a `shy audit` plugin (v1.x).

`shy audit` performs static analysis on installed scripts and
plugins, flagging suspicious patterns. It reports, it does not
enforce. The operator decides what to do with the findings.

### Plugin sandboxing (planned for v2)

When `shy audit` reveals consistent gaps between declared
`[capabilities]` and actual plugin behaviour, the next layer is
runtime sandboxing via `bubblewrap` (Linux) or `firejail`.

### Daemon architecture (hypothetical v3)

Considered only if one of these specific triggers occurs:

- Bubblewrap or firejail CVEs at high frequency.
- Plugins need persistent shared state that filesystem cannot
  provide cleanly.
- Concurrency corruption between parallel shy invocations.
- Plugin ecosystem exceeds 50+ active plugins with diverse
  capability needs that gitops + audit + sandbox cannot govern.

Not built without one of these triggers.

### Does not defend against

- Compromised git hosts serving malicious collection content
- Compromised binaries (no signing in v1)
- Local privilege escalation through plugin scripts
- Compromised operator
- Compromised sourced scripts (architectural limit)

## Dependencies

- **Bash 4+** for snippets that use associative arrays
- **`git`** for collections and `shy update`
- **`curl` or `wget`** for `install.sh` and fetching collection
  references
- **`sha256sum`** for asset verification by `install.sh`
- **`tar`** for unpacking release archives
- **A POSIX shell** for `install.sh` and `init.bash`
- **GitHub access** for primary distribution
- **GoReleaser** for building releases (CI dependency, not runtime)

No external services. No telemetry. No phone-home.

## Limitations and risks

**Sourced scripts cannot be sandboxed in any version of shy.** This
is architectural, not implementation-defer.

**Security patches require user action** for non-`[security]`-
tagged items. `[security]`-tagged patches bypass throttle and
appear immediately.

**Overrides are additive, not subtractive.**

**Alias and completion conflicts are surfaced explicitly** at
install/subscribe time, since they are not namespaced.

**GitHub Releases as single-point-of-failure for `install.sh`.**
Building from source remains possible.

**Plugin discovery cost.** Each `shy <command>` walks plugin
manifests. Cached in `cache.json`; rebuilt on plugin install/
remove.

**`install.sh` is a permanent contract.** Asset name schema cannot
be changed.

**macOS default bash is 3.2.** Some snippets will fail. Mitigated
by `[requires.bash]` declarations.

**Collection author can change content after subscription.** Default
pinning mitigates for explicit installs.

**`[security]`-tag is trust-based in v1.** Authors who falsely tag
non-security updates as security to force notification face only
social consequences. v2 refactor toward CVE verification is
planned.

**Solo-developer focus may not generalise.** Multi-tenant scenarios
out of scope.

## Future direction

v1.0 covers the design above: solid mechanics for managing bash
snippets within a single user's ecosystem.

Reserved future directions, grouped by scope:

### v1.x — plugins shipped on the v1 base

- **`shy audit` plugin.** Static analysis of installed scripts and
  plugins; flags gaps between declared `[capabilities]` and actual
  code (eval of untrusted input, network calls to undeclared hosts,
  reads of sensitive paths, subprocess spawns to undeclared
  binaries). Reports; does not enforce.
- **Auto-completions plugin** (`@alfred-intelligence/auto-completions`).
  Weekly scanner that maps installed binaries on `$PATH` against
  an index of known completion-generation patterns.

### v2 — workspace shy

The v2 vision is shy as a terminal workspace, not just a CLI tool.
Concretely: a multiplexed terminal environment with shy as the
controller layer, tmux as the backend, and named windows for shell
interaction, AI assistant (local Ollama and/or Claude Code), git
repository browser, and an error console.

- **Plugin sandboxing via bubblewrap/firejail.** Enforces declared
  `[capabilities]` at plugin runtime. Scripts remain unsandboxed by
  architecture.
- **Workspace-sandbox** (outer) and plugin-sandbox (inner) as a
  two-layer defence model. Scripts sourced inside the workspace
  become sandboxed via the outer layer — the architectural fix
  to v1's "scripts can never be sandboxed" limitation.
- **Named persistent sessions** (`shy workspace --new dev`,
  `--attach`, `--list`, `--kill`). Each session has isolated state:
  separate AI history, separate Claude Code session ID, separate
  bash history, separate layout.
- **Error console window.** Observability surface showing runtime
  events (info/warn/error/fatal) from workspace, plugins, and
  resource monitors. Mini-DSL for filtering
  (`level>=warn source=plugin/* since=5m`), colour-coded by level
  and source, with a resource watchdog that flags memory balloons
  and sustained CPU.
- **`[security]` tag CVE verification.** Deterministic verification
  against NVD or GitHub Advisory Database; severity levels become
  enforced rather than trust-based.
- **Zsh and fish support.** Cobra's completion generation already
  covers both; runtime sourcing layer needs a zsh/fish variant of
  `init.bash`.

### Out of shy-core, into plugin scope

Two items previously sketched as v2-core have been moved to
independent plugin scope. They are not part of shy's roadmap; they
are separate projects that may exist in the `alfred-intelligence`
ecosystem and integrate via shy's plugin model and extensible
manifest parser.

- **`@alfred-intelligence/shy-kebab-conformance`.** Cross-ecosystem
  variable-namespace convention enforcement. Plugins read a
  `[kebab-conformance]` section from item manifests declaring which
  named convention the item follows; the plugin verifies semantic
  compatibility between installed items. Couples with kebab-it's
  librarian agent, which extracts convention candidates from
  real-world infrastructure tasks. Speculative: solves a problem
  that exists at substantial ecosystem scale, not at v1 scale.
- **`@alfred-intelligence/shy-sign`.** GPG signing for plugin and
  collection releases, modelled on kebab-it-core's signing flow.
  Operates as a verification plugin invoked at install or update
  time.

### Hypothetical v3

Daemon mode (similar to dockerd: socket, privilege boundary,
mediated access) is theoretically possible but gated on specific
triggers documented in the security section. Not planned.

## Assumptions for Phase B

These were assessed silently during Phase A and have been confirmed
at the start of Phase B operationalisation.

- **Licence for code: MPL-2.0.** File-level copyleft; preserves shy
  as open source while allowing plugin/integration code to use any
  licence.
- **Licence for documentation: CC-BY-SA 4.0.** Share-alike for the
  whitepaper, design documents, and other documentation in the
  repository. Symmetric with MPL-2.0's copyleft principle — work
  derived from shy's documentation must remain under the same
  licence.
- **Strictness level: `solo+contrib`.** Operator-primary; external
  PRs welcome but not the default audience.
- **Branch strategy:** four-branch time-sequenced model. `main` is
  stable and reflects the latest tagged release; `next` is active
  development for the upcoming release; `after` is experimental
  post-v1 work (workspace, sandboxing); `before` is backport-only
  for fixes to older stable releases. See `05-engineering-
  handbook.md` for full lifecycle. During v0.x (pre-1.0) the
  operator may push directly to `main` for documentation and
  scaffolding; strict automation-only enforcement applies from
  v1.0 onward.
- **Release cadence:** tagged releases via GoReleaser, automated
  versioning via release-please. No fixed schedule.
- **Conventional Commits** for changelog generation.
- **`CODE_OF_CONDUCT.md` and `SECURITY.md`:** present from the
  start, even during private phase.
- **`CONTRIBUTING.md`:** short.
- **Repository: `github.com/alfred-intelligence/shy`.** Organisation
  namespace, not personal.
- **Visibility:** private during v0.x development. Repository
  becomes public at v1.0 release. All public-facing documentation
  (README, CODE_OF_CONDUCT, SECURITY, CONTRIBUTING, issue templates)
  is written for public context and waits in repo until release.
