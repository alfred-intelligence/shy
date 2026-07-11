#!/usr/bin/env bash
# tldr.sh — `shy tldr <colleague>`: 5-line compressed summary. See
# ./README.md.
# SPDX-License-Identifier: MPL-2.0
set -euo pipefail

# shellcheck source=./_lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

if [[ "${1-}" == "__complete" ]]; then
    fleet_roster_list
    exit 0
fi

case "${1-}" in
    ""|help|--help|-h)
        fleet_usage_line tldr "5-line compressed summary for one fleet colleague"
        printf 'Usage: shy tldr <colleague>\n\n'
        printf 'Colleagues:\n'
        fleet_roster_list | sed 's/^/  /'
        exit 0
        ;;
esac

name="$1"
file="$(fleet_agent_file "$name")"

if ! fleet_is_frida "$name" && ! fleet_roster_list | grep -qx "$name"; then
    printf 'shy tldr: %s is not in the colleague-stab roster (run "shy tldr" for the list)\n' "$name" >&2
    exit 1
fi

if [[ ! -f "$file" ]]; then
    printf 'shy tldr: no agent file at %s\n' "$file" >&2
    exit 1
fi

desc="$(fleet_extract_frontmatter "$file" | cut -f2-)"
if [[ -z "$desc" ]]; then
    printf 'shy tldr: %s has no description field in %s\n' "$name" "$file" >&2
    exit 1
fi

fleet_tldr "$name" "$desc"
