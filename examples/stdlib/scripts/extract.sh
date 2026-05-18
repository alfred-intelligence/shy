#!/usr/bin/env bash
# extract — unpack an archive based on its file extension.

extract() {
    if [[ $# -ne 1 || ! -f "$1" ]]; then
        echo "usage: extract <archive>" >&2
        return 2
    fi
    case "$1" in
        *.tar.bz2|*.tbz2) tar xjf "$1" ;;
        *.tar.gz|*.tgz)   tar xzf "$1" ;;
        *.tar.xz|*.txz)   tar xJf "$1" ;;
        *.tar.zst)        tar --zstd -xf "$1" ;;
        *.tar)            tar xf "$1" ;;
        *.zip)            unzip "$1" ;;
        *.7z)             7z x "$1" ;;
        *.rar)            unrar x "$1" ;;
        *.gz)             gunzip "$1" ;;
        *.bz2)            bunzip2 "$1" ;;
        *.xz)             unxz "$1" ;;
        *.zst)            unzstd "$1" ;;
        *)
            echo "extract: don't know how to handle $1" >&2
            return 1
            ;;
    esac
}
