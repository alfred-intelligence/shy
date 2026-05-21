# Code skeletons

Pre-implementation files for the post-flight metrics layer. Copy into the repo
at the indicated paths once `cli/` exists.

## Files

| Path in this folder | Path in repo | Purpose |
|---|---|---|
| `cli/internal/metrics/metrics.go` | `cli/internal/metrics/metrics.go` | Opt-in runtime timing (Layer 2) |
| `cli/internal/metrics/metrics_test.go` | `cli/internal/metrics/metrics_test.go` | Unit tests + disabled-path benchmark |
| `cli/internal/manifest/manifest_bench_test.go` | `cli/internal/manifest/manifest_bench_test.go` | Manifest-parse benchmarks |
| `.github/workflows/bench.yml` | `.github/workflows/bench.yml` | CI benchmark workflow |

## Preconditions

- `cli/internal/manifest/manifest.go` with `func Parse([]byte) (*Manifest, error)`
  must exist before `manifest_bench_test.go` will compile. See Step 10 of
  `../03-short-horizon.md`.
- `bench.yml` references `benchmark-action/github-action-benchmark@v1`. Verify
  the action is still maintained before merging and pin to a commit SHA rather
  than the `v1` tag for supply-chain hygiene.
- Go 1.17+ is required (uses `testing.B.Setenv`).

## Using the metrics module

In production code:

```go
import "github.com/alfred-intelligence/shy/cli/internal/metrics"

func runInstall(args []string) error {
    defer metrics.Start("install").End()
    // existing install logic
    return nil
}
```

Runtime activation:

```bash
SHY_METRICS=1 shy install @alice/some-script
cat ~/.shy/metrics.jsonl
# {"name":"install","start_nanos":1716123456789012345,"duration_ns":127483921}
```

Default OFF. The disabled path is one env read plus one branch.

## Privacy

`~/.shy/metrics.jsonl` records only:

- Span name (developer-defined, e.g. `install`, `plugin-dispatch`)
- Start timestamp (nanoseconds since epoch)
- Duration (nanoseconds)

It does NOT record paths, arguments, environment variables, or user content.
The file is safe to share for debugging; inspect before sharing.

## What's missing

These exist now because the underlying code from the design exists in concept.
Add when the relevant code lands:

- **Plugin-dispatch benchmark** — `cli/internal/dispatch/dispatch_bench_test.go`
  when Phase 5 of `../02-long-horizon.md` is implemented.
- **Sandbox-overhead benchmark** — v2 scope; add when bubblewrap/firejail
  integration lands.
- **`hyperfine` invocation scripts** for `next`/`before`/`after` worktree
  comparison — pure shell commands, no Go code. Document the command line in
  `../05-engineering-handbook.md` when v1 is far enough along to benchmark
  end-to-end.

## Mapping to the six real hybrids

| Hybrid | Skeleton coverage |
|---|---|
| #1 Sourced + dispatched | Partial: dispatch benchmark missing until Phase 5 |
| #2 v1 CLI + v2 workspace | Not covered; v2 scope |
| #3 Path-based branch protection | N/A (no code) |
| #4 Type-based Dependabot | N/A (no code) |
| #5 Two completion conventions | Not covered; add `completion_bench_test.go` when Phase 5 lands |
| #6 Two-layer sandbox | Not covered; v2 scope |

The current skeletons exercise the metrics infrastructure itself plus the
manifest-parse hot path. Everything else needs the corresponding code to
exist before benchmarks can be written.
