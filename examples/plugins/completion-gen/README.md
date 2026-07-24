# completion-gen

Generate shell completions for an **arbitrary binary** by introspecting
what it supports — no framework assumed. Tries the tool's own completion
generator first (a wider net than shy's built-in `completion add`), falls
back to parsing `--help`/`-h` text when the tool has none.

Built as a candidate replacement for shy's own `shy completion add`
(`internal/cmd/completion.go`) — see "Relationship to `shy completion
add`" and "Core-graduation path" below. It is a **plugin today**, not a
core change: `docs/04-agent-instructions.md`'s "plugins absorb feature
pressure" discipline says exactly that — prove it as a plugin before it
earns a place in the binary.

## Usage

```bash
shy completion-gen <tool> [options]
shy completion-gen --list-methods
```

| Option | Default | Meaning |
|---|---|---|
| `--shell bash\|zsh\|fish` | `bash` | Target shell |
| `--install` | off | Write to `$SHY_HOME/helpers/completions/<tool>` instead of stdout |
| `--force` | off | With `--install`: overwrite regardless of conflict policy |
| `--depth N` | `2` | Max subcommand recursion depth |
| `--timeout SECS` | `3` | Per-probe timeout |
| `--max-calls N` | `40` | Max tool invocations during introspection (bounds runtime on huge trees — docker, gh, aws, kubectl, …) |
| `--json` | off | Print diagnostics instead of the script |
| `-h, --help` | | Usage text |

Without `--install`, the script prints to stdout. That is deliberate: it
is exactly the shape shy's own manifest hook already expects
(`internal/install/install.go`'s `[[completions]] generate = "..."`), so
`generate = "shy completion-gen <tool>"` slots into an **existing**
manifest today, with **no shy-core change** — verified end to end (see
"Verified integration" below), not just asserted.

```bash
shy install ./examples/plugins/completion-gen
shy completion-gen gh              # native cobra passthrough, printed to stdout
shy completion-gen my-legacy-tool --install --force
```

## How it decides (two tiers)

**Tier 1 — native passthrough.** If the tool can generate its own
completions, that is always more accurate than a text-scraped
approximation, so it is tried first, for every requested `--shell`:

```
shy completion-gen --list-methods
```

```
1. <tool> completion <shell>
2. <tool> completion --shell <shell>
3. <tool> completion -s <shell>
4. <tool> completions <shell>
5. <tool> --completion <shell>
6. <tool> bash-completion                       (bash only; legacy)
7. _<TOOL>_COMPLETE=<shell>_source <tool>        (click convention)
8. register-python-argcomplete [--shell <shell>] <tool>  (if on PATH)
```

shy's own `completion add` tries only #1/#2 (hardcoded to bash) and #6.
This is a strict superset at tier 1, before introspection even starts —
already a direct upgrade with zero parsing risk.

A candidate is accepted only if it exits `0`, is non-trivial (≥20 bytes),
and is **not byte-identical to the tool's own `--help`/`-h` text** — the
tell that a probe just triggered a generic help dump because the tool
didn't recognise the arguments, not a real completion script.

Probe #8 additionally requires the target to actually look like a Python
entry point (shebang check) before it runs — `register-python-argcomplete
<name>` was verified to exit `0` and print generic, non-functional
boilerplate for **any** name at all, including compiled binaries with no
relation to Python (`git` included). Trusting its mere success would have
been a guaranteed false positive for most non-Python tools.

**Tier 2 — `--help`/`-h` text introspection.** Only runs if tier 1 found
nothing. Captures the tool's `--help`, falling back to `-h`, and:

- extracts **subcommands** from a header line ending in
  "commands"/"subcommands" (optionally with a leading qualifier like
  "Available"/"Management", and/or a trailing parenthetical like
  kubectl's `"Basic Commands (Beginner):"`), then reads indented lines
  until a blank line or a new section;
- extracts **flags** by scanning the *whole* text for `-x`/`--long-name`
  tokens in flag position — no header required, which is what makes this
  work for plain POSIX getopt-style tools that never label an "Options:"
  section at all (verified against `curl --help`: 28 flags, correctly 0
  subcommands);
- **recurses** into each discovered subcommand (`<tool> <sub> --help`),
  up to `--depth` and `--max-calls`, building a tree.

A safety guard that mattered in practice: if a subcommand's `--help`
output is byte-identical to the **root's**, it is treated as "no
distinct help available", not real content — many real CLIs just
reprint root usage for an unrecognised subcommand instead of erroring,
and without this guard the walk recurses into that reprinted text as if
it had genuinely distinct (and identically-named) children, producing an
exponential, meaningless tree. Caught with a synthetic fixture: 3 real
subcommands blew up to 33 bogus nodes at depth 3 before this guard;
3 correct nodes after.

## Safety envelope

The target tool is **only ever** invoked two ways:

1. With an explicit `--help`/`-h` flag (never bare — a bare invocation of
   an arbitrary binary can have real side effects: open an editor,
   attempt a network connection, wait on a prompt; an explicit
   `--help`/`-h` request is the one thing every CLI convention treats as
   read-only by contract), or
2. Via one of the eight documented completion-generator conventions above
   (the same risk class shy's own `completion add` already accepts today).

Every invocation runs under `timeout`/`gtimeout` (falls back to no limit
with a logged warning if neither is on `PATH` — flagged, not silent) with
stdin closed (`</dev/null`), so a probe can never hang the whole run
waiting on input.

Probes deliberately force `LC_ALL=C LANGUAGE=en` when capturing help
text for parsing: the tier-2 regexes are English-pattern-based (as is
virtually every practical `--help` scraper), so a localised `--help`
(verified against git's own Swedish translation on this host) would
otherwise silently yield zero subcommands/flags from a tool that
actually has plenty.

## Shell coverage — fidelity is uneven, and that's documented, not hidden

- **bash** gets a full, context-aware `COMPREPLY` function that walks the
  whole parsed tree (shy is bash-first per `docs/01-whitepaper.md`).
- **zsh** gets a `bashcompinit` compatibility shim that reuses the exact
  same bash function — correct by construction, not a second parser to
  maintain and get subtly wrong.
- **fish** gets native `complete` directives, but only for depth 0-1:
  fish's data-driven completion primitives don't model exact positional
  nesting past one subcommand level without a custom completion function
  per node, which is out of scope for v0.1. Depth ≥2 nodes are still
  parsed (visible in `--json`) but not emitted for fish; use `--shell
  bash` for the full-depth tree.

## Known limitations (found by testing against real tools, not guessed)

- **Free-text false positives are possible and accepted as the lesser
  evil.** git's own `--help` mentions `-a`/`-g` in prose ("`git help -a`
  and `git help -g` list available subcommands…") and both get picked up
  as flag candidates, because the flag scanner has no header to anchor
  on. False positives here are harmless (an extra, wrong completion
  candidate); false negatives (a real flag never offered) are worse — the
  tradeoff is deliberate.
- **Unconventional help formats degrade gracefully but leave gaps.**
  git's own top-level `--help` uses grouped free-text section blurbs
  ("start a working area (see also: …)") with no literal
  "Commands:"-style header at all, so `git` currently yields 0
  subcommands and only its top-level flags. This is an honest,
  documented gap for one specific well-known tool with genuinely
  unusual conventions, not a silent wrong answer — `--json` shows exactly
  what was and wasn't found.
- **fish depth cap** — see "Shell coverage" above.

## Relationship to `shy completion add`

| | `shy completion add <tool>` (core, today) | `completion-gen` (this plugin) |
|---|---|---|
| Native conventions tried | 3 (`completion bash`, `completion --shell bash`, `bash-completion`) | 8, per-shell |
| Fallback when none work | none — hard failure | `--help`/`-h` introspection: subcommand tree + flags |
| Shells | bash only | bash (full), zsh (shim), fish (depth-limited) |
| Conflict policy | `SHY_ON_CONFLICT` (fail/prefer-new/prefer-existing/skip) | same four values, same default — deliberately mirrored |
| Diagnostics | none | `--json`: method used, node/sub/flag counts, warnings |

## Verified integration (not just designed — actually run)

- `shy install ./examples/plugins/completion-gen` against a real build of
  the `shy` binary: installs correctly under `installed/@<ns>/completion-gen/`,
  shows up in `shy list`, dispatches via `shy completion-gen <tool>` exactly
  like any other plugin command.
- A **separate** test manifest with:
  ```toml
  [[completions]]
  tool = "some-tool"
  generate = "shy completion-gen some-tool"
  ```
  installed through shy's **own, unmodified** `install.go` — core called
  `bash -lc "shy completion-gen some-tool"` through its existing generic
  `generate` hook and wrote the result to
  `$SHY_HOME/helpers/completions/some-tool`, with zero changes to any
  shy-core file. This is the concrete proof for the "no core change
  required today" claim above, not an assumption.
- Native tier-1 passthrough verified against a real `kubectl` (cobra):
  correctly detected via `kubectl completion bash` and returned kubectl's
  own real completion script unmodified.
- Tier-2 introspection verified against a synthetic multi-level fixture
  (2 command sections including a kubectl-style qualified header, a
  comma-aliased subcommand line, a 2-level `deploy → prod` subtree) — the
  **generated bash completion function** was then sourced and exercised
  with simulated `COMP_WORDS`/`COMP_CWORD` at the root, at one level deep,
  at two levels deep with a partial flag, and with a partial subcommand
  prefix; all five returned the correct candidate set.

## Core-graduation path

This started as, and remains, a plugin — proving the mechanism before it
earns core status, per `docs/04-agent-instructions.md`'s "plugins absorb
feature pressure" discipline. To actually replace
`internal/cmd/completion.go`'s `shy completion add`, in order:

1. **Operator sign-off** — `docs/04-agent-instructions.md` "Approval
   boundaries" lists "any new subcommand or change to existing subcommand
   semantics" as requiring explicit approval before code. Replacing
   `completion add`'s behavior is exactly that.
2. **Field usage** — run as a plugin across the fleet's actual toolset for
   a real stretch of time; the `--json` diagnostics exist specifically so
   that period produces evidence (method-used distribution, how often
   tier 2 fires, how often its output needed hand-correction) instead of
   a subjective "seems fine."
3. **Genuine port to Go**, not a shell-out — a promoted `completion add`
   should not `exec` a shell script from the compiled binary. The tier-1
   probe list and tier-2 regex heuristics documented here are the
   **spec** for that port; this plugin is the reference implementation to
   port from, not a wrapper to keep calling forever.
4. **Decide the fallback boundary** — should core's `completion add`
   itself fall back to introspection, or should introspection remain an
   explicit opt-in (`shy completion add --introspect`, or similar) so a
   silent native-probe failure doesn't quietly hand the operator a
   best-effort script instead of a hard error? That is a product decision
   for the operator, not something this plugin should presume.
5. **Decide shell scope for v1 core** — port bash first (matches shy's
   own bash-first identity); zsh/fish parity is a v1.x-or-later call.

## Local testing

```bash
go build -o /tmp/shy ./cmd/
SHY_HOME=/tmp/shy-home /tmp/shy install ./examples/plugins/completion-gen
SHY_HOME=/tmp/shy-home /tmp/shy completion-gen gh --json
```

## Conventions referenced

- `docs/01-whitepaper.md` — plugin model, completion conventions, `--json`/`--silent` API
- `docs/02-long-horizon.md` Phase 10 — ecosystem plugins (`shy-auto-completions` is a
  scanning/scheduling layer that could later sit on top of this plugin's
  per-tool generation primitive; this plugin does not implement scanning)
- `docs/04-agent-instructions.md` — plugin API conventions, approval boundaries, "plugins absorb feature pressure"
