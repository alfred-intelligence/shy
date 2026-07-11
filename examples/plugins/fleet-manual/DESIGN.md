# fleet-manual — design draft

Status: implemented by `feat/fleet-manual-plugin` — forks resolved by the
operator 2026-07-11 (Fork A → A1 bare names; Fork B → B2 colleague-stab
only; Fork C → C1 local-only/own-seat). This document remains the
design record; the working plugin lives beside it as three sibling
packages — [`man/`](man/), [`tldr/`](tldr/), [`who/`](who/), each
mirroring [`hello-world`](../hello-world/)'s single-item layout — plus
shared [`_lib.sh`](_lib.sh) and [`completions/fleet-manual.bash`](completions/fleet-manual.bash).
See [`README.md`](README.md) for usage, the confirmed shy-core
multi-item-manifest dispatch bug that forced the three-package split
instead of this doc's §7 single-manifest sketch, and the `shy
help`/completion gaps filed as core follow-ups (not built here — plugin
only, zero shy-core edits per the task).

## 1. Problem

The operator now runs 60+ agent definitions under `~/.claude/agents/`
(colleague stab like `librarian`/`priest`/`warden` plus generic ECC
specialists like `code-reviewer`/`tdd-guide`). There is no way to browse
or search that roster except opening files by hand or holding it in
memory — which the operator has explicitly said doesn't scale ("too many
to track without it", per the [Shy fleet manual] memory note). The ask is
a `man`/`tldr`-style navigator: full charter, 5-line summary, and
keyword-based routing suggestion.

## 2. UX

Three read-only subcommands. All operate on local files only — no writes,
no shy state touched.

### `shy man <colleague>`

Full charter view — the agent's raw body, minus the boilerplate
Prompt Defense Baseline block (identical across all 60+ files, adds no
distinguishing signal, just noise on a terminal read). Rendered through
whatever pretty-printer is available (see §3.4), falling back to `cat`.

```
$ shy man librarian
librarian

Documentation steward for the alfred-intelligence org, and bearer of the
alfred persona — the operator's memory steward and counsel...

## Mission
- Author and maintain READMEs, docs/, codemaps, and contract documents...
[... full body, Prompt Defense Baseline section elided ...]
```

### `shy tldr <colleague>`

5-line compressed summary: purpose, trigger conditions, explicit "when
NOT" (most colleague descriptions already state this — "NOT a general
code/security reviewer (that's inquisitor)" style phrasing — extracted
via pattern match), and one example dispatch line.

```
$ shy tldr governator
governator — governance-conformance linter (call sign "Governator")
triggers: scheduled DECISIONS.md conformance sweeps, flag-triggered re-verification
NOT for: building/fixing CI or branch protection (that's priest); one-off code review (that's inquisitor)
dispatch: "run a governance sweep against alfred-intelligence/.github-private/DECISIONS.md"
confidence: description-derived, verify against ~/.claude/agents/governator.md if unsure
```

### `shy who <task-phrase>`

Grep-based routing suggestion, not a dispatcher. Matches the phrase
(word-split, case-insensitive) against each agent's `description` field
and ranks by hit count (all-words match ranked above any-word match).
Never dispatches — direktor/the operator still decides.

```
$ shy who "branch protection ruleset"
best match: priest       (3/3 words hit — "branch protection", "org rulesets")
also:       governator   (2/3 words hit — "branch protection", "ruleset alignment")
also:       inquisitor   (1/3 words hit — generic "review")
```

## 3. Data flow (pure bash)

```
~/.claude/agents/*.md
        │
        ▼
  [1] frontmatter extractor (awk)
        │  isolates the block between the first two `---` lines
        │  handles YAML folded scalars (`description: >-`) by
        │  concatenating continuation lines until the next
        │  top-level `key:` or the closing `---`
        ▼
  name=<value>  description=<value>
        │
        ▼
  [2] filter: drop hidden-by-default entries (frida — see §5)
        │
        ▼
  [3a] man   → cat body, sed-delete the "## Prompt Defense Baseline"
              …next "## " range, pipe through §3.4 renderer
  [3b] tldr  → description field, split on the "NOT ... (that's X)"
              pattern via a fixed sed/grep pattern already common to
              every colleague description; first sentence = purpose;
              "Use ... when" clause = triggers
  [3c] who   → grep -il each query word against description; tally
              hits per file; sort by hit-count desc
```

Illustrative frontmatter extraction (sketch, not final code):

```bash
awk '
  /^---$/ { fm++; next }
  fm==1 && /^[a-zA-Z_]+:/ {
    key=$0; sub(/:.*/, "", key)
    val=$0; sub(/^[a-zA-Z_]+:[[:space:]]*/, "", val)
    cur=key; if (cur=="description") { desc=val; next }
    if (cur=="name") { name=val }
    next
  }
  fm==1 && cur=="description" && /^[[:space:]]/ { desc = desc " " $0; next }
  fm==2 { exit }
  END { print name "\t" desc }
' "$file"
```

No `jq`, no `python`, no YAML library — a hand-rolled frontmatter reader
is sufficient because the schema is small and fixed (`name`,
`description`, `model`, `tools`, occasional `color`). This matches the
existing `manifest.toml` parsing philosophy in shy-core: small, explicit,
own-parser rather than pulling in a dependency for a schema this narrow.

### 3.1 No caching needed

All source files are local, small (a few KB each), and read fresh on
every invocation — 60-odd `Read`+`awk` calls is sub-100ms territory. No
`cache.json` entry, no index file, no staleness problem. Simpler than the
plugin-discovery cache shy already has for its own manifests, because
this data isn't shy's data — it's read-only and external.

### 3.2 No shy plugin-API touch

The plugin conventions in `docs/01-whitepaper.md` (§"Plugin API
conventions") govern access to shy's *own* installed-item state
(`cache.json`, `shy list --json`, etc.) — deliberately walled off so the
schema can evolve. `fleet-manual` never touches that state; it only reads
`~/.claude/agents/*.md`, which is entirely outside shy's domain. No API
surface needed beyond what any bash script already has.

### 3.3 Roster is single source of truth

Per the binding constraint: no second registry. `fleet-manual` never
caches a snapshot of the roster to disk, never hand-maintains a colleague
list — every invocation re-reads `~/.claude/agents/*.md` frontmatter
directly. If a colleague is added, renamed, or retired, the plugin's view
updates automatically on next run with zero maintenance.

### 3.4 Rendering fallback chain

`shy info` (core) uses the binary's embedded `glamour` renderer for
markdown. A plugin (bash) cannot call into that — the plugin API has no
"render arbitrary file" endpoint (see §3.2), and shy-core deliberately
avoids delegating to external renderers to keep the *binary* dependency-free
(whitepaper design table: "Glamour embedded ... consistent UX without
external dependency"). That discipline is a core-only constraint — plugins
are free to *soft-depend* on an optional pretty-printer via `[requires]`
and fall back gracefully:

```
1. glow (if on $PATH)   — best rendering
2. bat  (if on $PATH)   — syntax highlight, no markdown render
3. cat                  — always works, zero deps
```

Declared in the manifest under `[requires]` as optional, not required —
consistent with the `[[dependencies]]` `required`/`recommended`/`optional`
schema already defined in `01-whitepaper.md` §"Manifest format".

## 4. Plugin vs. core — recommendation: **plugin**

Run the whitepaper's own three-question test (`docs/04-agent-instructions.md`
§"Plugins absorb feature pressure"):

1. **Does this belong in core, or as a plugin?** — It reads a directory
   entirely outside shy's own domain (`~/.claude/agents/`, not
   `$SHY_HOME`). shy-core's job is bash snippets/aliases/completions/
   plugins distributed via git; a Claude-agent-roster browser is a
   different product surface that happens to share the operator's
   terminal.
2. **Is the plugin mechanism sufficient?** — Yes. Plugins get a
   dispatched subcommand, `__complete` tab-completion (useful here —
   completing colleague names), and full filesystem read access
   ("inheriting the operator's environment" per the plugin model). Nothing
   about `man`/`tldr`/`who` needs shy's internal state.
3. **What would make it native instead?** — Per the stated criterion,
   only "interacts with the binary's internal state in a way plugins
   cannot." This doesn't: no manifest writes, no `cache.json` reads, no
   installed-item awareness required.

This also matches `CONTRIBUTING.md`'s explicit bias: *"Anything that can
live as a plugin probably should — the binary's surface area is
deliberately small."*

**Correction to the task's framing:** the brief describes this as a
"shy plugin **or** pure-bash core addition." Worth flagging — shy-core
itself is not bash. Per `docs/01-whitepaper.md` (`## Dependencies`,
`## Design decisions and rationale`) and `docs/04-agent-instructions.md`,
the shy binary is Go (cobra + embedded glamour, packaged via GoReleaser);
only `init.bash` and `install.sh` are bash, and neither is where new
subcommands live. "Core addition" in this repo means a Go change to
`cli/`, not a bash script. The "bash+git, zero deps, jq-as-plugin"
discipline referenced in the task brief is real (see design decisions
table: dependencies are `git`, `curl`/`wget`, `sha256sum`, `tar`, bash for
snippets) but it describes the **plugin-script runtime baseline** — what a
*plugin's entry script* can assume without declaring `[requires]` — not
the implementation language of shy's own binary. Recommend the operator's
working shorthand for this discipline get tightened to: *"shy-core is Go;
plugin scripts default to bash+git baseline; anything beyond that is a
declared `[requires]`."* Filed here rather than silently corrected,
since it names a chosen framing and belongs in `DECISIONS.md`/memory, not
a silent edit by librarian.

Given the plugin recommendation, `fleet-manual` is pure bash regardless
(all plugin entry scripts are bash, per the `hello-world` reference and
the whitepaper's plugin model) — so the task's "no jq/python in core"
constraint is honored either way: the plugin entry script uses only
`awk`/`sed`/`grep`, no `jq`, no `python`.

## 5. Hidden-colleague filtering (binding, not a fork)

Per `dante-ops/CLAUDE.md`'s roster table and the standing memory
"Ignorera stash tills det nämns": `frida` is invisible-by-default. `man`,
`tldr`, and `who` MUST exclude `frida` from listings, tldr summaries, and
`who` search results unless the operator's query phrase names her or
Stash explicitly. This is not a design fork — it is an existing binding
rule this plugin has to inherit, most simply by hard-excluding her from
the default roster walk (a one-line filename skip) and only including her
if the query string literally contains `frida` or `stash`.

## 6. Open design forks for the operator

Three genuine ones — everything else above is either resolved by
shy's existing conventions or not worth a decision.

**Fork A — command-namespace collision risk.**
`shy man`/`shy tldr`/`shy who` claim bare top-level command names in
shy's flat plugin-dispatch namespace (native subcommands are checked
first, but any future *native* shy command named `man`/`tldr`/`who` would
be permanently blocked while this plugin is installed, and vice versa —
whoever registers a command first wins the slot). Options:
- **A1 — bare names** (`shy man`, `shy tldr`, `shy who`) — matches the
  familiar UNIX `man`/`tldr` idiom the task asked for; readable; but
  `man` in particular risks confusion with `man shy` (the real, packaged
  OS man page for the shy binary itself, generated via `cobra.GenManTree()`
  — a completely different thing living in `/usr/share/man/man1/`).
- **A2 — namespaced** (`shy fleet man`, `shy fleet tldr`, `shy fleet who`,
  via a single `fleet` dispatch command with subcommand routing inside the
  entry script) — avoids the collision and the `man shy` confusion
  entirely, costs one extra word per invocation.
- **Tipping factor:** how much the operator minds the `man shy` (OS man
  page) vs. `shy man <colleague>` (plugin) naming echo. Not resolvable
  from the repo — this is a personal-workflow call.

**Fork B — roster scope: colleague stab only, or the full 60+ agent set?**
- **B1 — full roster** — every file under `~/.claude/agents/*.md`,
  colleagues and generic ECC specialists (`a11y-architect`,
  `code-reviewer`, `tdd-guide`, etc.) alike. Matches the task's stated
  source of truth literally. `shy who` becomes more useful as a search
  tool since it covers everything dispatchable.
- **B2 — colleague stab only** — the ~14 human-named colleagues from
  `dante-ops/CLAUDE.md`'s roster table (`librarian`, `priest`, `warden`,
  `wizard`, `inquisitor`, `physician`, `governator`, `lagman`, `frida`,
  etc.), excluding the generic ECC specialist agents that are mostly
  invoked *by* other agents rather than dispatched directly by the
  operator. Shorter, more curated `man`/`tldr` output; `shy who` misses
  matches against specialists the operator might occasionally want to
  reach directly (e.g. `security-reviewer`).
- **Tipping factor:** whether the operator ever dispatches the generic
  ECC specialists directly, or only via colleague delegation. No second
  registry is invented either way — B2 just means a filter list (by name
  or by a lightweight marker convention) rather than a full directory
  walk; the "no second registry" constraint is honored in both, since a
  filter list of *names* isn't a competing roster of *facts* (description,
  purpose) — it just says which names to show.

**Fork C — single-host or fleet-wide roster?**
`~/.claude/agents/` is per-identity (per the fleet identity map memory:
`gg`, `ninja`, and `redhat` each have their own curated `.claude/` —
ninja's is explicitly "lean" and does not mirror gg's full set). Running
`fleet-manual` as `gg` shows a different roster than running it as
`ninja`.
- **C1 — local-only** (read whatever `~/.claude/agents/` resolves to for
  the invoking identity) — trivial, zero extra dependency, but "the fleet"
  as the operator experiences it (colleagues dispatched from any host/
  identity) isn't fully visible from one seat.
- **C2 — fleet-aggregate** (read across known identities/hosts, e.g. via
  the already-existing `sudo -u ninja` path or a kebab-it-style remote
  read) — matches "too many to track" as a fleet-wide problem, not a
  single-seat one, but adds real complexity (cross-identity/host reads,
  staleness, permissions) to what is otherwise a trivial plugin.
- **Tipping factor:** whether the operator's actual pain point is "I
  forget colleagues that exist on *this* seat" (C1 solves it today) or
  "I forget colleagues that exist on *other* seats/hosts" (needs C2,
  probably as a v1.x follow-on plugin rather than blocking v1 of
  `fleet-manual`).

## 7. Suggested manifest shape (once a fork decision lands)

```toml
name = "fleet-manual"
version = "0.1.0"
description = "man/tldr/who navigator for the Claude agent fleet roster."
license = "MPL-2.0"
type = "plugin"
command = "man"            # or "fleet", pending Fork A
entry = "./fleet-manual.sh"

[source]
repo = "alfred-intelligence/shy/examples/plugins/fleet-manual"

[requires]
bash = ">=4"

[[dependencies]]
name = "glow"
required = false
recommended = true

[capabilities]
binaries = ["awk", "sed", "grep", "glow", "bat"]
network = []
filesystem = ["~/.claude/agents"]
```

`[capabilities]` is v1-reserved/documentation-only per the whitepaper,
but worth declaring accurately from day one so a future `shy audit`
run has correct ground truth to check against.
