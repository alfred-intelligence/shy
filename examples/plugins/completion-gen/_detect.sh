#!/usr/bin/env bash
# _detect.sh — tier 1: native/self-declared completion generators.
#
# Tried FIRST, before any --help parsing, because a tool's own generator
# is always more accurate than a text-scraped approximation. This tier is
# a strict superset of shy's existing `shy completion add` probe list
# (`completion bash`, `completion --shell bash`, `bash-completion`) plus
# five more conventions actually in use across the cobra/clap/click/
# argcomplete ecosystems. Sourced by entry.sh.
#
# shellcheck disable=SC2034  # CG_METHOD/CG_NATIVE_OUT are read by entry.sh
# after this file's functions return (sourced separately).

# _cg_valid_native_output OUTPUT
# Rejects a non-zero exit (CG_LAST_RC, set by the _cg_run call immediately
# preceding — real generators exit 0 on success; an unrecognised-flag
# error path is the common false-positive source and usually exits non-
# zero), rejects empty/trivial output, and rejects output that is byte-
# identical to the tool's own --help/-h text (CG_HELP_REF) — the tell
# that a probe just triggered a generic help dump, not a real completion
# script.
_cg_valid_native_output() {
    local out="$1"
    [[ "${CG_LAST_RC:-1}" -eq 0 ]] || return 1
    [[ -n "$out" ]] || return 1
    [[ "${#out}" -ge 20 ]] || return 1
    if [[ -n "${CG_HELP_REF:-}" && "$out" == "$CG_HELP_REF" ]]; then
        return 1
    fi
    return 0
}

# _cg_detect_native TOOL SHELL
# On success: sets CG_NATIVE_OUT (the winning script) and CG_METHOD,
# returns 0. On failure: returns 1, CG_NATIVE_OUT undefined. Every probe
# is shell-parametrised where the convention supports it (cobra, clap,
# and click all generate per-shell output; the shell argument is
# threaded through rather than hardcoded to bash).
#
# Deliberately does NOT print to stdout / return via command
# substitution: the caller needs BOTH the script content and the
# CG_METHOD side-channel, and `script="$(_cg_detect_native ...)"` would
# run this function in a subshell — any global variable this function
# sets (CG_METHOD) would then vanish the instant that subshell exits,
# while only stdout survived. Verified the hard way against a real
# `kubectl completion bash` success that silently lost its CG_METHOD.
# A plain global-out-var, called without $(...), has no such trap.
_cg_detect_native() {
    local tool="$1" shell="$2"

    _cg_run "$CG_TIMEOUT" "$tool" completion "$shell"
    if _cg_valid_native_output "$CG_LAST_OUT"; then
        CG_METHOD="native: ${tool} completion ${shell}"
        CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
    fi

    _cg_run "$CG_TIMEOUT" "$tool" completion --shell "$shell"
    if _cg_valid_native_output "$CG_LAST_OUT"; then
        CG_METHOD="native: ${tool} completion --shell ${shell}"
        CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
    fi

    _cg_run "$CG_TIMEOUT" "$tool" completion -s "$shell"
    if _cg_valid_native_output "$CG_LAST_OUT"; then
        CG_METHOD="native: ${tool} completion -s ${shell}"
        CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
    fi

    _cg_run "$CG_TIMEOUT" "$tool" completions "$shell"
    if _cg_valid_native_output "$CG_LAST_OUT"; then
        CG_METHOD="native: ${tool} completions ${shell}"
        CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
    fi

    _cg_run "$CG_TIMEOUT" "$tool" --completion "$shell"
    if _cg_valid_native_output "$CG_LAST_OUT"; then
        CG_METHOD="native: ${tool} --completion ${shell}"
        CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
    fi

    if [[ "$shell" == "bash" ]]; then
        _cg_run "$CG_TIMEOUT" "$tool" bash-completion
        if _cg_valid_native_output "$CG_LAST_OUT"; then
            CG_METHOD="native: ${tool} bash-completion"
            CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
        fi
    fi

    # Click (Python) convention: activated via an env var derived from the
    # program name, not a subcommand — e.g. "flask" -> _FLASK_COMPLETE.
    # Click supports bash_source/zsh_source/fish_source, so this is fully
    # shell-parametrised too.
    local envname
    envname="_$(printf '%s' "$tool" | tr '[:lower:]-' '[:upper:]_')_COMPLETE"
    _cg_run "$CG_TIMEOUT" env "${envname}=${shell}_source" "$tool"
    if _cg_valid_native_output "$CG_LAST_OUT"; then
        CG_METHOD="native: click-env ${envname}=${shell}_source"
        CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
    fi

    # argcomplete (Python), only if its registration helper is present AND
    # the target actually looks like a Python entry point. Verified the
    # hard way: register-python-argcomplete <name> exits 0 and prints
    # generic (non-functional) boilerplate for ANY name at all, including
    # compiled binaries with no relation to Python (e.g. `git`) — it
    # never validates its argument. Trusting its mere success would be a
    # guaranteed false positive for most non-Python tools, so this probe
    # requires the corroborating shebang check below before it runs.
    if command -v register-python-argcomplete >/dev/null 2>&1 && _cg_looks_like_python "$tool"; then
        if [[ "$shell" == "bash" ]]; then
            _cg_run "$CG_TIMEOUT" register-python-argcomplete "$tool"
        else
            _cg_run "$CG_TIMEOUT" register-python-argcomplete --shell "$shell" "$tool"
        fi
        if _cg_valid_native_output "$CG_LAST_OUT"; then
            CG_METHOD="native: register-python-argcomplete (--shell ${shell}) ${tool}"
            CG_NATIVE_OUT="$CG_LAST_OUT"; return 0
        fi
    fi

    return 1
}
