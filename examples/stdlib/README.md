# shy-stdlib

A small set of broadly useful shell snippets curated as the first
collection for shy.

This lives under `examples/` in the shy repository while v0.x is
private. At v1.0 it will be extracted to its own repo at
`github.com/alfred-intelligence/shy-stdlib` so it can release on
its own cadence.

## Items

| Name        | Type   | Purpose                                                  |
|-------------|--------|----------------------------------------------------------|
| `mkcd`      | script | `mkdir -p` and `cd` in one step                          |
| `extract`   | script | Extract any archive by file extension                    |
| `serve`     | script | Start an HTTP server in the current directory            |
| `path-list` | script | Print `$PATH` one entry per line                         |
| `up`        | script | `cd ..` N times                                          |
| `gst`       | alias  | `git status -sb`                                         |
| `ll`        | alias  | `ls -alh --color=auto`                                   |
| `la`        | alias  | `ls -A --color=auto`                                     |

## Install

While shy-stdlib lives in this repo:

```bash
shy install ./examples/stdlib
```

Once extracted to its own repository:

```bash
shy collection subscribe github:alfred-intelligence/shy-stdlib
```

## Contributing a new snippet

1. Drop the file under `scripts/<name>.sh`.
2. Add a `[[items]]` entry to `manifest.toml` with a one-line
   description.
3. PR against `next`.

Stdlib snippets should be broadly useful — every operator should
want them, not just the author. Anything more specialised belongs
in its own collection.
