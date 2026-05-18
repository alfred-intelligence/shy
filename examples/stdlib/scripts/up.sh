#!/usr/bin/env bash
# up — cd up N parent directories (default 1).

up() {
    local count="${1:-1}"
    local path=""
    local i
    for ((i = 0; i < count; i++)); do
        path="../$path"
    done
    cd -- "$path" || return
}
