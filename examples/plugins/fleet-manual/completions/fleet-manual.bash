# fleet-manual.bash — bash tab-completion for `shy man`, `shy tldr`,
# and `shy who` colleague-name arguments.
# SPDX-License-Identifier: MPL-2.0
#
# Ships as a static file because shy-core does not yet wire the
# plugin `__complete` convention (docs/01-whitepaper.md "Plugin
# completion conventions") into its cobra-generated `shy completion
# bash` output — plugin commands are dispatched before cobra ever
# parses argv (internal/cmd/plugin_dispatch.go), so they are invisible
# to cobra's own completion machinery. See ../README.md "shy help &
# completion: what shy-core actually supports today" for the verified
# gap and the follow-up filed for shy-core (out of scope for this
# plugin — no shy-core edits here).
#
# Install (until the core gap above is closed):
#   echo 'source /path/to/fleet-manual/completions/fleet-manual.bash' >> ~/.bashrc
# `shy install` only copies each package's own entry script + its
# `_`-prefixed helpers + manifest.toml/README.md (internal/install
# copyHelpers) — this completions/ directory is a sibling of the
# man/, tldr/, who/ packages, not inside any of them, so it is never
# copied by `shy install`. Source it straight from the checkout, or
# copy/symlink it into your own bash-completion.d by hand.
#
# zsh: not shipped (bash required by the task; zsh users can `bashcompinit`
# and source this file, or wire zsh's own completion — optional, not done
# here).

# _fleet_manual_colleagues is the completion-time roster allowlist.
# Duplicated from _lib.sh's fleet_roster_names on purpose: bash
# completion runs in the *calling* shell before any plugin entry
# script execs, so it cannot reliably source an installed plugin's
# helper across every install layout. Keep both lists in sync by hand
# if the roster allowlist ever changes (noted again in ../README.md).
# frida is deliberately absent from this base list — see
# _fleet_manual_candidates for how she is conditionally added back.
_fleet_manual_colleagues() {
    printf '%s\n' \
        librarian priest master-at-arms wizard warden inquisitor \
        physician governator lagman jenny herald jester \
        stonemason labassistent alchemist coinmaster
}

# _fleet_manual_candidates prints completion candidates for a
# man/tldr colleague-name argument, given the current partial word
# ($1). frida is offered ONLY once the typed prefix already begins
# her name (board #55 item 6, binding hidden-colleague rule) — an
# empty prefix (bare TAB) must NOT surface her, matching "listings
# exclude her by default" everywhere else in this plugin.
_fleet_manual_candidates() {
    local cur="$1"
    _fleet_manual_colleagues
    if [[ -n "$cur" && "frida" == "$cur"* ]]; then
        printf 'frida\n'
    fi
}

_fleet_manual_complete() {
    local cur cmd
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    cmd="${COMP_WORDS[1]-}"

    case "$cmd" in
        man|tldr)
            if [[ "$COMP_CWORD" -eq 2 ]]; then
                mapfile -t COMPREPLY < <(compgen -W "$(_fleet_manual_candidates "$cur")" -- "$cur")
                return 0
            fi
            ;;
        who)
            # Free-text task phrase — nothing to complete.
            return 0
            ;;
    esac

    # Not one of ours: delegate to shy's own completion function
    # (`_shy`, registered by `shy completion bash`) if it is already
    # loaded, so sourcing this file does not break native `shy
    # install`/`shy list`/... completion.
    if declare -F _shy >/dev/null 2>&1; then
        _shy
        return $?
    fi
    return 0
}

complete -F _fleet_manual_complete shy
