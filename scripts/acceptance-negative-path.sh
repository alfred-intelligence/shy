#!/usr/bin/env bash
# acceptance-negative-path.sh — every command that should refuse does
# so with a meaningful diagnostic, and exits non-zero.

set -uo pipefail   # not -e — we expect failures

bin="${SHY_BIN:-./dist/shy}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

export SHY_HOME="$tmp/home"
export SHY_TEST_BASHRC="$tmp/.bashrc"

assert_fails() {
    local label="$1"; shift
    if "$@" >"$tmp/out" 2>"$tmp/err"; then
        echo "[$label] expected failure, got success"; cat "$tmp/err" >&2
        exit 1
    fi
    if [[ ! -s "$tmp/err" && ! -s "$tmp/out" ]]; then
        echo "[$label] failed silently — no stderr or stdout diagnostic"; exit 1
    fi
}

# 1. publish a nonexistent item — informative error
"$bin" init >/dev/null
assert_fails "publish-missing" "$bin" publish does-not-exist

# 2. install a path with no manifest.toml — informative error
mkdir -p "$tmp/empty"
assert_fails "install-no-manifest" "$bin" install "$tmp/empty"

# 3. malformed manifest — parse error
mkdir -p "$tmp/bad"
printf 'name = invalid toml ===\n' >"$tmp/bad/manifest.toml"
assert_fails "install-bad-manifest" "$bin" install "$tmp/bad"

# 4. semver override that isn't a semver — validation error
mkdir -p "$tmp/draft" "$SHY_HOME/installed/%host/draft"
echo '#!/usr/bin/env bash' >"$SHY_HOME/installed/%host/draft/entry.sh"
assert_fails "publish-bad-version" "$bin" publish draft --version not-a-version

# 5. override add without root — refusal
assert_fails "override-not-root" "$bin" override add alias/ll

# 6. system-reset without --yes-i-know — refusal
SHY_TEST_FAKE_ROOT=1 "$bin" system-reset >"$tmp/out" 2>"$tmp/err" || true
if grep -q "yes-i-know" "$tmp/err" "$tmp/out"; then
    :
else
    echo "[system-reset-without-flag] missing --yes-i-know hint"; exit 1
fi

echo "acceptance-negative-path: PASS"
