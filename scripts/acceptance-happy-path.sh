#!/usr/bin/env bash
# acceptance-happy-path.sh — the v1.0 acceptance sequence from
# docs/02-long-horizon, runnable both in a container and on a host.
#
# Inputs (env):
#   SHY_BIN       — path to a pre-built shy binary (default: ./dist/shy)
#   SHY_HOME      — install target (default: a fresh tempdir)
#   SHY_STDLIB    — local stdlib path (default: ./examples/stdlib)

set -euo pipefail

bin="${SHY_BIN:-./dist/shy}"
stdlib="${SHY_STDLIB:-./examples/stdlib}"

if [[ ! -x "$bin" ]]; then
    echo "acceptance: $bin is not executable; build first" >&2
    exit 2
fi
if [[ ! -d "$stdlib" ]]; then
    echo "acceptance: stdlib path $stdlib does not exist" >&2
    exit 2
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

export SHY_HOME="${SHY_HOME:-$tmp/home}"
export SHY_TEST_BASHRC="$tmp/.bashrc"
mkdir -p "$SHY_HOME"

assert_eq() {
    if [[ "$1" != "$2" ]]; then
        printf 'assert_eq failed:\n  got:  %q\n  want: %q\n' "$1" "$2" >&2
        exit 1
    fi
}

# 1. init
"$bin" init >/dev/null
[[ -f "$SHY_HOME/init.bash" ]] || { echo "init.bash missing"; exit 1; }
grep -q 'shy/init.bash' "$SHY_TEST_BASHRC" || { echo "bashrc not configured"; exit 1; }

# 2. install stdlib
"$bin" install "$stdlib" >/dev/null

# 3. create + publish a fresh script
export EDITOR=true
export HOME="$tmp"
git config --global user.name "acceptance"
git config --global user.email "acceptance@example.com"
"$bin" create my-first --no-editor >/dev/null
"$bin" publish my-first --version 0.1.0 </dev/null >/dev/null
test -f "$SHY_HOME/installed/%acceptance/my-first/manifest.toml" || {
    echo "publish did not move to author namespace"; exit 1; }

# 4. install reference plugin and dispatch
"$bin" install ./examples/plugins/hello-world >/dev/null
got=$("$bin" hello-world acceptance)
assert_eq "$got" "hello, acceptance"

# 5. list + JSON contract
"$bin" list --json >"$tmp/list.json"
grep -q '"installed"' "$tmp/list.json" || grep -q '"items"' "$tmp/list.json" || {
    echo "list --json missing items field"; exit 1; }

# 6. shy-reload alias and source the new script through init.bash
SHY_TEST_BASHRC="$SHY_TEST_BASHRC" SHY_HOME="$SHY_HOME" bash -c '
    source "$SHY_HOME/init.bash"
    type my_first >/dev/null || { echo "my-first function not defined"; exit 1; }
    type mkcd >/dev/null     || { echo "mkcd function not defined";     exit 1; }
'

echo "acceptance-happy-path: PASS"
