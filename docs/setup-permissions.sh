#!/usr/bin/env bash
# Configure branch and tag protection for the shy repository.
#
# Generates JSON config files under .github/protection/ and
# optionally applies them via the gh CLI. Idempotent — safe to
# rerun.
#
# Run from the operator's local clone of the shy repository.
# Native execution; not invoked through shy itself.

set -euo pipefail

# Clear screen before output to avoid mixing with prior session
clear

# Repository target (override with --repo or SHY_REPO env var)
REPO="${SHY_REPO:-alfred-intelligence/shy}"

# Behavior flags
DRY_RUN=false
GENERATE_ONLY=false
APPLY_AUTO=false

# Output directory for generated JSON configs
OUT_DIR=".github/protection"

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Generates and applies branch + tag protection for $REPO.
Idempotent: safe to rerun. Existing protections are overwritten.

Options:
  --dry-run         Show planned changes; do not apply
  --generate-only   Write JSON files but skip gh api calls
  --apply           Skip the confirmation prompt; apply immediately
  --repo OWNER/NAME Override repo target (or set SHY_REPO env var)
  --help            Show this help and exit

Requirements:
  - gh CLI authenticated (gh auth status)
  - Write permission on the target repository
  - jq installed (used for tag protection lookup)
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)       DRY_RUN=true; shift ;;
    --generate-only) GENERATE_ONLY=true; shift ;;
    --apply)         APPLY_AUTO=true; shift ;;
    --repo)          REPO="$2"; shift 2 ;;
    --help)          usage; exit 0 ;;
    *)               echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

# Sanity checks on environment
if ! command -v gh >/dev/null 2>&1; then
  echo "Error: gh CLI not found in PATH" >&2
  echo "Install: https://cli.github.com/" >&2
  exit 1
fi

if [[ "$GENERATE_ONLY" == false ]] && ! command -v jq >/dev/null 2>&1; then
  echo "Error: jq not found in PATH (needed for tag protection lookup)" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

# Main: read-only, automation-pushed, post-release-sync target
cat > "$OUT_DIR/main.json" <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["go-test", "go-lint", "shell-lint"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": false
}
JSON

# Next: active development branch for the upcoming release
cat > "$OUT_DIR/next.json" <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "go-test",
      "go-lint",
      "shell-lint",
      "goreleaser-check",
      "perf-bench/tab-completion",
      "perf-bench/native-subcommand",
      "perf-bench/plugin-dispatch",
      "perf-bench/init-bash",
      "acceptance/ubuntu-22.04",
      "acceptance/ubuntu-24.04",
      "acceptance/debian-12",
      "acceptance/fedora-40",
      "acceptance/macos"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": false
}
JSON

# After: experimental branch for post-v1 work
cat > "$OUT_DIR/after.json" <<'JSON'
{
  "required_status_checks": {
    "strict": false,
    "contexts": ["go-test", "go-lint", "shell-lint"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null,
  "allow_force_pushes": true,
  "allow_deletions": false,
  "required_conversation_resolution": false,
  "lock_branch": false
}
JSON

# Before: backport branch for older stable releases
cat > "$OUT_DIR/before.json" <<'JSON'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "go-test",
      "go-lint",
      "shell-lint",
      "goreleaser-check",
      "perf-bench/tab-completion",
      "perf-bench/native-subcommand",
      "perf-bench/plugin-dispatch",
      "perf-bench/init-bash",
      "acceptance/ubuntu-22.04",
      "acceptance/ubuntu-24.04",
      "acceptance/debian-12",
      "acceptance/fedora-40",
      "acceptance/macos"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 1
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true,
  "lock_branch": false
}
JSON

# Tag protection: only automation may create v* tags
cat > "$OUT_DIR/tags.json" <<'JSON'
{
  "pattern": "v*"
}
JSON

echo "Generated protection config files in $OUT_DIR/:"
ls -1 "$OUT_DIR"/

if [[ "$GENERATE_ONLY" == true ]]; then
  echo
  echo "Generation complete. Skipping apply step (--generate-only)."
  exit 0
fi

# Confirm before applying unless --apply or --dry-run
if [[ "$DRY_RUN" == false ]] && [[ "$APPLY_AUTO" == false ]]; then
  echo
  echo "About to apply protection to: $REPO"
  echo "Press Enter to continue, Ctrl-C to abort."
  read -r
fi

# Apply branch protection only when the branch actually exists
apply_branch_protection() {
  local branch="$1"
  local file="$OUT_DIR/${branch}.json"

  if ! gh api "repos/$REPO/branches/$branch" >/dev/null 2>&1; then
    echo "Branch '$branch' does not exist on $REPO — skipping"
    return 0
  fi

  if [[ "$DRY_RUN" == true ]]; then
    echo "DRY: gh api repos/$REPO/branches/$branch/protection --method PUT --input $file"
    return 0
  fi

  echo "Applying protection to branch '$branch'..."
  gh api "repos/$REPO/branches/$branch/protection" \
    --method PUT \
    --input "$file" \
    > /dev/null
  echo "  done"
}

# Apply tag protection unless already configured
apply_tag_protection() {
  local file="$OUT_DIR/tags.json"

  if [[ "$DRY_RUN" == true ]]; then
    echo "DRY: gh api repos/$REPO/tags/protection --method POST --input $file"
    return 0
  fi

  echo "Configuring tag protection for v*..."

  # Avoid duplicate rule creation; GitHub returns existing rules as an array
  local existing
  existing=$(gh api "repos/$REPO/tags/protection" 2>/dev/null \
    | jq -r '.[]?.pattern // empty' || true)

  if echo "$existing" | grep -qx 'v\*'; then
    echo "  v* already protected — skipping"
    return 0
  fi

  gh api "repos/$REPO/tags/protection" \
    --method POST \
    --input "$file" \
    > /dev/null
  echo "  done"
}

echo
echo "Applying to $REPO..."
[[ "$DRY_RUN" == true ]] && echo "(dry run — no changes will be applied)"
echo

# Branches applied in time-sequenced order (main, next, after, before)
for branch in main next after before; do
  apply_branch_protection "$branch"
done

apply_tag_protection

echo
echo "Complete. JSON configs in $OUT_DIR/ for audit and reuse."
