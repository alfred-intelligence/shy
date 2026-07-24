#!/usr/bin/env bash
# completion-gen — shy plugin: generate shell completions for an
# arbitrary binary by introspecting what it supports.
#
# Usage:
#   shy completion-gen <tool> [--shell bash|zsh|fish] [--install] [--force]
#                       [--depth N] [--timeout SECS] [--max-calls N] [--json]
#   shy completion-gen --list-methods
#
# See README.md for the safety envelope, the method list, and the
# core-graduation note (this plugin is a candidate replacement for
# `shy completion add` — manifest.toml [completion-gen-meta]).
set -uo pipefail
# NOT `set -e`: introspection deliberately tolerates per-probe failure — a
# disqualified native-generator guess, a subcommand whose --help timed
# out, a tool with no "Commands:" section — each is data (one more
# disqualified strategy, one more skipped node), never a reason to abort
# the whole run. Every command whose failure IS fatal to the run checks
# its own exit status explicitly below.

_cg_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=/dev/null
source "$_cg_dir/_common.sh"
# shellcheck source=/dev/null
source "$_cg_dir/_detect.sh"
# shellcheck source=/dev/null
source "$_cg_dir/_introspect.sh"
# shellcheck source=/dev/null
source "$_cg_dir/_emit.sh"

# ---- __complete: shy's tab-completion hook for THIS plugin's own args ----
# (Not to be confused with the completions this plugin GENERATES for
# other tools — this is completion-gen completing its own invocation.)
if [[ "${1-}" == "__complete" ]]; then
    shift
    if [[ $# -eq 0 ]]; then
        compgen -A command -- "" 2>/dev/null | sort -u
        exit 0
    fi
    case "${*: -1}" in
        --shell) printf 'bash\nzsh\nfish\n' ;;
        *) printf -- '--shell\n--install\n--force\n--depth\n--timeout\n--max-calls\n--json\n--list-methods\n--help\n' ;;
    esac
    exit 0
fi

_cg_usage() {
    cat <<'EOF'
completion-gen — generate shell completions for an arbitrary binary via
introspection: try the tool's own completion generator first (a wider
net than shy's built-in `completion add`), fall back to parsing --help/-h
text when it has none.

Usage:
  shy completion-gen <tool> [options]

Options:
  --shell bash|zsh|fish   Target shell (default: bash)
  --install               Write to $SHY_HOME/helpers/completions/<tool>
                          instead of stdout. Honors SHY_ON_CONFLICT with
                          the SAME four values as shy's own installer
                          (fail|prefer-new|prefer-existing|skip; default
                          fail).
  --force                 With --install: overwrite regardless of policy
  --depth N               Max subcommand recursion depth (default: 2)
  --timeout SECS          Per-probe timeout in seconds (default: 3)
  --max-calls N           Max tool invocations during introspection
                          (default: 40) — bounds runtime on huge command
                          trees (docker, gh, aws, kubectl, ...)
  --json                  Print diagnostics instead of the script
  --list-methods          List the native-generator conventions tried
  -h, --help              This text

Without --install, the script prints to stdout — exactly the shape shy's
own manifest hook expects (internal/install/install.go's `[[completions]]
generate = "..."`), so `generate = "shy completion-gen <tool>"` slots into
an EXISTING manifest today with no shy-core change required.

Safety envelope: the target tool is only ever invoked with an explicit
--help/-h flag, or via a documented completion-generator convention (same
risk class as shy's own `completion add`) — never bare, always under a
timeout, always with stdin closed. See README.md.
EOF
}

_cg_list_methods() {
    cat <<'EOF'
Native-generator conventions tried, in order, before falling back to
--help-parsing (each tried for the requested --shell):

  1. <tool> completion <shell>
  2. <tool> completion --shell <shell>
  3. <tool> completion -s <shell>
  4. <tool> completions <shell>
  5. <tool> --completion <shell>
  6. <tool> bash-completion                       (bash only; legacy)
  7. _<TOOL>_COMPLETE=<shell>_source <tool>        (click convention)
  8. register-python-argcomplete [--shell <shell>] <tool>  (if on PATH)

shy's own `completion add` tries only #1 and #2 (hardcoded to bash) and
#6 — this plugin is a strict superset at tier 1, before introspection
(--help parsing) even starts.
EOF
}

CG_TIMEOUT=3
CG_MAX_DEPTH=2
CG_MAX_CALLS=40
CG_SHELL=bash
CG_INSTALL=0
CG_FORCE=0
CG_JSON=0
# shellcheck disable=SC2034  # read by _introspect.sh / _emit.sh (sourced above)
CG_WARNINGS=()
# shellcheck disable=SC2034  # read by _introspect.sh / _emit.sh (sourced above)
CG_TREE=()
tool=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --list-methods) _cg_list_methods; exit 0 ;;
        -h|--help) _cg_usage; exit 0 ;;
        --shell) CG_SHELL="${2:?--shell requires bash|zsh|fish}"; shift 2 ;;
        --install) CG_INSTALL=1; shift ;;
        --force) CG_FORCE=1; shift ;;
        --json) CG_JSON=1; shift ;;
        --depth) CG_MAX_DEPTH="${2:?--depth requires a number}"; shift 2 ;;
        --timeout) CG_TIMEOUT="${2:?--timeout requires seconds}"; shift 2 ;;
        --max-calls) CG_MAX_CALLS="${2:?--max-calls requires a number}"; shift 2 ;;
        --)
            shift
            if [[ -n "${1-}" ]]; then tool="$1"; shift; fi
            ;;
        -*)
            echo "completion-gen: unknown flag: $1 (see --help)" >&2
            exit 2
            ;;
        *)
            if [[ -z "$tool" ]]; then
                tool="$1"
            else
                echo "completion-gen: unexpected extra argument: $1" >&2
                exit 2
            fi
            shift
            ;;
    esac
done

if [[ -z "$tool" ]]; then
    _cg_usage >&2
    exit 2
fi

case "$CG_SHELL" in
    bash|zsh|fish) ;;
    *) echo "completion-gen: --shell must be bash, zsh, or fish (got: $CG_SHELL)" >&2; exit 2 ;;
esac

if ! [[ "$CG_MAX_DEPTH" =~ ^[0-9]+$ ]] || ! [[ "$CG_MAX_CALLS" =~ ^[0-9]+$ ]] || ! [[ "$CG_TIMEOUT" =~ ^[0-9]+$ ]]; then
    echo "completion-gen: --depth, --timeout, and --max-calls must be non-negative integers" >&2
    exit 2
fi

if ! _cg_tool_exists "$tool"; then
    echo "completion-gen: '$tool' not found on PATH (and not an executable file path)" >&2
    exit 1
fi

CG_CALLS_LEFT="$CG_MAX_CALLS"

# Capture root --help/-h once; reused both as the tier-1 false-positive
# reference (_cg_valid_native_output) and as the tier-2 parse input, so a
# tool that only answers one of --help/-h is only ever probed once for it.
CG_HELP_REF=""
if _cg_capture_help "$tool"; then
    CG_HELP_REF="$CG_LAST_OUT"
fi

CG_METHOD=""
CG_NATIVE_OUT=""
script=""
if _cg_detect_native "$tool" "$CG_SHELL"; then
    script="$CG_NATIVE_OUT"
fi

if [[ -z "$script" ]]; then
    if [[ -z "$CG_HELP_REF" ]]; then
        echo "completion-gen: '$tool': no native completion generator worked, and --help/-h produced no output — nothing to introspect" >&2
        exit 1
    fi
    _cg_walk "$tool" 0
    CG_METHOD="introspected(--help-parsed; depth=${CG_MAX_DEPTH}; calls=$((CG_MAX_CALLS - CG_CALLS_LEFT)))"
    case "$CG_SHELL" in
        bash) script="$(_cg_emit_bash "$tool")" ;;
        zsh)  script="$(_cg_emit_zsh "$tool")" ;;
        fish) script="$(_cg_emit_fish "$tool")" ;;
    esac
fi

if [[ "$CG_JSON" -eq 1 ]]; then
    _cg_emit_json "$tool"
    exit 0
fi

if [[ "$CG_INSTALL" -eq 1 ]]; then
    _cg_conflict_policy() {
        case "$(tr '[:upper:]' '[:lower:]' <<< "${SHY_ON_CONFLICT:-}")" in
            prefer-existing|prefer-new|skip) tr '[:upper:]' '[:lower:]' <<< "${SHY_ON_CONFLICT}" ;;
            *) printf 'fail' ;;
        esac
    }

    home="${SHY_HOME:-$HOME/.shy}"
    dest="$home/helpers/completions/$tool"
    mkdir -p "$(dirname -- "$dest")" || { echo "completion-gen: mkdir failed for $dest" >&2; exit 1; }

    if [[ -e "$dest" ]]; then
        existing="$(cat -- "$dest" 2>/dev/null || true)"
        if [[ "$existing" == "$script" ]]; then
            echo "completion-gen: $dest already up to date" >&2
            exit 0
        fi
        policy="$(_cg_conflict_policy)"
        if [[ "$CG_FORCE" -ne 1 ]]; then
            case "$policy" in
                prefer-existing|skip)
                    echo "completion-gen: $dest exists and differs; SHY_ON_CONFLICT=$policy — keeping existing" >&2
                    exit 0
                    ;;
                prefer-new) : ;;
                *)
                    echo "completion-gen: conflict at $dest (existing file would be overwritten). Pass --force, or set SHY_ON_CONFLICT=prefer-new|skip|prefer-existing (same semantics as shy's own installer)." >&2
                    exit 1
                    ;;
            esac
        fi
    fi

    printf '%s\n' "$script" > "$dest" || { echo "completion-gen: write failed for $dest" >&2; exit 1; }
    echo "completion-gen: wrote $dest ($(printf '%s' "$script" | wc -c) bytes, method: ${CG_METHOD})" >&2
    exit 0
fi

printf '%s\n' "$script"
