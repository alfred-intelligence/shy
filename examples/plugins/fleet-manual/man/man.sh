#!/usr/bin/env bash
# man.sh — `shy man <colleague>`: full charter view, minus the
# Prompt Defense Baseline boilerplate. See ./README.md.
# SPDX-License-Identifier: MPL-2.0
set -euo pipefail

# shellcheck source=./_lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# __complete: plugin tab-completion convention (docs/01-whitepaper.md
# "Plugin completion conventions", Convention 1). shy-core does not
# yet dispatch __complete into real shell tab-completion — see
# ./README.md for the verified gap and the follow-up filed for
# shy-core. Kept anyway for forward-compat and for anyone driving the
# plugin directly. Emits the default (frida-filtered) roster list;
# real completion (frida-aware, prefix-aware) is handled by
# ./completions/fleet-manual.bash instead.
if [[ "${1-}" == "__complete" ]]; then
    fleet_roster_list
    exit 0
fi

case "${1-}" in
    ""|help|--help|-h)
        fleet_usage_line man "full charter view for one fleet colleague"
        printf 'Usage: shy man <colleague>\n\n'
        printf 'Colleagues:\n'
        fleet_roster_list | sed 's/^/  /'
        exit 0
        ;;
esac

name="$1"
file="$(fleet_agent_file "$name")"

# frida is invisible in listings, never in a direct, explicit lookup —
# typing her name IS "the query phrase names her explicitly" (binding
# rule, DESIGN.md §5).
if ! fleet_is_frida "$name" && ! fleet_roster_list | grep -qx "$name"; then
    printf 'shy man: %s is not in the colleague-stab roster (run "shy man" for the list)\n' "$name" >&2
    exit 1
fi

if [[ ! -f "$file" ]]; then
    printf 'shy man: no agent file at %s\n' "$file" >&2
    exit 1
fi

{
    printf '%s\n\n' "$name"
    fleet_strip_pdb "$file"
} | fleet_render
