# shy — Short Horizon (Next Phase)

Detailed plan covering Phase 1 (skeleton) and Phase 2 (core
commands). Written as numbered steps an operator or AI agent can
follow without needing to read anything else.

**Note:** This document may be revised at the start of Phase B
(operationalisation) once the repository exists and the operational
assumptions from `01-whitepaper.md` (Assumptions for Phase B) have
been confirmed by the operator. Items marked with `[B-CONFIRM]`
explicitly depend on those decisions.

## Critical path

**Step 1 → Step 3 → Step 6 → Step 9 → Step 11** cannot be
parallelised. Steps 2, 4, 5, 7, 8, 10 can happen in any order once
their dependencies are met.

---

## Pre-flight checks (run before Step 1)

The operator indicated that a pre-release may already exist on the
`next` branch — mentioned as `v0.2-pre` or `v0.9-pre` in different
statements with some uncertainty. Verify the actual repository
state before proceeding.

**Checks:**

1. Clone the repository locally if not already:
   `git clone https://github.com/alfred-intelligence/shy.git`
2. Fetch tags: `git fetch --tags`
3. List existing tags: `git tag -l 'v*'`
4. Check the `next` branch exists and has commits:
   `git log next --oneline | head -5`
5. Check GitHub Releases page for any release marked as
   pre-release.

**Decision tree:**

- **If a pre-release tag (`v0.2*` or `v0.9*`) exists** on `next`:
  proceed to Step 1 noting the pre-existing release as the starting
  point. release-please will produce the next version from that
  baseline.
- **If no pre-release tag exists despite the operator's claim:**
  do not proceed with Step 1. Report this finding to the operator:
  "No pre-release `v0.2*` or `v0.9*` was found on `next`. Please
  verify the intended release state before implementation begins."
  The operator handles any follow-up from there; the implementer's
  responsibility ends with the report.
- **If `next` branch does not exist yet:** the repository was
  initialised but no branching has been done. Proceed to Step 1,
  which will create `next`.

This pre-flight check exists because the operator's recollection
of the release state was uncertain. Verifying first prevents
implementer rework if the assumed baseline is wrong.

---

## Step 1 — Create repository

**Task:** Create `github.com/alfred-intelligence/shy` as a private
repository (made public at v1.0 per the Phase B assumptions in
`01-whitepaper.md`) with the minimum scaffolding to start committing.

**Commands:**

```bash
gh repo create alfred-intelligence/shy --private \
    --description "Small Shell Utility — bash snippet, alias, completion manager"

git clone git@github.com:alfred-intelligence/shy.git
cd shy

# Initial files
echo "# shy" > README.md
curl -fsSL https://www.mozilla.org/media/MPL/2.0/index.txt -o LICENSE  # MPL-2.0
printf '/dist/\n*.test\n*.out\ncoverage.out\n' > .gitignore

git add -A
git commit -m "chore: initial scaffold"
git push origin main
```

**Done when:**

- Repository visible at `https://github.com/alfred-intelligence/shy`
- README and `.gitignore` committed on `main`

---

## Step 2 — Initialise Go module

**Task:** Create the Go module structure and stub `main.go`.

**Commands:**

```bash
go mod init github.com/alfred-intelligence/shy

mkdir -p cmd internal
cat > cmd/main.go << 'EOF'
package main

import "fmt"

var version = "0.1.0-draft"

func main() {
    fmt.Printf("shy %s\n", version)
}
EOF

go build -o dist/shy ./cmd
./dist/shy   # should print "shy 0.1.0-draft"

git add cmd go.mod
git commit -m "Add Go module and stub binary"
```

**Done when:**

- `go build` succeeds
- `./dist/shy` prints the version string

---

## Step 3 — Write `init.bash` template

**Task:** Create the shell template that `shy init` will install
into `$HOME/.shy/`. Sources `entry.sh` from every script directory
under `installed/%<ns>/<name>/`, and flat files from `helpers/aliases/`
and `helpers/completions/`. Overrides sourced last from the matching
`overrides.d/` paths.

**Commands:**

```bash
mkdir -p init
cat > init/init.bash << 'EOF'
# ~/.shy/init.bash
# Sourced from ~/.bashrc to activate installed shy items.

export SHY_HOME="${SHY_HOME:-$HOME/.shy}"

# Source files in a flat directory; skip _-prefixed; tolerate errors.
_shy_source_flat() {
    local dir="$1"
    [[ -d "$dir" ]] || return 0
    local f
    for f in "$dir"/*; do
        [[ -f "$f" ]] || continue
        [[ "$(basename "$f")" == _* ]] && continue
        # shellcheck source=/dev/null
        source "$f" 2>/dev/null || printf 'shy: failed to source %s\n' "$f" >&2
    done
}

# Source files in a namespaced directory tree (namespace/name/*.sh).
_shy_source_namespaced() {
    local dir="$1"
    [[ -d "$dir" ]] || return 0
    local ns item sh
    for ns in "$dir"/*/; do
        [[ -d "$ns" ]] || continue
        for item in "$ns"*/; do
            [[ -d "$item" ]] || continue
            for sh in "$item"*.sh; do
                [[ -f "$sh" ]] || continue
                [[ "$(basename "$sh")" == _* ]] && continue
                # shellcheck source=/dev/null
                source "$sh" 2>/dev/null || printf 'shy: failed to source %s\n' "$sh" >&2
            done
        done
    done
}

# User layer — primary source of truth.
_shy_source_namespaced "$SHY_HOME/scripts"
_shy_source_flat "$SHY_HOME/aliases"
_shy_source_flat "$SHY_HOME/completions"

# Overrides — re-define items from the user layer.
_shy_source_namespaced "$SHY_HOME/overrides.d/scripts"
_shy_source_flat "$SHY_HOME/overrides.d/aliases"
_shy_source_flat "$SHY_HOME/overrides.d/completions"

unset -f _shy_source_flat _shy_source_namespaced
EOF

shellcheck init/init.bash

git add init/
git commit -m "Add init.bash template with namespaced walking and _-prefix skip"
```

**Done when:**

- `init/init.bash` exists and passes `shellcheck`
- Sourcing it manually in a test shell with an empty `~/.shy/`
  produces no errors

---

## Step 4 — Write `install.sh`

**Task:** Create the `curl | bash` entry point. Detects OS/arch,
fetches the matching binary from GitHub Releases, verifies SHA256,
unpacks into `$HOME/.shy/bin/`.

**Commands:**

```bash
cat > install.sh << 'EOF'
#!/usr/bin/env bash
# install.sh — curl|bash entry point for shy. Stays backward compatible forever.
set -euo pipefail

VERSION="${SHY_VERSION:-latest}"
PREFIX="${SHY_HOME:-$HOME/.shy}"
REPO="alfred-intelligence/shy"

# Detect OS and architecture.
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
    x86_64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "shy: unsupported arch: $arch" >&2; exit 1 ;;
esac

# Resolve "latest" to a concrete tag.
if [[ "$VERSION" == "latest" ]]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
              | grep '"tag_name"' | head -1 | cut -d'"' -f4)
fi
[[ -n "$VERSION" ]] || { echo "shy: could not resolve version" >&2; exit 1; }

# Fetch and verify.
asset="shy_${VERSION#v}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$asset"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$url" -o "$tmp/$asset"
curl -fsSL "$url.sha256" -o "$tmp/$asset.sha256"
( cd "$tmp" && sha256sum -c "$asset.sha256" >/dev/null )

# Install.
mkdir -p "$PREFIX/bin"
tar -xzf "$tmp/$asset" -C "$PREFIX/bin" shy
chmod +x "$PREFIX/bin/shy"

# Hand off to the binary for further initialisation.
"$PREFIX/bin/shy" init

echo "shy $VERSION installed at $PREFIX/bin/shy"
EOF
chmod +x install.sh

shellcheck install.sh

git add install.sh
git commit -m "Add install.sh entry point"
```

**Done when:**

- `install.sh` exists and passes `shellcheck`
- Logic reviewed against the contract in `01-whitepaper.md`
  (immutable asset schema, SHA verification, idempotent install)

---

## Step 5 — Set up GoReleaser

**Task:** Add `.goreleaser.yaml` so future tags produce the correct
release artefacts that `install.sh` depends on. Include nfpms config
for `.deb`/`.rpm` packages and man-page packaging.

**Commands:**

```bash
cat > .goreleaser.yaml << 'EOF'
version: 2

before:
  hooks:
    - go mod tidy
    - go run ./cmd gen-man /tmp/shy-man   # generated by cobra

builds:
  - id: shy
    main: ./cmd
    binary: shy
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: shy
    name_template: >-
      shy_{{ .Version }}_{{ .Os }}_{{ if eq .Arch "amd64" }}amd64{{ else if eq .Arch "arm64" }}arm64{{ end }}
    format: tar.gz
    files: [LICENSE, README.md]

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256
  split: true   # emits per-asset .sha256 files

nfpms:
  - id: packages
    package_name: shy
    vendor: alfred-intelligence
    homepage: https://github.com/alfred-intelligence/shy
    maintainer: alfred-intelligence <alfred@gegge.se>
    description: Small Shell Utility — bash snippet, alias, completion manager
    license: MPL-2.0
    formats: [deb, rpm]
    bindir: /usr/bin
    contents:
      - src: /tmp/shy-man/
        dst: /usr/share/man/man1/
        type: tree

release:
  github:
    owner: alfred-intelligence
    name: shy

snapshot:
  name_template: "{{ incpatch .Version }}-next"
EOF

git add .goreleaser.yaml
git commit -m "Add GoReleaser configuration with packages and man-pages"
```

**Done when:**

- `goreleaser check` exits 0
- Asset name template confirmed to produce
  `shy_<version>_<os>_<arch>.tar.gz`

---

## Step 6 — CI workflow

**Task:** Add a GitHub Actions workflow that runs `go vet`,
`go test`, and `shellcheck` on every push and PR.

**Commands:**

```bash
mkdir -p .github/workflows
cat > .github/workflows/ci.yml << 'EOF'
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go vet ./...
      - run: go test ./...

  shell:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          sudo apt-get update
          sudo apt-get install -y shellcheck
      - run: shellcheck install.sh init/init.bash
EOF

git add .github/workflows/ci.yml
git commit -m "Add CI workflow for Go and shell checks"
git push origin main
```

**Done when:**

- CI workflow runs on the push and passes

---

## Step 7 — Cut draft release `v0.1.0-draft`

**Task:** Tag the current main as a draft release; verify
GoReleaser produces expected artefacts.

**Commands:**

```bash
git tag -a v0.1.0-draft -m "v0.1.0 — skeleton, not feature-complete"
git push origin v0.1.0-draft

# Locally simulate GoReleaser (requires goreleaser installed).
goreleaser release --snapshot --clean

ls dist/
# Expected: shy_0.1.0-draft_linux_amd64.tar.gz and similar, plus per-asset .sha256
```

**Done when:**

- Tag `v0.1.0-draft` pushed
- `goreleaser release --snapshot --clean` produces expected
  artefacts locally
- Asset names match the schema baked into `install.sh`

---

## Step 8 — Release workflow

**Task:** Add a GitHub Actions workflow that runs GoReleaser on tag
push.

**Commands:**

```bash
cat > .github/workflows/release.yml << 'EOF'
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
EOF

git add .github/workflows/release.yml
git commit -m "Add release workflow"
git push origin main

# Re-tag to trigger the new workflow:
git tag -d v0.1.0-draft
git push origin --delete v0.1.0-draft
git tag -a v0.1.0-draft -m "v0.1.0 — skeleton, not feature-complete"
git push origin v0.1.0-draft
```

**Done when:**

- Release workflow runs successfully on tag push
- GitHub Releases page shows `v0.1.0-draft` with archive, checksum,
  `.deb`, and `.rpm` assets

---

## Step 9 — End-to-end install on a clean VM

**Task:** Verify that `install.sh` fetches the draft release, places
the binary, and `shy --version` works.

**Commands (on a clean VM, e.g. Ubuntu 22.04):**

```bash
# Should pull the draft release and install it.
curl -fsSL https://raw.githubusercontent.com/alfred-intelligence/shy/main/install.sh | bash

# Verify.
~/.shy/bin/shy
# Expected: shy 0.1.0-draft

# Verify init.bash got placed.
test -f ~/.shy/init.bash && echo "init.bash present"

# Verify .bashrc got the source line.
grep 'shy/init.bash' ~/.bashrc && echo "bashrc integration in place"
```

**Done when:**

- `install.sh` succeeds without manual intervention
- `shy --version` prints the expected string
- `init.bash` exists in `$HOME/.shy/`
- `.bashrc` contains a single source line for `shy/init.bash`

---

## Step 10 — Manifest parser

**Task:** Implement the TOML manifest parser. Validate single-item,
multi-item, source, and dependency forms. Parser tolerates unknown
top-level sections (extensible by design — plugins can add their
own metadata sections that shy preserves and surfaces via the
JSON API).

**Commands:**

```bash
cd cli
go get github.com/pelletier/go-toml/v2

mkdir -p internal/manifest
cat > internal/manifest/manifest.go << 'EOF'
package manifest

import (
    "github.com/pelletier/go-toml/v2"
)

type Manifest struct {
    Name         string              `toml:"name"`
    Version      string              `toml:"version"`
    Description  string              `toml:"description,omitempty"`
    License      string              `toml:"license,omitempty"`
    Type         string              `toml:"type,omitempty"`
    Entry        string              `toml:"entry,omitempty"`
    Source       *Source             `toml:"source,omitempty"`
    Items        []Item              `toml:"items,omitempty"`
    Aliases      map[string]string   `toml:"aliases,omitempty"`
    Completions  []CompletionItem    `toml:"completions,omitempty"`
    Dependencies []Dependency        `toml:"dependencies,omitempty"`
    Requires     *Requires           `toml:"requires,omitempty"`
    Capabilities *Capabilities       `toml:"capabilities,omitempty"`
    Security     *Security           `toml:"security,omitempty"`
    Unknown      map[string]any      `toml:"-"`
}

type Source struct {
    Repo string `toml:"repo"`
    Ref  string `toml:"ref,omitempty"`
}

type Item struct {
    Name        string `toml:"name"`
    Type        string `toml:"type"`
    Path        string `toml:"path,omitempty"`
    Command     string `toml:"command,omitempty"`
    Value       string `toml:"value,omitempty"`
    Description string `toml:"description,omitempty"`
    Source      string `toml:"source,omitempty"`
    Ref         string `toml:"ref,omitempty"`
}

type CompletionItem struct {
    Tool     string `toml:"tool"`
    Generate string `toml:"generate"`
}

type Dependency struct {
    Source     string `toml:"source"`
    Constraint string `toml:"constraint,omitempty"`
    Type       string `toml:"type"`
}

type Requires struct {
    Bash     string   `toml:"bash,omitempty"`
    Binaries []string `toml:"binaries,omitempty"`
}

// Capabilities — declared, parsed, surfaced; not enforced in v1.
type Capabilities struct {
    Network    []string `toml:"network,omitempty"`
    Binaries   []string `toml:"binaries,omitempty"`
    Filesystem []string `toml:"filesystem,omitempty"`
}

// Security — marks an update as a security fix; bypasses throttle.
type Security struct {
    Fixes       string `toml:"fixes,omitempty"`
    Severity    string `toml:"severity,omitempty"`
    Description string `toml:"description,omitempty"`
}

// Parse extracts known fields into the typed struct and preserves
// the full raw document for plugin-defined sections.
func Parse(data []byte) (*Manifest, error) {
    var m Manifest
    if err := toml.Unmarshal(data, &m); err != nil {
        return nil, err
    }
    // Re-parse into a generic map and store unknown top-level keys.
    var raw map[string]any
    if err := toml.Unmarshal(data, &raw); err != nil {
        return nil, err
    }
    known := map[string]struct{}{
        "name": {}, "version": {}, "description": {}, "license": {},
        "type": {}, "entry": {}, "source": {}, "items": {},
        "aliases": {}, "completions": {}, "dependencies": {},
        "requires": {}, "capabilities": {}, "security": {},
    }
    m.Unknown = make(map[string]any)
    for k, v := range raw {
        if _, ok := known[k]; !ok {
            m.Unknown[k] = v
        }
    }
    return &m, nil
}
EOF

# Smoke tests
cat > internal/manifest/manifest_test.go << 'EOF'
package manifest

import "testing"

func TestParseSingleItem(t *testing.T) {
    data := []byte(`
name = "git-autofetch"
version = "1.0.0"
type = "script"
entry = "./git-autofetch.sh"

[source]
repo = "alice/git-autofetch"
`)
    m, err := Parse(data)
    if err != nil { t.Fatal(err) }
    if m.Name != "git-autofetch" { t.Errorf("name: %s", m.Name) }
    if m.Type != "script" { t.Errorf("type: %s", m.Type) }
    if m.Source == nil || m.Source.Repo != "alice/git-autofetch" {
        t.Errorf("source not parsed: %+v", m.Source)
    }
}

func TestParseMultiItem(t *testing.T) {
    data := []byte(`
name = "collection"
version = "2.0.0"

[[items]]
name = "foo"
type = "script"
path = "./scripts/foo.sh"

[aliases]
ll = "ls -alh"
`)
    m, err := Parse(data)
    if err != nil { t.Fatal(err) }
    if len(m.Items) != 1 { t.Errorf("items: %d", len(m.Items)) }
    if m.Aliases["ll"] != "ls -alh" { t.Errorf("alias ll: %s", m.Aliases["ll"]) }
}
EOF

go test ./internal/manifest/

cd ..
git add cmd internal/
git commit -m "Add TOML manifest parser with basic tests"
```

**Done when:**

- `go test ./internal/manifest/` passes
- Single-item, multi-item, and source forms parse correctly

---

## Step 11 — Implement `shy init`

**Task:** Wire up `init` as a real subcommand. Creates directory
structure with namespacing, writes `init.bash`, modifies `.bashrc`
once, auto-installs shy's own completion via its own mechanism.

**Behavioural specification:**

- If running as root: print hint about `shy system-install` and exit
- Create `$HOME/.shy/` with `chmod 700` (protects against other
  users on shared hosts)
- Create subdirectories:
  `installed/`, `helpers/aliases/`, `helpers/completions/`,
  `overrides.d/installed/`, `overrides.d/helpers/aliases/`,
  `overrides.d/helpers/completions/` — subdirectories
  inherit standard permissions from umask
- Write `init.bash` if missing
- Add `[ -f "$HOME/.shy/init.bash" ] && source "$HOME/.shy/init.bash"`
  to `~/.bashrc` if not already present (interactive prompt unless
  `--no-bashrc` is passed; idempotent)
- Generate `shy completion bash` output and write it to
  `$HOME/.shy/helpers/completions/shy` (shy bootstrapping its own
  completion via its own mechanism — a sanity check that the
  mechanism works end-to-end)
- Print summary: directories created, files written, bashrc modified
  yes/no

**Implementation:** Use `cobra` for command structure; implement
the steps in `internal/cmd/init.go`. Helper functions in
`internal/cmd/init_helpers.go` handle copy-tree-skip-conflict,
bashrc-line-detection, and completion-bootstrap.

**Done when:**

- `shy init` creates the directory layout under `~/.shy/` with the
  namespaced sub-structure
- `init.bash` is written if missing
- `.bashrc` gets one source line; running `shy init` again does not
  duplicate it
- `~/.shy/helpers/completions/shy` contains shy's own bash-completion
- `sudo shy init` refuses with the hint message pointing to
  `shy system-install`
- Integration test: source the new init.bash in a fresh bash; no
  errors, even with empty directories; verify `shy <tab>` produces
  completions

---

## Step 12 — Implement `shy install` (local path only)

**Task:** Implement `shy install <path>` for local manifest+files
bundles. Defer URL and `@user/repo` syntax to Step 15.

**Behavioural specification:**

- Parse manifest from `<path>/manifest.toml`
- Determine installation namespace:
  - If `[source].repo` exists in manifest, extract author part
    (e.g., `alice` from `alice/git-autofetch`) → use as
    namespace
  - Otherwise, derive safe-name from `$HOSTNAME` (lowercase,
    `a-z0-9-` only, `.local` suffix stripped) → use as namespace
- For each item:
  - Validate type-specific fields (plugin needs command, alias
    needs value, etc.)
  - For scripts: create
    `$SHY_HOME/installed/%<namespace>/<name>/`, write `entry.sh`
    as the canonical entry point, and copy `manifest.toml` and optional
    `README.md`
  - For plugins: same pattern under `installed/@<namespace>/<name>/`
  - For aliases: write `$SHY_HOME/helpers/aliases/<name>` with `alias`
    line; if file exists with different content, trigger conflict
    flow
  - For completions: write `$SHY_HOME/helpers/completions/<tool>` with
    the captured output of `<generate>` command; same conflict
    semantics
- Update `cache.json` with installation record (namespace, name,
  source, version, pinned commit/ref)

**Implementation:** `internal/cmd/install.go` with helpers in
`internal/install/`.

**Done when:**

- `shy install ./test-bundle/` succeeds on a local manifest+files
  bundle
- Each item lands in the correct namespaced subdirectory
- `cache.json` records the installation
- Test fixtures for each item type committed under
  `internal/install/testdata/`

---

## Steps 13–18 — Remaining Phase 2 commands

Pattern continues for:

- **Step 13** — `shy list` with `--type` filtering, colour output,
  and `--sources` flag for ownership trace
- **Step 14** — `shy info <namespace>/<name>` rendering README.md
  via glamour; `--raw` for raw markdown output
- **Step 15** — `shy install` for URLs and `@user/repo` (uses git
  clone), with default-pinning to current HEAD commit
- **Step 16** — `shy alias <name>='<value>'` imperative helper
- **Step 17** — `shy completion add <tool>` imperative helper
- **Step 18** — `shy remove <namespace>/<name>`
- **Step 19** — `shy update [<name>]` with refusal when `[source]`
  is missing

Each step follows the same shape: implement the subcommand in
`internal/cmd/`, add unit tests, integration-test on author's
machines, commit with a clear message.

---

## Phase 1 + Phase 2 acceptance test

When Steps 1–19 are complete:

```bash
# On a clean machine:
curl -fsSL https://raw.githubusercontent.com/alfred-intelligence/shy/main/install.sh | bash
shy init
shy alias 'll=ls -alh'
shy completion add gh
shy list --sources
# Expect: aliases (ll, source=local), completions (gh, source=local)

# In a new shell:
ll
# Expect: detailed directory listing
shy <tab>
# Expect: subcommand completions
```

If this sequence works without manual intervention, Phase 1 + Phase
2 are complete and Phase 3 (collections) can begin.

---

## Operational notes

**Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, etc.).
Effect, not implementation. Short.

**Branches:** Direct push to `main` allowed for the operator;
external contributors via PR.

**Code style:** `gofmt`, `go vet`, `golangci-lint` (configured in
`.golangci.yaml` — added in Step 6 as part of CI). Bash:
`shellcheck` clean.

**Documentation language:** English everywhere in the repository.
Operator dialogue (issues, conversations) may be Swedish.

**Comments in code:** intention, not implementation. One sentence,
max one-and-a-half. Reasoning that does not fit goes in `docs/` or
a PR description.

**Test discipline:** every command needs at least one happy-path
test before it lands in `main`. Edge-case coverage can come after.
