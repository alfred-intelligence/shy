# Security Policy

## Supported versions

shy is on the v0.x track. Only the latest tagged release is supported
with security fixes during pre-1.0; once v1.0 ships, the support
window will be defined here.

## Reporting a vulnerability

Please report security issues **privately**, not as public GitHub
issues:

1. Open a private advisory at
   <https://github.com/alfred-intelligence/shy/security/advisories/new>.
2. Provide the affected versions, reproduction steps, and the
   smallest proof-of-concept you have.
3. Expect an initial reply within seven days. We coordinate fix
   timing with the reporter and request a CVE when appropriate.

## Threat model — what shy defends against

- **Tampered binary downloads.** `install.sh` verifies SHA256
  against the per-asset `.sha256` file published with each release.
- **Other users on shared hosts.** `shy init` creates
  `$HOME/.shy/` with `chmod 700`.
- **Accidental upgrades.** `shy install @user/repo` pins to the
  current HEAD commit; `shy collection update` defaults to
  `--dry-run`.

## What shy does **not** defend against

These are architectural and documented in `docs/01-whitepaper.md`:

- **Sourced scripts cannot be sandboxed in any version of shy.**
  Operator discipline is the only mechanism for snippets that get
  sourced into the interactive shell. Pick the collections you
  subscribe to like you pick apt repositories.
- **Compromised git hosts** serving malicious collection content.
- **Compromised binaries** — there is no GPG signing in v1.
- **Local privilege escalation through plugin scripts** — plugins
  run with the user's privileges; sandboxing is planned for v2.
- **A compromised operator account**.

## Plugin and capability declarations

The `[capabilities]` block in `manifest.toml` is parsed and ignored
in v1. The forthcoming `shy audit` plugin will read it for
static-vs-declared analysis; v2 sandboxing will enforce it.
Declaring capabilities today buys forward compatibility, not
runtime protection.

## `[security]` tag

Manifests can mark a release as a security fix via:

```toml
[security]
fixes = "CVE-YYYY-NNNNN"
severity = "high"
description = "..."
```

In v1 these claims are trusted at face value and bypass the
update-throttle and snooze. v2 will verify CVE references against
the NVD or GitHub Advisory Database. False severity claims today
have only social consequences; that may not be true in v2.
