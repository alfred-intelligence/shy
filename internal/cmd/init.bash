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

# Source entry.sh from every script under <base>/installed/%<ns>/<name>/.
# The % prefix marks script directories; @ (plugins) and # (collections) are
# exec'd or unused at shell start.
_shy_source_installed_scripts() {
    local base="$1"
    local installed="$base/installed"
    [[ -d "$installed" ]] || return 0
    local ns_dir item_dir entry
    for ns_dir in "$installed"/'%'*/; do
        [[ -d "$ns_dir" ]] || continue
        for item_dir in "$ns_dir"*/; do
            [[ -d "$item_dir" ]] || continue
            entry="$item_dir/entry.sh"
            [[ -f "$entry" ]] || continue
            # shellcheck source=/dev/null
            source "$entry" 2>/dev/null || printf 'shy: failed to source %s\n' "$entry" >&2
        done
    done
}

# User layer — primary source of truth.
_shy_source_installed_scripts "$SHY_HOME"
_shy_source_flat "$SHY_HOME/helpers/aliases"
_shy_source_flat "$SHY_HOME/helpers/completions"

# Overrides — re-define items from the user layer; last-source-wins.
_shy_source_installed_scripts "$SHY_HOME/overrides.d"
_shy_source_flat "$SHY_HOME/overrides.d/helpers/aliases"
_shy_source_flat "$SHY_HOME/overrides.d/helpers/completions"

unset -f _shy_source_flat _shy_source_installed_scripts

# Re-source init.bash without opening a new shell.
alias shy-reload='source "$SHY_HOME/init.bash"'
