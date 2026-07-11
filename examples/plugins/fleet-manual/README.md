# fleet-manual

`man`/`tldr`/`who` navigator for the Claude agent colleague-stab roster
(`~/.claude/agents/`). Board task #55; design draft in
[`DESIGN.md`](DESIGN.md); forks A/B/C resolved by the operator
2026-07-11 (see `DESIGN.md`'s Status line).

## Install

From a checkout of this repository — **three separate installs**, one
per subcommand (see "Why three packages, not one multi-item manifest"
below for why this isn't a single `shy install ./fleet-manual`):

```bash
shy install ./examples/plugins/fleet-manual/man
shy install ./examples/plugins/fleet-manual/tldr
shy install ./examples/plugins/fleet-manual/who
```

Each installs as its own bare top-level command (Fork A resolved as
A1: bare names, not `shy fleet man`) — `shy man`, `shy tldr`, `shy
who`. All three share `_lib.sh` (root of this directory) via a
per-package symlink, so the roster/frida/rendering logic lives in one
place.

## Usage

```bash
$ shy man governator
governator

Governance-conformance linter (call sign "Governator") for the
alfred-intelligence org and fleet. ...
[... full charter, Prompt Defense Baseline section elided ...]

$ shy tldr governator
governator — Governance-conformance linter (call sign "Governator") for the alfred-intelligence org and fleet.
triggers: Use for scheduled conformance sweeps and flag-triggered re-verification against alfred-intelligence/.github-private/DECISIONS.
NOT for: NOT a general code/security reviewer (that's inquisitor); NOT the one who builds or fixes CI/branch-protection/rulesets (that's priest)
dispatch: delegate to governator via the colleague-stab (dante-ops/CLAUDE.md roster table)
confidence: description-derived, verify against /home/gg/.claude/agents/governator.md if unsure

$ shy who "branch protection ruleset"
best match: priest        (2/3 words hit)
also:       governator     (2/3 words hit)

$ shy man frida
frida
[... her charter — explicit lookup by name always works ...]

$ shy man   # no args: listing — frida is absent
Colleagues:
  librarian
  priest
  ...
```

`man` and `tldr` also answer `shy man help` / `shy man --help` /
`shy man -h` (and the `tldr`/`who` equivalents) with usage text — see
"shy help & completion" below for why that's the plugin-side answer
rather than a `shy help` core change.

## Roster scope (board #55 item 2)

Filtered to the colleague-stab — the ~14-17 human-named colleagues in
`dante-ops/CLAUDE.md`'s roster table — via an explicit name allowlist
in `_lib.sh`'s `fleet_roster_names`, NOT the full 60+ agent directory
(which also holds generic ECC specialists like `code-reviewer`,
`tdd-guide`; those are invoked BY colleagues, not browsed by the
operator directly). Nothing in the frontmatter schema currently
distinguishes the two groups, so a named allowlist is the simplest
correct filter; a schema-level marker would be a bigger change out of
scope here.

## Hidden colleague — frida (binding, DESIGN.md §5)

`frida` is invisible-by-default everywhere in this plugin:

- `shy man` / `shy tldr` with no argument (the roster listing) never
  shows her.
- `shy who <phrase>` never matches against her description **unless**
  the phrase itself literally contains `frida` or `stash`
  (case-insensitive).
- `shy man frida` / `shy tldr frida` (explicit, direct lookup) **always
  works** — naming her IS the "query phrase names her explicitly"
  condition.
- Tab-completion (`completions/fleet-manual.bash`) offers her only
  once the typed prefix already begins her name (e.g. `fr<TAB>`); a
  bare `shy man <TAB>` with nothing typed does not surface her.

## Data flow

Pure bash (`awk`/`sed`/`grep`, no `jq`, no `python`) — see
`_lib.sh` for the frontmatter extractor, PDB-block stripper, tldr
pattern-matcher, and rendering fallback chain (`glow` → `bat` →
`cat`, all optional). Every invocation re-reads
`~/.claude/agents/*.md` fresh; no cache, no second roster registry
(DESIGN.md §3.1, §3.3).

## Why three packages, not one multi-item manifest (confirmed shy-core bug)

The first implementation used ONE `manifest.toml` with `[[items]]`
declaring all three commands (`man`, `tldr`, `who`) — the schema
`internal/manifest/manifest.go` and `examples/plugins/hello-world`'s
sibling `collectionManifest` test fixture both support, and the form
`DESIGN.md`'s own §7 sketch implied. It builds and shellchecks clean,
but **breaks at real dispatch**, verified end-to-end with a locally
built `shy` binary against a scratch `$SHY_HOME`:

- `internal/install/install.go`'s `installScriptOrPlugin` copies the
  entry-dir's **whole original `manifest.toml`** (still listing all
  three items) into *each* item's own install directory
  (`copyFileIfExists(filepath.Join(srcDir, "manifest.toml"), ...)`,
  not a per-item trimmed copy).
- `internal/plugin/plugin.go`'s `Discover()` walks every installed
  item directory and blindly re-parses **all** `type="plugin"` items
  out of whatever `manifest.toml` it finds there — so walking the
  `man/` install directory alone re-adds `tldr` and `who` as
  *additional* cache entries, each pointing at `man/entry.sh` (the
  only entry script actually present in that directory).
- `plugin.Lookup()` returns the *first* cache match for a command
  name. Because `os.ReadDir` visits item directories alphabetically
  and each directory's copied manifest lists items in `man, tldr, who`
  order, the first `tldr` entry found anywhere in the whole cache
  originates from walking `man/` — so it resolves to `man/entry.sh`.
  Confirmed by running the built binary: `shy tldr governator` printed
  the **full `man` charter**, and `shy who "branch protection
  ruleset"` printed **`man`'s "not in the roster" error** treating the
  phrase as a colleague name. `shy man` itself happened to resolve
  correctly only because it's alphabetically first *and* the first
  item in every copied manifest — not a load-bearing guarantee for any
  other multi-item plugin.
- Net effect: **any multi-item plugin manifest with 2+ `type="plugin"`
  items sharing an install namespace mis-dispatches for every item
  except (by luck, not design) the first**, and the plugin cache grows
  O(n²) (9 entries for this plugin's original 3 items, all logged in
  `shy help`'s `Plugins:` section 3× each).

This is a shy-core defect, not a design choice available to fix from
plugin-only bash — filed as a follow-up (`internal/install/install.go`
+ `internal/plugin/plugin.go`, likely fix: copy/synthesize a per-item
manifest containing only that item at install time, or have
`Discover()` filter to the item matching the directory name) rather
than patched here, per this task's "plugin only, zero shy-core edits"
constraint and `docs/04-agent-instructions.md`'s "new subcommand /
manifest schema change" approval boundary.

**Workaround adopted:** three independent single-item plugin packages
(`man/`, `tldr/`, `who/`, each mirroring `examples/plugins/hello-world`
exactly — one manifest, one command, one entry script). Each install
directory only ever contains its own single-item manifest, so
`Discover()` can't cross-contaminate. Verified fixed against the same
built binary: `shy man`/`shy tldr`/`shy who` each now resolve and
render correctly (see "Usage" above).

## shy help & completion: what shy-core actually supports today

Investigated against `internal/plugin/plugin.go`,
`internal/cmd/plugin_dispatch.go`, `internal/cmd/plugin_help.go`,
`internal/cmd/completion.go`, and `internal/cmd/root.go` on this
branch (board #55, 2026-07-11).

**`shy help` listing (item 5) — already works, no core change
needed.** `root.go` calls `installPluginHelp(root)`, which wraps
cobra's default help function to append a `Plugins:` section listing
every discovered plugin's `command` + `namespace/name` +
`description` (`plugin_help.go`). Once this plugin is `shy install`-ed,
`shy help` (and bare `shy` / `shy --help`) will list:

```
Plugins:
  man    <ns>/man    Full charter view for one fleet colleague — shy man <colleague>
  tldr   <ns>/tldr   5-line tldr summary for one fleet colleague — shy tldr <colleague>
  who    <ns>/who    Grep-based colleague routing suggestion for a task phrase — shy who <phrase>
```

because each manifest item's own `description` field feeds directly
into that section — the one-liners the task asked for come from the
manifest, not from any new plugin-side code.

**Per-command `shy man --help` usage text — a genuine, confirmed
gap.** Plugin commands are matched and exec'd in
`tryDispatchPlugin` (`plugin_dispatch.go`) *before* `root.ExecuteContext`
ever runs (`root.go` `Execute()`), so cobra never sees `man`/`tldr`/`who`
as real subcommands — there is no cobra `Command` to attach
`Long`/`Example` text to, and no core hook exists for a plugin to
register per-subcommand help into cobra's tree. The plugin-side
equivalent implemented here: each entry script handles `""`/`help`/
`--help`/`-h` itself (see `man.sh`/`tldr.sh`/`who.sh`).

**Tab-completion — also a genuine, confirmed gap.** The whitepaper
documents a `__complete` convention (`docs/01-whitepaper.md` "Plugin
completion conventions") and this plugin implements it in each entry
script for forward-compat, but grepping `internal/cmd/*.go` and
`cmd/*.go` for `__complete`, `ValidArgsFunction`, or a custom
`BashCompletionFunction` on the root command turns up nothing: `shy
completion bash` (`completion.go`) is a plain
`cobra.Command.GenBashCompletion` call with zero plugin awareness.
Because plugin commands are dispatched outside cobra entirely (same
reason as above), the generated completion script has no way to know
`man`/`tldr`/`who` exist, let alone shell out to their `__complete`.
The plugin-side equivalent shipped here:
`completions/fleet-manual.bash`, a static bash completion function
that recognises `shy man`/`shy tldr`/`shy who` directly and delegates
to shy's own `_shy` completion function for everything else (see that
file's header for the install line).

**Follow-up filed, not built here:** wiring `__complete` dispatch into
`shy completion bash`'s generated output (e.g. via cobra's
`BashCustomFunction` hook, checking the plugin cache the same way
`tryDispatchPlugin` does) would let `shy man <TAB>` "just work" for
every plugin, not only this one — and would let per-command help text
be sourced from a manifest field, closing both gaps above at once.
That is a `internal/cmd/completion.go` + `plugin_dispatch.go` change,
which `docs/04-agent-instructions.md`'s "Approval boundaries" section
marks as substantial (new subcommand / completion-convention change)
— out of scope for this plugin task; flagged here for the operator/
priest rather than built.

## Local checks

```bash
shellcheck man/man.sh man/_lib.sh tldr/tldr.sh tldr/_lib.sh who/who.sh who/_lib.sh completions/fleet-manual.bash
SHY_FLEET_AGENTS_DIR=~/.claude/agents ./man/man.sh governator
SHY_FLEET_AGENTS_DIR=~/.claude/agents ./tldr/tldr.sh governator
SHY_FLEET_AGENTS_DIR=~/.claude/agents ./who/who.sh "branch protection ruleset"
```

Full end-to-end (build the binary, install all three, dispatch through
it for real):

```bash
go build -o /tmp/shy ./cmd
export SHY_HOME=/tmp/shy-home-test
/tmp/shy init
/tmp/shy install ./examples/plugins/fleet-manual/man
/tmp/shy install ./examples/plugins/fleet-manual/tldr
/tmp/shy install ./examples/plugins/fleet-manual/who
/tmp/shy man governator
/tmp/shy tldr governator
/tmp/shy who "branch protection ruleset"
```

`SHY_FLEET_AGENTS_DIR` exists only for pointing tests at a fixture
directory; the plugin defaults to the real `~/.claude/agents` per
board #55 item 3 (own seat only).

## Conventions referenced

- `docs/01-whitepaper.md` — plugin model, manifest schema, completion
  conventions
- `docs/04-agent-instructions.md` — approval boundaries (`--json` /
  `--silent` plugin API; this read-only navigator needs neither)
- `examples/plugins/hello-world/` — reference layout this plugin
  mirrors (manifest.toml + entry script(s), `__complete` convention)
