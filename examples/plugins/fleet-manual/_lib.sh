#!/usr/bin/env bash
# _lib.sh — shared helpers for fleet-manual (man/tldr/who).
# SPDX-License-Identifier: MPL-2.0
#
# Underscore-prefixed on purpose: shy's installer copies only
# `_*.sh` siblings alongside each item's entry.sh (internal/install
# copyHelpers), so this file rides along with man.sh/tldr.sh/who.sh
# without becoming a dispatchable command of its own.
set -euo pipefail

# fleet_agents_dir prints the roster directory for the invoking
# identity's own seat only — no cross-home/cross-host reads (board
# #55 operator decision 2026-07-11, item 3). SHY_FLEET_AGENTS_DIR
# exists solely so this plugin's own test runs can point at a fixture
# directory without touching the real ~/.claude/agents.
fleet_agents_dir() {
    printf '%s\n' "${SHY_FLEET_AGENTS_DIR:-$HOME/.claude/agents}"
}

# fleet_roster_names is the explicit colleague-stab allowlist (board
# #55 operator decision 2026-07-11, item 2 — resolves DESIGN.md Fork
# B as B2). These are the ~14-17 human-named colleagues the operator
# dispatches directly, not the 60+ generic ECC specialist suite
# (code-reviewer, tdd-guide, ...) which is invoked BY colleagues, not
# browsed by the operator. A named allowlist, not a marker
# convention, because nothing in the frontmatter schema today
# distinguishes the two groups; adding such a marker would be a
# schema decision out of scope for this plugin. frida is listed here
# but every caller filters her per the binding hidden-colleague rule
# below — she is never dropped from the underlying data, only from
# what gets shown by default.
fleet_roster_names() {
    printf '%s\n' \
        librarian priest master-at-arms wizard warden inquisitor \
        physician governator lagman jenny frida herald jester \
        stonemason labassistent alchemist coinmaster
}

# fleet_is_frida reports whether $1 is the hidden colleague's name.
fleet_is_frida() { [[ "$1" == "frida" ]]; }

# fleet_query_names_frida reports whether the free-text query/phrase
# $* literally names frida or Stash — the only condition (DESIGN.md
# §5, binding, inherited from dante-ops/CLAUDE.md's roster table and
# the standing "ignore stash until mentioned" rule) under which a
# search may surface her.
fleet_query_names_frida() {
    local q
    q="$(printf '%s' "$*" | tr '[:upper:]' '[:lower:]')"
    [[ "$q" == *frida* || "$q" == *stash* ]]
}

# fleet_roster_list prints one name per line for the default,
# frida-filtered roster — used by no-arg listings and __complete.
fleet_roster_list() {
    fleet_roster_names | grep -v '^frida$'
}

# fleet_agent_file prints the frontmatter file path for colleague $1,
# regardless of allowlist membership. Callers decide whether $1 is
# allowed to be looked up: explicit-name access (typing the name) is
# always allowed; listings are pre-filtered before this is ever
# called for enumeration.
fleet_agent_file() {
    printf '%s/%s.md\n' "$(fleet_agents_dir)" "$1"
}

# fleet_extract_frontmatter prints "name<TAB>description" for one
# agent file. Handles both a plain single-line `description: ...`
# scalar and a YAML folded scalar (`description: >-` with indented
# continuation lines) — the two are indistinguishable on disk once
# you're past the first line, so one branch covers both. No jq, no
# YAML library: the schema is small and fixed (name, description,
# model, tools), matching shy-core's own hand-rolled-parser-over-
# dependency philosophy for its manifest.toml.
fleet_extract_frontmatter() {
    awk '
        /^---$/ { fm++; next }
        fm==1 && /^[a-zA-Z_]+:/ {
            key=$0; sub(/:.*/, "", key)
            val=$0; sub(/^[a-zA-Z_]+:[[:space:]]*/, "", val)
            sub(/^[>|]-?[[:space:]]*$/, "", val)
            cur=key
            if (cur=="description") { desc=val; next }
            if (cur=="name") { name=val }
            next
        }
        fm==1 && cur=="description" && /^[[:space:]]/ {
            sub(/^[[:space:]]+/, "", $0)
            desc = desc " " $0
            next
        }
        fm==2 { exit }
        END {
            gsub(/^[[:space:]]+|[[:space:]]+$/, "", desc)
            print name "\t" desc
        }
    ' "$1"
}

# fleet_strip_pdb prints a file's markdown body (everything after the
# closing frontmatter `---`) with the "## Prompt Defense Baseline"
# section removed. That block is bullet-list-shaped and identical
# across every colleague file (verified against governator.md,
# librarian.md, frida.md); stopping the skip at the first line that
# is neither blank nor a bullet correctly keeps each colleague's own
# intro paragraph (e.g. "You are **librarian**, ..."), which always
# precedes the first real "## " heading (typically "## Mission"). A
# naive "delete up to the next ## heading" rule — DESIGN.md's own
# illustrative sketch, §3 [3a] — would wrongly eat that intro prose
# too; this is a deliberate improvement over the sketch, not a fork.
fleet_strip_pdb() {
    awk '
        /^---$/ { fm++; next }
        fm<2 { next }
        /^## Prompt Defense Baseline/ { skip=1; next }
        skip && ($0=="" || /^-/) { next }
        { skip=0; print }
    ' "$1"
}

# fleet_render pipes markdown through the best available prettifier —
# glow (full markdown render), then bat (syntax highlight only), then
# a bare cat. Mirrors DESIGN.md §3.4: shy-core's own binary stays
# dependency-free by embedding glamour, but a *plugin* is free to
# soft-depend on an optional renderer and fall back gracefully.
fleet_render() {
    if command -v glow >/dev/null 2>&1; then
        glow -
    elif command -v bat >/dev/null 2>&1; then
        bat --language=markdown --paging=never --style=plain -
    else
        cat
    fi
}

# fleet_usage_line prints the shared "no args / help" banner so
# man.sh, tldr.sh, and who.sh stay consistent. This doubles as the
# plugin-side answer for board #55 item 5 ("shy help integration"):
# shy-core's cobra `shy help` cannot show a plugin subcommand's own
# *usage text* today (dispatch bypasses cobra entirely for plugin
# commands — see README "shy help & completion: what shy-core
# actually supports today"). `shy man help` / `shy man --help` is the
# plugin-side equivalent.
fleet_usage_line() {
    local cmd="$1" desc="$2"
    printf '%s\n\n' "shy ${cmd} — ${desc}"
}

# fleet_tldr prints the 5-line compressed summary for one colleague,
# derived from the frontmatter `description` field via pattern match
# (DESIGN.md §2 "shy tldr"): purpose = first sentence, triggers = the
# "Use ... " clause, "NOT for" = every "NOT ... (that's X)" clause,
# falling back to a looser "NOT ..." grep for descriptions phrasing it
# differently (e.g. librarian's "NOT the chief of staff — that role is
# direktor's", which has no "(that's X)" parenthetical). Approximate
# by design — the confidence line says so explicitly.
fleet_tldr() {
    local name="$1" desc="$2"
    local purpose triggers notfor
    purpose="${desc%%. *}."
    triggers="$(printf '%s' "$desc" | grep -oE 'Use [^.]*\.' | head -1 || true)"
    notfor="$(printf '%s' "$desc" \
        | grep -oE "NOT [^.]*\\(that's [a-zA-Z0-9_-]+\\)" \
        | sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/; /g' || true)"
    if [[ -z "$notfor" ]]; then
        notfor="$(printf '%s' "$desc" | grep -oE 'NOT [^.—]*' | head -1 | sed -e 's/[[:space:]]*$//' || true)"
    fi
    if [[ -z "$notfor" ]]; then
        notfor="(not stated in the description; read the full charter via: shy man ${name})"
    fi
    printf '%s — %s\n' "$name" "$purpose"
    [[ -n "$triggers" ]] && printf 'triggers: %s\n' "$triggers"
    printf 'NOT for: %s\n' "$notfor"
    printf 'dispatch: delegate to %s via the colleague-stab (dante-ops/CLAUDE.md roster table)\n' "$name"
    printf 'confidence: description-derived, verify against %s if unsure\n' "$(fleet_agent_file "$name")"
}
