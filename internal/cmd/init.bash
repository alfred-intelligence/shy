# ~/.shy/init.bash
# Sourced from ~/.bashrc to activate installed shy items.
# Runtime layer follows filesystem structure; manifests are never read here.

export SHY_HOME="${SHY_HOME:-$HOME/.shy}"

# Add shy's own binary to PATH if not already present.
case ":$PATH:" in
    *":$SHY_HOME/bin:"*) ;;
    *) export PATH="$SHY_HOME/bin:$PATH" ;;
esac

# Source files in a flat directory; skip _-prefixed; tolerate per-file errors.
_shy_source_flat() {
    local dir="$1"
    [[ -d "$dir" ]] || return 0
    local f
    for f in "$dir"/*; do
        [[ -f "$f" ]] || continue
        [[ "$(basename "$f")" == _* ]] && continue
        # shellcheck source=/dev/null
        source "$f" 2>/dev/null || printf 'shy: failed to source %s\n' "$f" >&2
    done
}

# Source files in a namespaced directory tree (namespace/name/*.sh).
_shy_source_namespaced() {
    local dir="$1"
    [[ -d "$dir" ]] || return 0
    local ns item sh
    for ns in "$dir"/*/; do
        [[ -d "$ns" ]] || continue
        for item in "$ns"*/; do
            [[ -d "$item" ]] || continue
            for sh in "$item"*.sh; do
                [[ -f "$sh" ]] || continue
                [[ "$(basename "$sh")" == _* ]] && continue
                # shellcheck source=/dev/null
                source "$sh" 2>/dev/null || printf 'shy: failed to source %s\n' "$sh" >&2
            done
        done
    done
}

# User layer — primary source of truth.
_shy_source_namespaced "$SHY_HOME/scripts"
_shy_source_flat "$SHY_HOME/aliases"
_shy_source_flat "$SHY_HOME/completions"

# Overrides — re-define items from the user layer; last-source-wins.
_shy_source_namespaced "$SHY_HOME/overrides.d/scripts"
_shy_source_flat "$SHY_HOME/overrides.d/aliases"
_shy_source_flat "$SHY_HOME/overrides.d/completions"

unset -f _shy_source_flat _shy_source_namespaced

# Re-source init.bash without opening a new shell.
alias shy-reload='source "$SHY_HOME/init.bash"'
