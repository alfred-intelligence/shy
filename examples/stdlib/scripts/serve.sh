#!/usr/bin/env bash
# serve — start an HTTP server in the current directory.

serve() {
    local port="${1:-8000}"
    if ! command -v python3 >/dev/null 2>&1; then
        echo "serve: python3 not found on PATH" >&2
        return 1
    fi
    echo "serving $(pwd) on http://localhost:${port}"
    python3 -m http.server "$port"
}
