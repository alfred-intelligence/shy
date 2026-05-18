#!/usr/bin/env bash
# install.sh — curl|bash entry point for shy.
# This file is a permanent contract from v1.0.0; the asset-name schema,
# URL location, and behavioural expectations may not change after v1.0.0
# without breaking all cached installer copies in the wild.

set -euo pipefail

VERSION="${SHY_VERSION:-latest}"
PREFIX="${SHY_HOME:-$HOME/.shy}"
REPO="alfred-intelligence/shy"

# Refuse to run as root; system-wide installs use the .deb/.rpm packages.
if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    echo "shy: refusing to install as root." >&2
    echo "shy: use the .deb or .rpm package for system-wide installs," >&2
    echo "shy: or rerun this script as your normal user for $HOME/.shy." >&2
    exit 1
fi

for cmd in curl tar sha256sum; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "shy: required command not found on PATH: $cmd" >&2
        exit 1
    fi
done

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
    linux|darwin) ;;
    *) echo "shy: unsupported OS: $os" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "shy: unsupported arch: $arch" >&2; exit 1 ;;
esac

if [[ "$VERSION" == "latest" ]]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
              | grep '"tag_name"' | head -1 | cut -d'"' -f4 || true)
fi
[[ -n "$VERSION" ]] || { echo "shy: could not resolve a release version." >&2; exit 1; }

asset="shy_${VERSION#v}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$asset"
sum_url="$url.sha256"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "shy: downloading $asset"
curl -fsSL "$url"     -o "$tmp/$asset"
curl -fsSL "$sum_url" -o "$tmp/$asset.sha256"

( cd "$tmp" && sha256sum -c "$asset.sha256" >/dev/null )

mkdir -p "$PREFIX/bin"
tar -xzf "$tmp/$asset" -C "$PREFIX/bin" shy
chmod +x "$PREFIX/bin/shy"

"$PREFIX/bin/shy" init

echo "shy: $VERSION installed at $PREFIX/bin/shy"
echo "shy: open a new shell or run \`source $PREFIX/init.bash\` to activate."
