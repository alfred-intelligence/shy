#!/usr/bin/env bash
# _introspect.sh — tier 2: --help/-h text parsing, recursive subcommand
# walk. Only used when tier 1 (_detect.sh) found no native generator.
# Sourced by entry.sh.

# _cg_extract_subcommands HELP_TEXT
# Prints one subcommand candidate per line. Looks for a header line whose
# trimmed, lower-cased form ends in "commands" or "subcommands", optionally
# followed by a parenthetical qualifier and/or a colon — this catches
# cobra's "Available Commands:", npm/docker's "Commands:"/"Management
# Commands:", kubectl's "Basic Commands (Beginner):", and plain
# "Subcommands:" uniformly. Once inside such a section, consumes indented
# lines until a blank line or a non-indented line (a new section), taking
# the first whitespace-delimited token of each (minus a trailing alias
# list) as the subcommand name.
_cg_extract_subcommands() {
    local text="$1"
    printf '%s\n' "$text" | awk '
        BEGIN { insec = 0 }
        {
            line = $0
            trimmed = line
            gsub(/^[ \t]+|[ \t]+$/, "", trimmed)
            lt = tolower(trimmed)
            if (lt ~ /(^|[ \t])(commands|subcommands)([ \t]*\([^)]*\))?[ \t]*:?[ \t]*$/) {
                insec = 1
                next
            }
            if (insec) {
                if (trimmed == "") { insec = 0; next }
                if (line !~ /^[ \t]/) { insec = 0; next }
                tok = trimmed
                sub(/[ \t].*/, "", tok)
                sub(/,.*/, "", tok)
                if (tok !~ /^-/ && tok != "") print tok
            }
        }
    '
}

# _cg_extract_flags HELP_TEXT
# Scans the WHOLE text (no header required — many POSIX getopt-style
# tools list flags with no "Options:" section at all) for flag-shaped
# tokens: "--long-name" and "-x" in flag position (start of line or
# preceded by whitespace). Two-stage grep: the first pass matches with
# the required leading-boundary context, the second strips that context
# back off, which is simpler and more portable than a single regex with
# a non-capturing lookaround (POSIX ERE has none).
_cg_extract_flags() {
    local text="$1"
    {
        printf '%s\n' "$text" \
            | grep -oE '(^|[[:space:]])--[A-Za-z][A-Za-z0-9-]*' \
            | grep -oE -- '--[A-Za-z][A-Za-z0-9-]*'
        printf '%s\n' "$text" \
            | grep -oE '(^|[[:space:]])-[A-Za-z]([^A-Za-z0-9=_-]|$)' \
            | grep -oE -- '-[A-Za-z]'
    } 2>/dev/null | sort -u
}

# CG_TREE accumulates one "path<US>subs_csv<US>flags_csv" record per
# visited node (<US> = ASCII Unit Separator, 0x1F). path is the
# space-joined subcommand path from the root ("" for the root itself).
# Populated by _cg_walk; consumed by _emit.sh via `IFS=$'\x1f' read`.
#
# Deliberately NOT a tab: bash's `read` treats IFS characters drawn from
# its "whitespace class" (space/tab/newline) as a single collapsing
# delimiter and strips leading occurrences — exactly the bug this
# comment is here to stop someone from reintroducing. A root node with
# an empty path AND no subcommands (real case: `git`, which has no
# "Commands:" header) produces a record with two leading empty fields;
# `IFS=$'\t' read` silently collapsed those and shifted the flags list
# into the path variable. \x1f is not whitespace, so empty fields survive.
CG_TREE=()

# _cg_walk TOOL DEPTH [PATH_TOKEN...]
# Depth-first. Stops recursing past CG_MAX_DEPTH or once CG_CALLS_LEFT
# (the total-invocation budget) reaches zero — both bound runtime against
# tools with enormous command trees (docker, gh, aws, kubectl, ...).
_cg_walk() {
    local tool="$1" depth="$2"; shift 2
    local -a path=("$@")

    if (( CG_CALLS_LEFT <= 0 )); then
        CG_WARNINGS+=("invocation budget exhausted (--max-calls=${CG_MAX_CALLS}) before the walk finished; the tree may be incomplete")
        return
    fi

    local help_text
    if [[ "${#path[@]}" -eq 0 ]]; then
        help_text="$CG_HELP_REF"
    else
        CG_CALLS_LEFT=$((CG_CALLS_LEFT - 1))
        if ! _cg_capture_help "$tool" "${path[@]}"; then
            CG_WARNINGS+=("no --help/-h output for '${tool} ${path[*]}' — node skipped")
            return
        fi
        help_text="$CG_LAST_OUT"
        # A very common real-world pattern: a tool with no dedicated help
        # for an unrecognised/unhandled subcommand just reprints its ROOT
        # usage instead of erroring. Left unguarded, that reprinted root
        # text would parse into the SAME subcommand names all over again,
        # and the walk would recurse into it as if "path" genuinely had
        # its own identical children — an exponential, meaningless tree
        # (verified against a synthetic fixture: 3 root subcommands with
        # this fallback behaviour blew up to 33 nodes at depth 3). If a
        # non-root node's help text is byte-identical to the root's, treat
        # it exactly like "no help available": record nothing, don't
        # descend.
        if [[ "$help_text" == "$CG_HELP_REF" ]]; then
            CG_WARNINGS+=("'${tool} ${path[*]} --help' just reprinted the root help text (no distinct subcommand help) — node skipped, not descended")
            return
        fi
    fi

    local subs flags subs_csv flags_csv
    subs="$(_cg_extract_subcommands "$help_text")"
    flags="$(_cg_extract_flags "$help_text")"
    subs_csv="$(printf '%s' "$subs" | paste -sd, - 2>/dev/null || true)"
    flags_csv="$(printf '%s' "$flags" | paste -sd, - 2>/dev/null || true)"

    CG_TREE+=("${path[*]}"$'\x1f'"${subs_csv}"$'\x1f'"${flags_csv}")

    if (( depth >= CG_MAX_DEPTH )); then
        return
    fi

    local sub
    while IFS= read -r sub; do
        [[ -n "$sub" ]] || continue
        (( CG_CALLS_LEFT <= 0 )) && break
        _cg_walk "$tool" "$((depth + 1))" "${path[@]}" "$sub"
    done <<< "$subs"
}
