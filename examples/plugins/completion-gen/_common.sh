#!/usr/bin/env bash
# _common.sh — shared low-level helpers for completion-gen. Sourced by
# entry.sh; never invoked standalone (no shebang execution path — it is
# only ever `source`d, so it declares functions and returns).
#
# shellcheck disable=SC2034  # CG_LAST_RC/CG_LAST_OUT are read by callers
# in _detect.sh/_introspect.sh (sourced separately) right after each
# _cg_run/_cg_run_combined call — shellcheck's per-file analysis can't see
# across the `source` boundary.

# Resolve a timeout binary once. GNU coreutils' `timeout` ships on every
# Linux target shy cares about; macOS/BSD userland does not, but Homebrew's
# coreutils provides it as `gtimeout` (no --default-names). Falling back to
# "no timeout available" rather than failing outright keeps the plugin
# usable on a bare macOS box, at the cost of losing the hang-protection —
# flagged once via CG_WARNINGS, not silently.
_cg_resolve_timeout_bin() {
    if command -v timeout >/dev/null 2>&1; then
        printf 'timeout'
    elif command -v gtimeout >/dev/null 2>&1; then
        printf 'gtimeout'
    fi
}
CG_TIMEOUT_BIN="$(_cg_resolve_timeout_bin)"
CG_WARNED_NO_TIMEOUT=0

_cg_warn_no_timeout() {
    if [[ "$CG_WARNED_NO_TIMEOUT" -eq 0 ]]; then
        CG_WARNINGS+=("no 'timeout'/'gtimeout' binary found on PATH — probes ran WITHOUT a time limit; install GNU coreutils to restore the hang-protection safety net")
        CG_WARNED_NO_TIMEOUT=1
    fi
}

# _cg_run_impl COMBINE TIMEOUT_SECS CMD [ARGS...]
# COMBINE=1 merges stderr into the captured output (needed for --help text:
# many POSIX getopt-style tools print usage to stderr, not stdout).
# COMBINE=0 captures stdout only (needed for native completion-generator
# output, where stderr noise must never leak into the emitted script).
# Stdin is always /dev/null — an introspection probe must never block
# waiting on input. Populates CG_LAST_OUT / CG_LAST_RC.
_cg_run_impl() {
    local combine="$1" t="$2"; shift 2
    local out
    if [[ -n "$CG_TIMEOUT_BIN" ]]; then
        if [[ "$combine" -eq 1 ]]; then
            out="$("$CG_TIMEOUT_BIN" "$t" "$@" </dev/null 2>&1)"
        else
            out="$("$CG_TIMEOUT_BIN" "$t" "$@" </dev/null 2>/dev/null)"
        fi
    else
        _cg_warn_no_timeout
        if [[ "$combine" -eq 1 ]]; then
            out="$("$@" </dev/null 2>&1)"
        else
            out="$("$@" </dev/null 2>/dev/null)"
        fi
    fi
    CG_LAST_RC=$?
    CG_LAST_OUT="$out"
}

_cg_run() { _cg_run_impl 0 "$@"; }
_cg_run_combined() { _cg_run_impl 1 "$@"; }

# _cg_tool_exists TOOL — resolves TOOL on PATH, or as an executable file
# path (relative or absolute), so `shy completion-gen ./my-script` works
# the same as `shy completion-gen gh`.
_cg_tool_exists() {
    local tool="$1"
    command -v -- "$tool" >/dev/null 2>&1 && return 0
    [[ -x "$tool" ]] && return 0
    return 1
}

# _cg_looks_like_python TOOL — true only if TOOL resolves to a readable
# text file whose shebang names a python interpreter. Gates the
# register-python-argcomplete probe in _detect.sh: that helper exits 0
# and prints boilerplate for ANY name regardless of whether it is
# actually a Python entry point, so its own success is not evidence of
# anything — this shebang check is the real signal.
#
# Checks the first 2 bytes ("#!") BEFORE reading a whole line: most
# resolved tools are compiled binaries (ELF), and `head -n 1` on one
# reads until the first newline byte wherever that happens to land —
# often after a long run of binary data — which bash's command
# substitution flags with a noisy "ignored null byte" warning even
# though the result is simply discarded. The 2-byte pre-check is cheap
# and short-circuits before that ever happens.
_cg_looks_like_python() {
    local tool="$1" path magic first_line
    path="$(command -v -- "$tool" 2>/dev/null)" || path="$tool"
    [[ -r "$path" ]] || return 1
    magic="$(head -c 2 -- "$path" 2>/dev/null)"
    [[ "$magic" == "#!" ]] || return 1
    first_line="$(head -n 1 -- "$path" 2>/dev/null)"
    [[ "$first_line" == "#!"*python* ]]
}

# _cg_capture_help TOOL [SUBPATH...]
# Tries "--help" then "-h", appended after SUBPATH. Deliberately never
# invokes the tool bare (no help flag at all) — see README "safety
# envelope": a bare invocation of an arbitrary binary can have real side
# effects (open an editor, attempt a network connection, wait on a
# prompt); an explicit --help/-h request is the one thing every CLI
# convention treats as read-only by contract. Combines stdout+stderr
# since many getopt-style tools print usage to stderr.
#
# Forces LC_ALL=C / LANGUAGE=en on the probe: the tier-2 parser's section-
# header and flag regexes are English-pattern-based (as is virtually every
# practical --help scraper), so an operator whose locale localises --help
# text (e.g. git's own translations) would otherwise silently get zero
# subcommands/flags out of a tool that actually has plenty. Tier 1 (native
# passthrough) does not force locale — it never parses the output, only
# relays it.
_cg_capture_help() {
    local tool="$1"; shift
    local -a subpath=("$@")
    _cg_run_combined "$CG_TIMEOUT" env LC_ALL=C LANGUAGE=en "$tool" "${subpath[@]}" --help
    [[ -n "$CG_LAST_OUT" ]] && return 0
    _cg_run_combined "$CG_TIMEOUT" env LC_ALL=C LANGUAGE=en "$tool" "${subpath[@]}" -h
    [[ -n "$CG_LAST_OUT" ]] && return 0
    return 1
}

# _cg_bash_ident STR — sanitizes STR into a valid bash function-name
# fragment (tool names can contain characters bash identifiers can't,
# e.g. "docker-compose" is fine but a path like "./tool.sh" is not).
_cg_bash_ident() {
    printf '%s' "$1" | tr -c 'A-Za-z0-9_' '_'
}

# _cg_squote STR — single-quote STR for safe embedding in generated shell
# code, escaping embedded single quotes. Same technique shy's own
# installer uses (internal/install/install.go shellQuote).
_cg_squote() {
    printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

# _cg_join SEP ARG... — joins ARGs with SEP (bash's "$*" only honours the
# first character of IFS, which breaks multi-character separators like
# "; and ").
_cg_join() {
    local sep="$1"; shift
    local out="" first=1 a
    for a in "$@"; do
        if [[ "$first" -eq 1 ]]; then out="$a"; first=0; else out+="${sep}${a}"; fi
    done
    printf '%s' "$out"
}

_cg_json_escape() {
    local s="$1"
    s="${s//\\/\\\\}"
    s="${s//\"/\\\"}"
    printf '%s' "$s"
}
