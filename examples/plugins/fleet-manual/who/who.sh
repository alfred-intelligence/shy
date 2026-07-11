#!/usr/bin/env bash
# who.sh — `shy who <task-phrase>`: grep-based routing suggestion,
# never a dispatcher. See ./README.md.
# SPDX-License-Identifier: MPL-2.0
set -euo pipefail

# shellcheck source=./_lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

if [[ "${1-}" == "__complete" ]]; then
    # who takes a free-text phrase, not a colleague name — nothing to
    # complete against the roster.
    exit 0
fi

case "${1-}" in
    ""|help|--help|-h)
        fleet_usage_line who "routing suggestion for a task phrase (never dispatches)"
        printf 'Usage: shy who "<task phrase>"\n'
        exit 0
        ;;
esac

phrase="$*"

include_frida=0
if fleet_query_names_frida "$phrase"; then
    include_frida=1
fi

# Word-split the phrase (case-insensitive), matching DESIGN.md §2.
words_flat="$(printf '%s' "$phrase" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9' ' ')"
read -ra words <<< "$words_flat"
total="${#words[@]}"
if [[ "$total" -eq 0 ]]; then
    printf 'shy who: empty phrase\n' >&2
    exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

roster="$(fleet_roster_list)"
if [[ "$include_frida" -eq 1 ]]; then
    roster="${roster}
frida"
fi

while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    file="$(fleet_agent_file "$name")"
    [[ -f "$file" ]] || continue
    desc_lc="$(fleet_extract_frontmatter "$file" | cut -f2- | tr '[:upper:]' '[:lower:]')"
    hits=0
    for w in "${words[@]}"; do
        [[ -z "$w" ]] && continue
        if [[ "$desc_lc" == *"$w"* ]]; then
            hits=$((hits + 1))
        fi
    done
    if [[ "$hits" -gt 0 ]]; then
        printf '%s\t%s\n' "$hits" "$name" >> "$tmp"
    fi
done <<< "$roster"

if [[ ! -s "$tmp" ]]; then
    printf 'shy who: no roster match for "%s"\n' "$phrase"
    exit 0
fi

first=1
while IFS=$'\t' read -r hits name; do
    if [[ "$first" -eq 1 ]]; then
        printf 'best match: %-12s (%s/%s words hit)\n' "$name" "$hits" "$total"
        first=0
    else
        printf 'also:       %-12s (%s/%s words hit)\n' "$name" "$hits" "$total"
    fi
done < <(sort -t "$(printf '\t')" -k1,1nr -k2,2 "$tmp")
