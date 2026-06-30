#!/usr/bin/env bash
# ground0/parity-test.sh — §10 acceptance gate for ground0-bash vs Go-shy.
#
# Asserts:
#   (a) same set of runtime artifact paths
#   (b) byte-identical content of every sourced file
#   (c) same shell functions and aliases after sourcing each tree
#
# Usage: bash ground0/parity-test.sh [go-shy-binary] [stdlib-dir]
# Defaults: /tmp/shy-ref  and  <repo>/examples/stdlib
#
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_SHY="${1:-/tmp/shy-ref}"
STDLIB="${2:-${REPO_ROOT}/examples/stdlib}"
G0_SHY="${REPO_ROOT}/ground0/shy"

REF_GO="${TMPDIR:-/tmp}/parity-go-$$"
REF_BASH="${TMPDIR:-/tmp}/parity-bash-$$"
BASHRC_GO="${TMPDIR:-/tmp}/parity-bashrc-go-$$"
BASHRC_BASH="${TMPDIR:-/tmp}/parity-bashrc-bash-$$"

PASS=0; FAIL=0
_failures=()

# shellcheck disable=SC2317  # cleanup is called via trap, not directly
cleanup() {
    rm -rf "$REF_GO" "$REF_BASH" "$BASHRC_GO" "$BASHRC_BASH"
}
trap cleanup EXIT INT TERM

log()  { printf '[parity] %s\n' "$*"; }
ok()   { log "PASS: $*"; PASS=$((PASS + 1)); }
fail() { log "FAIL: $*"; FAIL=$((FAIL + 1)); _failures+=("$*"); }

# ---------------------------------------------------------------------------
# Preflight checks
# ---------------------------------------------------------------------------

if [[ ! -x "$GO_SHY" ]]; then
    printf 'parity-test: Go-shy binary not found at %s\n' "$GO_SHY" >&2
    exit 1
fi
if [[ ! -d "$STDLIB" ]]; then
    printf 'parity-test: stdlib dir not found at %s\n' "$STDLIB" >&2
    exit 1
fi
if [[ ! -x "$G0_SHY" ]]; then
    printf 'parity-test: ground0/shy not executable at %s\n' "$G0_SHY" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Step 1 — populate Go-shy tree
# ---------------------------------------------------------------------------

log "Building Go-shy reference tree in $REF_GO"
mkdir -p "$REF_GO"
touch "$BASHRC_GO"
SHY_HOME="$REF_GO" SHY_TEST_BASHRC="$BASHRC_GO" "$GO_SHY" init  >/dev/null
SHY_HOME="$REF_GO" SHY_TEST_BASHRC="$BASHRC_GO" "$GO_SHY" install "$STDLIB" >/dev/null

# ---------------------------------------------------------------------------
# Step 2 — populate ground0-bash tree
# ---------------------------------------------------------------------------

log "Building ground0-bash tree in $REF_BASH"
mkdir -p "$REF_BASH"
touch "$BASHRC_BASH"
SHY_HOME="$REF_BASH" SHY_TEST_BASHRC="$BASHRC_BASH" bash "$G0_SHY" init    >/dev/null
SHY_HOME="$REF_BASH" SHY_TEST_BASHRC="$BASHRC_BASH" bash "$G0_SHY" install "$STDLIB" >/dev/null

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Collect runtime artifact paths relative to a tree root
_runtime_paths() {
    local root="$1"
    {
        # Script entry points
        find "${root}/installed" -name "entry.sh" -type f 2>/dev/null | \
            sed "s|^${root}/||" | sort
        # Alias files
        find "${root}/helpers/aliases" -type f 2>/dev/null | \
            sed "s|^${root}/||" | sort
        # Completion files
        find "${root}/helpers/completions" -type f 2>/dev/null | \
            sed "s|^${root}/||" | sort
    } | sort
}

# ---------------------------------------------------------------------------
# (a) Same set of runtime artifact paths
# ---------------------------------------------------------------------------

log "--- (a) Checking runtime artifact path sets ---"

paths_go="$(_runtime_paths "$REF_GO")"
paths_bash="$(_runtime_paths "$REF_BASH")"

if [[ "$paths_go" == "$paths_bash" ]]; then
    ok "artifact path sets are identical"
    log "$paths_go"
else
    fail "artifact path sets differ"
    diff <(printf '%s\n' "$paths_go") <(printf '%s\n' "$paths_bash") || true
fi

# ---------------------------------------------------------------------------
# (b) Byte-identical content of every sourced file
# ---------------------------------------------------------------------------

log "--- (b) Checking byte-identical sourced file content ---"

# Collect relative paths of all sourced files from the Go tree
mapfile -t sourced_files < <(
    {
        find "${REF_GO}/installed" -name "entry.sh" -type f 2>/dev/null
        find "${REF_GO}/helpers/aliases"      -type f 2>/dev/null
        find "${REF_GO}/helpers/completions"  -type f 2>/dev/null
    } | sed "s|^${REF_GO}/||" | sort
)

b_pass=0; b_fail=0
for rel in "${sourced_files[@]}"; do
    go_file="${REF_GO}/${rel}"
    bash_file="${REF_BASH}/${rel}"

    if [[ ! -f "$bash_file" ]]; then
        fail "missing in bash tree: $rel"
        b_fail=$((b_fail + 1))
        continue
    fi

    if cmp -s "$go_file" "$bash_file"; then
        b_pass=$((b_pass + 1))
    else
        fail "content differs: $rel"
        b_fail=$((b_fail + 1))
        diff "$go_file" "$bash_file" || true
    fi
done

if [[ $b_fail -eq 0 ]]; then
    ok "all $b_pass sourced files are byte-identical"
else
    log "  $b_pass files identical, $b_fail differ"
fi

# ---------------------------------------------------------------------------
# (c) Shell introspection after sourcing each tree
# ---------------------------------------------------------------------------

log "--- (c) Checking shell function/alias parity after sourcing ---"

# Functions and aliases verified in the probe script below (listed here for documentation)
# Scripts define: mkcd extract serve up path-list
# Aliases defined: gst ll la

# Build a probe script that sources a tree and then prints type/alias output
_make_probe() {
    local home="$1"
    cat <<PROBE
#!/usr/bin/env bash
set +e
export SHY_HOME="${home}"
# source the init script
source "${home}/init.bash" 2>/dev/null || true

# Check functions
for fn in mkcd extract serve up; do
    t=\$(type -t "\$fn" 2>/dev/null || printf 'missing')
    printf 'type:%s=%s\n' "\$fn" "\$t"
done
# hyphenated function
t=\$(type -t 'path-list' 2>/dev/null || printf 'missing')
printf 'type:path-list=%s\n' "\$t"

# Check aliases
alias_out=\$(alias gst ll la 2>/dev/null || true)
printf '%s\n' "\$alias_out"
PROBE
}

probe_go="$(mktemp /tmp/probe-go-XXXXXX.sh)"
probe_bash="$(mktemp /tmp/probe-bash-XXXXXX.sh)"

_make_probe "$REF_GO"   > "$probe_go"
_make_probe "$REF_BASH" > "$probe_bash"

out_go="$(bash --norc "$probe_go" 2>/dev/null)"
out_bash="$(bash --norc "$probe_bash" 2>/dev/null)"

rm -f "$probe_go" "$probe_bash"

if [[ "$out_go" == "$out_bash" ]]; then
    ok "shell introspection output is identical"
    log "$out_go"
else
    fail "shell introspection output differs"
    diff <(printf '%s\n' "$out_go") <(printf '%s\n' "$out_bash") || true
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

log ""
log "========================================"
log "Results: $PASS passed, $FAIL failed"
log "========================================"

if [[ $FAIL -gt 0 ]]; then
    log "FAILURES:"
    for f in "${_failures[@]}"; do
        log "  - $f"
    done
    exit 1
fi

log "ALL CHECKS PASSED — ground0-bash is at §10 parity with Go-shy"
exit 0
