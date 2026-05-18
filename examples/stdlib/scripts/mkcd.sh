#!/usr/bin/env bash
# mkcd — create a directory (with -p) and cd into it.

mkcd() {
    if [[ $# -ne 1 ]]; then
        echo "usage: mkcd <dir>" >&2
        return 2
    fi
    mkdir -p -- "$1" && cd -- "$1" || return
}
