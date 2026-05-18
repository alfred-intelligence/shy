# shy

> Small Shell Utility — a CLI for managing bash snippets, aliases, and
> completions across machines via git-distributed collections.

`shy` keeps your `.bashrc` to a single sourcing line. Everything else —
snippets, aliases, completions, plugins — lives under `$HOME/.shy/`
and is sourced from there at shell start.

**Status: v0.x — pre-release development.** API and CLI surface may
change before v1.0. The repository will be made public at the v1.0
release; see `docs/02-long-horizon.md` for the roadmap.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/alfred-intelligence/shy/main/install.sh | bash
```

The installer lands the binary at `$HOME/.shy/bin/shy` and runs
`shy init`, which sets up `$HOME/.shy/` and adds one source line to
`~/.bashrc`.

System-wide installation via `.deb`/`.rpm` packages is available from
the [GitHub Releases](https://github.com/alfred-intelligence/shy/releases)
page.

## Usage

```bash
shy alias 'll=ls -alh'                # add an alias
shy completion add gh                 # add a completion
shy install @user/some-snippet        # install a published snippet
shy collection subscribe github:user/setup
                                      # subscribe to a collection
shy list --sources                    # see what's installed and from where
shy info @user/some-snippet           # render README for an item
shy create my-script && shy publish my-script
                                      # author and publish a new script
```

Full command reference: `shy --help` or `man shy`.

## Design

The product vision, architecture, manifest schema, security model and
roadmap live in [`docs/`](docs/). Start with
[`docs/00-index.md`](docs/00-index.md).

## License

[MPL-2.0](LICENSE).
