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

## Solution

`shy` is one CLI binary plus a thin shell layer.

**1. The binary `shy`.** Written in Go, distributed as a static
single-file release. Handles all operations: install, remove,
update, list, create, publish, init, override, system-reset, info,
outdated, snooze, upgrade, and plugin dispatch. The binary embeds
`glamour` for rendering README files inline and `cobra` for
subcommand structure, help text, man-page generation, and shell-
completion generation. It exposes a structured API to plugins via
`--json` on read commands and `--silent` on write commands.

Two complementary installation channels:

- **User-level install** via `curl | bash`. Lands in
  `$HOME/.shy/bin/`. Targets the user who ran the command. No root
  required. The installer script lives at a stable URL and remains
  backward compatible forever.
- **System-level install** via distribution packages (`.deb`,
  `.rpm`). Lands in `/usr/share/shy/` and `/usr/bin/shy`. Targets
  all users on the machine via standard FHS paths. Man-pages
  installed to `/usr/share/man/man1/`.

Both can be installed simultaneously; the user binary wins via
`$PATH` ordering.

**2. The shell layer.** A small `init.bash` file sourced from
`~/.bashrc`. It walks `$HOME/.shy/scripts/`, `aliases/`,
`completions/`, and `overrides.d/` and sources each file. One bad
file does not break shell startup — errors are reported per-file to
stderr while sourcing continues.

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

### Three locations, two runtime layers

```
/usr/share/shy/                  ← System seed (root-installed via apt/dnf)
├── scripts/                     ← Never sourced directly at runtime
│   └── <namespace>/<name>/      ← Copied to user via `shy init`
├── plugins/                     ← Same namespacing
├── aliases/                     ← Flat files (no namespacing)
├── completions/                 ← Flat files (no namespacing)
└── overrides.d/

$HOME/.shy/                      ← User canonical, chmod 700 by default
├── bin/shy                      ← Binary (if installed user-level)
├── init.bash                    ← Sourced from ~/.bashrc
├── scripts/                     ← Sourced at runtime, first
│   └── <namespace>/<name>/      ← *.sh files sourced; _*.sh skipped
├── plugins/                     ← Not sourced; dispatched on `shy <command>`
│   └── <namespace>/<name>/
├── aliases/                     ← Flat files, all sourced
├── completions/                 ← Flat files, all sourced
├── overrides.d/                 ← Sourced at runtime, last (wins on conflict)
│   ├── scripts/<namespace>/<name>/
│   ├── aliases/
│   └── completions/
├── collections/                 ← Cloned subscribed collections
└── cache.json                   ← Internal runtime cache; not a plugin API
```

### Namespacing strategy

**Scripts and plugins are namespaced; aliases and completions are
not.**

A script lives at `$HOME/.shy/scripts/<namespace>/<name>/<name>.sh`.
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

### How `shy init` interacts with the layers

`shy init` is the seed-mirror command. It:

1. Creates `$HOME/.shy/` with `chmod 700` (protects against other
   users on shared hosts).
2. Creates subdirectories under `$HOME/.shy/` with standard
   permissions inherited from the umask.
3. Copies all files from `/usr/share/shy/` into `$HOME/.shy/`,
   skipping any file that already exists in the user location.
   Re-running is idempotent: new files from a system upgrade land
   in the user location, existing user files remain untouched.
4. Auto-installs shy's own bash-completion to
   `$HOME/.shy/completions/shy` via its own completion mechanism
   (`shy completion bash > $HOME/.shy/completions/shy`).
5. Writes the source line to `~/.bashrc` if not already present.

`sudo shy init` is deliberately disabled and prints a hint pointing
to `shy system-reset` (the explicit destructive command for
restoring a clean state across all users on the machine).

### Runtime layer order

At every shell start, `init.bash` sources files in two layers:

1. **User layer** — `$HOME/.shy/{scripts,aliases,completions}/`.
   The user's canonical content.
2. **Overrides** — `$HOME/.shy/overrides.d/{scripts,aliases,completions}/`.
   Sourced after the user layer; later wins by standard bash
   semantics.

System install never sources directly; it is purely a seed source
for `shy init`. The user is always above the system at runtime.

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
| `[[conformance]]` | Reserved | Future Layer 2; ignored in v1 |

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
| `curl \| bash` is user-only | Auto-detect via root | Predictability beats cleverness |
| System install via distro packages | One channel, multiple use | apt/dnf handle permissions, dependencies, upgrades |
| User always above system at runtime | Mixed precedence | Consistent with XDG and pip conventions |
| Overrides as user-owned | System-owned at `/etc/shy/overrides.d/` | Symmetry with the rest of `$HOME/.shy/` |
| `shy init` copies seed, skip-on-conflict | Replace user files | Protects user customisation |
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
| Layer 2 reserved in manifest, deferred to v2 | Build now, build never | `[[conformance]]` accepted but ignored in v1 |
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

Reserved future directions, in rough priority order:

- **`shy audit` plugin** (v1.x). Static analysis of installed
  scripts and plugins; flags gaps between declared
  `[capabilities]` and actual code.
- **Plugin sandboxing via bubblewrap/firejail** (v2). Enforces
  declared `[capabilities]` at plugin runtime.
- **`[security]` tag CVE verification** (v2). Deterministic
  verification against NVD or GitHub Advisory Database; severity
  levels become enforced.
- **GPG signing for binary releases** (v2).
- **Layer 2 — Convention namespace** (v2). `[[conformance]]`
  sub-schema for cross-ecosystem snippet portability.
- **Auto-completions plugin** (v1.x). `@alfred-intelligence/auto-completions`.
- **Zsh and fish support** (v2+).

Daemon mode is a hypothetical v3, gated on the specific triggers in
the security section.

## Assumptions for Phase B

These were assessed silently during Phase A and have been confirmed
at the start of Phase B operationalisation.

- **Licence: MPL-2.0.** File-level copyleft; preserves shy as open
  source while allowing plugin/integration code to use any licence.
- **Strictness level: `solo+contrib`.** Operator-primary; external
  PRs welcome but not the default audience.
- **Branch strategy:** stable/next pattern. `main` is stable and
  reflects the latest tagged release; `next` is active development
  and the default branch for PRs. Release-please runs on `next`;
  post-release workflow merges `next` → `main` automatically.
  Hotfixes cherry-pick to `next` and follow the standard release
  flow.
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
