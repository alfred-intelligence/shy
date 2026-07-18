#!/usr/bin/env bash
# install.sh — curl|bash entry point for shy.
# This file is a permanent contract from v1.0.0; the asset-name schema,
# URL location, and behavioural expectations may not change after v1.0.0
# without breaking all cached installer copies in the wild.
#
# Usage: install.sh [--system] [--no-bashrc] [--help]
#   --system     install system-wide to /usr/local/bin/shy (requires root/sudo)
#   --no-bashrc  skip adding the source line to ~/.bashrc
#   --help       print this help

set -euo pipefail

VERSION="${SHY_VERSION:-latest}"
PREFIX="${SHY_HOME:-${HOME:-}/.shy}"
REPO="alfred-intelligence/shy"
SYSTEM_BINDIR="/usr/local/bin"
LOCKDIR="${TMPDIR:-/tmp}/shy-install.lock"

die() {
    echo "shy: $*" >&2
    exit 1
}

usage() {
    cat <<'EOF'
usage: install.sh [--system] [--no-bashrc] [--help]
  --system     install system-wide to /usr/local/bin/shy (requires root/sudo)
  --no-bashrc  skip adding the source line to ~/.bashrc
  --help       print this help
EOF
}

system_install=""
no_bashrc=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --system) system_install=1 ;;
        --no-bashrc) no_bashrc=1 ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option: $1 (supported: --system, --no-bashrc, --help)" ;;
    esac
    shift
done

require_cmd() {
    command -v "$1" >/dev/null 2>&1 \
        || die "required command not found on PATH: $1. Install it with your package manager and re-run."
}

# sha256 <file> — portable digest (GNU sha256sum, else BSD/macOS shasum).
sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
}

have_lock=""
tmp=""
cleanup() {
    if [[ -n "$tmp" ]]; then
        rm -rf "$tmp" 2>/dev/null || true
    fi
    if [[ -n "$have_lock" ]]; then
        rm -rf "$LOCKDIR" 2>/dev/null || true
    fi
}

if [[ -n "$system_install" ]]; then
    # System-wide: binary only, into /usr/local/bin. Users run `shy init`
    # themselves; no per-user state is touched.
    [[ "${EUID:-$(id -u)}" -eq 0 ]] \
        || die "--system requires root. Re-run with: curl -fsSL ... | sudo bash -s -- --system"
else
    # Refuse to run as root for a user-level install; system-wide installs
    # use --system or the .deb/.rpm packages.
    if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
        die "refusing user-level install as root. Use --system (or the .deb/.rpm package) for system-wide installs, or rerun as your normal user."
    fi
    [[ -n "${HOME:-}" ]] || die "\$HOME is not set; cannot do a user-level install. Set HOME or use --system."
fi

require_cmd curl
require_cmd tar
command -v sha256sum >/dev/null 2>&1 || command -v shasum >/dev/null 2>&1 \
    || die "required command not found on PATH: sha256sum (or shasum)."

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
    linux|darwin) ;;
    *) die "unsupported OS: $os (supported: linux, darwin). Build from source instead: go install github.com/$REPO/cmd@latest" ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "unsupported arch: $arch (supported: amd64, arm64). Build from source instead: go install github.com/$REPO/cmd@latest" ;;
esac

# Lock to prevent parallel invocations from corrupting the install dir.
# mkdir is atomic and portable (flock is not available on macOS).
if ! mkdir "$LOCKDIR" 2>/dev/null; then
    other_pid=$(cat "$LOCKDIR/pid" 2>/dev/null || true)
    if [[ -n "$other_pid" ]] && ! kill -0 "$other_pid" 2>/dev/null; then
        # Stale lock: owner is gone (crashed or was killed). Take it over.
        rm -rf "$LOCKDIR" 2>/dev/null || true
        mkdir "$LOCKDIR" 2>/dev/null \
            || die "another shy install is in progress (lock: $LOCKDIR). Re-run when it completes."
    else
        die "another shy install is in progress (lock: $LOCKDIR). Re-run when it completes, or remove the lock if it is stale."
    fi
fi
have_lock=1
trap cleanup EXIT INT TERM
echo "$$" > "$LOCKDIR/pid"

if [[ "$VERSION" == "latest" ]]; then
    api_url="https://api.github.com/repos/$REPO/releases/latest"
    VERSION=$(curl -fsSL "$api_url" 2>/dev/null \
              | grep '"tag_name"' | head -1 | cut -d'"' -f4 || true)
fi
[[ -n "$VERSION" ]] || die "could not resolve a release version from GitHub. Set SHY_VERSION=vX.Y.Z explicitly or check your network."

asset="shy_${VERSION#v}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/$VERSION"
url="$base_url/$asset"

tmp=$(mktemp -d)

echo "shy: downloading $asset"
if ! curl -fsSL "$url" -o "$tmp/$asset"; then
    die "download failed: $url. Check your network or that $VERSION exists at https://github.com/$REPO/releases."
fi

# Verify against the release SHA256SUMS before unpacking anything. Older
# releases (< v0.3) shipped per-asset .sha256 files instead; fall back.
expected=""
if curl -fsSL "$base_url/SHA256SUMS" -o "$tmp/SHA256SUMS" 2>/dev/null; then
    expected=$(awk -v a="$asset" '$2 == a || $2 == "*"a {print $1; exit}' "$tmp/SHA256SUMS")
    [[ -n "$expected" ]] || die "$asset is not listed in SHA256SUMS for $VERSION. The release may be incomplete — refusing to install."
elif curl -fsSL "$url.sha256" -o "$tmp/$asset.sha256" 2>/dev/null; then
    expected=$(awk '{print $1; exit}' "$tmp/$asset.sha256")
fi
[[ -n "$expected" ]] || die "checksum download failed for $VERSION (tried SHA256SUMS and $asset.sha256). The release may be incomplete; try a different version with SHY_VERSION=vX.Y.Z."

actual=$(sha256 "$tmp/$asset")
if [[ "$expected" != "$actual" ]]; then
    die "SHA256 mismatch for $asset. The download is corrupted or has been tampered with — refusing to install."
fi

tar -xzf "$tmp/$asset" -C "$tmp" shy

if [[ -n "$system_install" ]]; then
    mkdir -p "$SYSTEM_BINDIR"
    # Stage next to the target so a half-written file cannot leave a
    # broken binary in place; mv on the same filesystem is atomic.
    install -m 0755 "$tmp/shy" "$SYSTEM_BINDIR/shy.new"
    mv -f "$SYSTEM_BINDIR/shy.new" "$SYSTEM_BINDIR/shy"
    echo "shy: $VERSION installed at $SYSTEM_BINDIR/shy"
    echo "shy: each user should now run \`shy init\` to set up their own \$HOME/.shy/."
    exit 0
fi

mkdir -p "$PREFIX/bin"
install -m 0755 "$tmp/shy" "$PREFIX/bin/shy.new"
mv -f "$PREFIX/bin/shy.new" "$PREFIX/bin/shy"

# Symlink shy into a standard personal bin directory so it is reachable
# without relying on init.bash being sourced (other shells, scripts, sudo -u).
# Prefer ~/.local/bin (XDG), then ~/bin; create ~/.local/bin if neither exists.
_shy_link_to_path() {
    local target="$PREFIX/bin/shy"
    local dir
    for dir in "$HOME/.local/bin" "$HOME/bin"; do
        if [[ -d "$dir" ]]; then
            ln -sf "$target" "$dir/shy" 2>/dev/null && return 0
        fi
    done
    if mkdir -p "$HOME/.local/bin" 2>/dev/null; then
        ln -sf "$target" "$HOME/.local/bin/shy" 2>/dev/null || true
    fi
}
_shy_link_to_path
unset -f _shy_link_to_path

# Ask before touching ~/.bashrc when a terminal is available. Under
# `curl | bash` stdin is the pipe, so read the answer from /dev/tty;
# with no terminal (CI, provisioning) default to adding the line.
# The subshell open is the reliable controlling-terminal test: -r/-w on
# /dev/tty can pass on permission bits even when no terminal is attached.
if [[ -z "$no_bashrc" ]] && (exec < /dev/tty) 2>/dev/null; then
    printf 'shy: add the init source line to ~/.bashrc? [Y/n] ' > /dev/tty
    answer=""
    read -r answer < /dev/tty || true
    case "$answer" in
        n|N|no|NO|No) no_bashrc=1 ;;
    esac
fi

if [[ -n "$no_bashrc" ]]; then
    "$PREFIX/bin/shy" init --no-bashrc
else
    "$PREFIX/bin/shy" init
fi

echo "shy: $VERSION installed at $PREFIX/bin/shy"
echo "shy: open a new shell or run \`source $PREFIX/init.bash\` to activate."
