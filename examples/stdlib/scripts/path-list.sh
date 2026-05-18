#!/usr/bin/env bash
# path-list — print $PATH one entry per line.

path-list() {
    printf '%s\n' "${PATH//:/$'\n'}"
}
