#!/usr/bin/env bash
# hello-world — reference plugin for shy.
# Invoked as `shy hello-world [<name>]`.
set -euo pipefail

# __complete is the convention for plugin tab-completion. shy invokes
# this with the arguments typed so far; one candidate per line on
# stdout.
if [[ "${1-}" == "__complete" ]]; then
    shift
    case "$#" in
        0) printf 'world\nfriend\noperator\n' ;;
    esac
    exit 0
fi

name="${1-world}"
echo "hello, ${name}"
