# hello-world

The minimal reference plugin for shy. Demonstrates the plugin contract:

- A `manifest.toml` declaring `type = "plugin"` and `command =
  "hello-world"`.
- A single entry script (`hello-world.sh`) that handles the dispatch
  arguments.
- The `__complete` convention for plugin tab-completion — shy will
  invoke `hello-world.sh __complete <args-so-far>` to fetch
  candidates.

## Install

From a checkout of this repository:

```bash
shy install ./examples/plugins/hello-world
```

## Usage

```bash
$ shy hello-world
hello, world

$ shy hello-world friend
hello, friend
```

## Conventions referenced

- `docs/01-whitepaper.md` — plugin model, completion conventions
- `docs/04-agent-instructions.md` — `--json` / `--silent` for the
  plugin API (this trivial plugin does not need them)
