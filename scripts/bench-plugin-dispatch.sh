#!/usr/bin/env bash
# bench-plugin-dispatch.sh — measure `shy <plugin>` dispatch overhead.
#
# Runs the hello-world reference plugin N times against an isolated
# SHY_HOME, reports the median wall-clock duration in milliseconds, and
# fails (exit 1) when the median exceeds the threshold.
#
# Inputs (env):
#   SHY_BIN         — path to shy binary (default: ./dist/shy)
#   SHY_BENCH_N     — sample count (default: 100)
#   SHY_BENCH_MAX   — fail threshold in milliseconds (default: 100)

set -euo pipefail

bin="${SHY_BIN:-./dist/shy}"
n="${SHY_BENCH_N:-100}"
max_ms="${SHY_BENCH_MAX:-100}"

if [[ ! -x "$bin" ]]; then
    echo "bench: $bin is not executable; build first" >&2
    exit 2
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

export SHY_HOME="$tmp/home"
export SHY_TEST_BASHRC="$tmp/.bashrc"
mkdir -p "$SHY_HOME"

"$bin" init >/dev/null
"$bin" install ./examples/plugins/hello-world >/dev/null

times_file="$tmp/times"
: >"$times_file"

for ((i = 0; i < n; i++)); do
    # %3N emits nanoseconds; cut to milliseconds.
    start_ns=$(date +%s%N)
    "$bin" hello-world >/dev/null
    end_ns=$(date +%s%N)
    echo $(( (end_ns - start_ns) / 1000000 )) >>"$times_file"
done

sorted=$(sort -n "$times_file")
median=$(echo "$sorted" | awk -v n="$n" 'BEGIN{i=int((n+1)/2)} NR==i {print; exit}')
p95=$(echo "$sorted" | awk -v n="$n" 'BEGIN{i=int(n*0.95)} NR==i {print; exit}')
worst=$(echo "$sorted" | tail -1)

printf 'bench-plugin-dispatch: n=%d median=%dms p95=%dms max=%dms threshold=%dms\n' \
    "$n" "$median" "$p95" "$worst" "$max_ms"

if (( median > max_ms )); then
    echo "bench: median ${median}ms exceeds threshold ${max_ms}ms" >&2
    exit 1
fi
