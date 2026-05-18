#!/usr/bin/env bash
# install.sh — curl|bash entry point for shy.
# This file is a permanent contract from v1.0.0; the asset-name schema,
# URL location, and behavioural expectations may not change after v1.0.0
# without breaking all cached installer copies in the wild.

set -euo pipefail

VERSION="${SHY_VERSION:-latest}"
PREFIX="${SHY_HOME:-$HOME/.shy}"
REPO="alfred-intelligence/shy"
LOCKFILE="${TMPDIR:-/tmp}/shy-install.lock"

die() {
    echo "shy: $*" >&2
    exit 1
}

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found on PATH: $1"
}

cleanup() {
    rm -rf "$tmp" 2>/dev/null || true
    rm -f "$LOCKFILE" 2>/dev/null || true
}

# Refuse to run as root; system-wide installs use the .deb/.rpm packages.
if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    die "refusing to install as root. Use the .deb or .rpm package for system-wide installs, or rerun as your normal user."
fi

# Lock to prevent parallel invocations from corrupting $PREFIX/bin/.
exec 9>"$LOCKFILE"
if ! flock -n 9; then
    die "another shy install is in progress (lockfile: $LOCKFILE). Re-run when it completes."
fi

require_cmd curl
require_cmd tar
require_cmd sha256sum

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
    linux|darwin) ;;
    *) die "unsupported OS: $os (supported: linux, darwin)" ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "unsupported arch: $arch (supported: amd64, arm64)" ;;
esac

if [[ "$VERSION" == "latest" ]]; then
    api_url="https://api.github.com/repos/$REPO/releases/latest"
    VERSION=$(curl -fsSL "$api_url" 2>/dev/null \
              | grep '"tag_name"' | head -1 | cut -d'"' -f4 || true)
fi
[[ -n "$VERSION" ]] || die "could not resolve a release version from GitHub. Set SHY_VERSION=vX.Y.Z explicitly or check your network."

asset="shy_${VERSION#v}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$asset"
sum_url="$url.sha256"

tmp=$(mktemp -d)
trap cleanup EXIT INT TERM

echo "shy: downloading $asset"
if ! curl -fsSL "$url" -o "$tmp/$asset"; then
    die "download failed: $url. Check your network or that $VERSION exists at https://github.com/$REPO/releases."
fi
if ! curl -fsSL "$sum_url" -o "$tmp/$asset.sha256"; then
    die "checksum download failed: $sum_url. The release may be incomplete; try a different version with SHY_VERSION=vX.Y.Z."
fi

if ! ( cd "$tmp" && sha256sum -c "$asset.sha256" >/dev/null 2>&1 ); then
    die "SHA256 mismatch for $asset. The download is corrupted or has been tampered with — refusing to install."
fi

mkdir -p "$PREFIX/bin"
# Stage to a temporary path inside $PREFIX/bin so a half-extracted file
# cannot leave the install in a broken state.
tar -xzf "$tmp/$asset" -C "$tmp" shy
install -m 0755 "$tmp/shy" "$PREFIX/bin/shy.new"
mv -f "$PREFIX/bin/shy.new" "$PREFIX/bin/shy"

"$PREFIX/bin/shy" init

echo "shy: $VERSION installed at $PREFIX/bin/shy"
echo "shy: open a new shell or run \`source $PREFIX/init.bash\` to activate."
