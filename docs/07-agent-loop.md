# shy — Agent Loop

How AI agents (Claude) work within the shy repository over time.
The shy *runtime* has no autonomous behaviour — `shy` is a
deterministic CLI binary. This document covers only the
*development-time* loop between the operator and AI assistants.

Where this document and `04-agent-instructions.md` overlap, `04`
covers *what an agent should and shouldn't do*; this document
covers *how the operator and agent interact over time*.

## Invocation cadence

An agent is invoked when the operator wants help. There is no
scheduled loop, no background runner, no autonomous agent watching
the repo. Solo+contrib projects don't justify that infrastructure.

Typical invocation contexts:

1. **Implementation session.** Operator opens a chat, gives a task
   (e.g. "implement `shy install` per Step 12 of `03-short-
   horizon.md`"), the agent generates the implementation, the
   operator commits.
2. **Design clarification.** Operator hits a design question that
   doesn't have a clean answer in the existing docs; agent helps
   reason through and proposes a revision.
3. **Code review assistance.** Operator pastes a PR diff or
   describes a change; agent points out concerns, suggests
   improvements.
4. **Maintenance task.** Operator wants to triage issues, draft a
   CHANGELOG note, or update a dependency; agent assists.

The agent does not initiate work. It responds.

## Session structure

Each session has three phases:

**1. Orient.** The agent reads `04-agent-instructions.md` first
(always), then any other docs relevant to the task. If the task
references code, the agent reads the code. If the task is about a
specific document, the agent reads that document and its
dependencies.

**2. Execute.** The agent does the work. For implementation
tasks, this means generating code, tests, and any necessary doc
updates. For design tasks, this means proposing changes and
discussing trade-offs.

**3. Hand off.** The agent summarises what was done, what remains,
and what the operator needs to verify before committing. If the
agent created files, it uses `present_files`. If the agent
modified content in chat, it summarises the diff and waits for
confirmation before proposing a commit message.

## Reporting format

When reporting on completed work, the agent uses this structure:

```
## Done

- [concrete outcome 1]
- [concrete outcome 2]

## Verify before commit

- [what the operator should check]

## Open questions

- [things the agent couldn't resolve without input]

## Suggested commit message

`<type>(<scope>): <subject>`

[body if non-trivial]
```

Format scales down for trivial tasks. Don't pad with empty
sections.

## Escalation criteria

The agent escalates to the operator (asks rather than acts) when:

- **Design decision is open.** Anything not covered by existing
  docs or where multiple reasonable answers exist. The agent
  proposes options with rationale and waits.
- **Confidentiality unclear.** See `04-agent-instructions.md` and
  the project-design skill's confidentiality checklist. If the
  agent is about to commit information whose disclosure status is
  ambiguous, stop and ask.
- **Breaking change suspected.** If a refactor or fix would
  change shy's public API (manifest schema, command surface,
  `install.sh` contract, GoReleaser asset names), flag explicitly
  before making the change.
- **Scope drift.** If the task as understood would expand beyond
  what was asked, stop and confirm scope.
- **Tooling failure.** If a command fails repeatedly with unclear
  cause, escalate rather than retry indefinitely.

Escalation is not failure. It's the agent honouring the limits of
what it can decide unilaterally.

## Termination criteria

A session terminates when one of:

1. The task is complete and the operator confirms.
2. The operator says "stop" or equivalent.
3. The agent has hit a hard blocker (escalation needed, decision
   from operator required) and the operator hasn't responded
   within the session's context window.
4. The agent recognizes the work has reached natural completion
   even without explicit confirmation — output delivered, no
   pending questions, operator silent for an extended exchange.

The agent does not continue work after termination. It doesn't
proactively suggest follow-on tasks unless explicitly asked.

## Context strategy between sessions

Each session starts fresh — the agent has no memory of previous
sessions unless the operator provides context. The operator's
options for continuity:

**1. Persistent docs.** The design package (`shy-design/00`–`07`),
README, CHANGELOG, and code comments are the long-term memory.
Every important decision lives in one of these files. A new
agent in a new session reads them and recovers context.

**2. Memory (operator-managed).** The Claude memory system can
hold ongoing context across sessions. This is operator-managed
(via `memory_user_edits`), not agent-managed. The agent reads
memory at session start but does not silently update it. Memory
holds:
   - Operator's preferences and conventions (already populated)
   - Default organisation, default tooling choices
   - Project-specific recurring context

**3. Conversation continuity.** The operator can resume a chat
where they left off. Anthropic's chat UI supports this. For
medium-complexity tasks that span hours, this is the simplest
strategy.

**4. Status documents in repo.** For long-running multi-session
tasks (e.g. "implement Phase 2"), the operator may commit a
`STATUS.md` or use GitHub issues to track progress. Each session,
the agent reads the status doc to know what's been done and
what's next.

The shy project explicitly does *not* use a `.claude-status.json`
or similar machine-managed file for state. Continuity flows
through human-readable documents that the operator owns.

## Multi-agent considerations

For v1, shy uses a single agent at a time. Multi-agent
orchestration (research agent, implementation agent, review agent
working in concert) is out of scope for v1 and likely v1.x.

If the operator ever wants to run a multi-agent flow on shy, the
existing `dev-flow` skill (separately maintained) can be invoked
explicitly. The shy project doesn't bake multi-agent assumptions
into its own design.

## Operator's role

The operator is the trust root, the design authority, and the
final reviewer. The agent:

- Drafts; the operator commits.
- Proposes; the operator decides.
- Implements; the operator verifies.

The agent does not push to `next`, does not merge PRs, does not
tag releases, does not approve PRs from external contributors.
All of those are operator actions, even when the operator's
decision is "yes" to something the agent recommended.

This is intentional in `solo+contrib`. As the project (hypothet-
ically) grows toward `team`, more authority could be delegated to
the agent, but that's a v2+ conversation.

## Failure modes

**Agent hallucinates a fact.** Operator catches it in review.
Mitigation: the agent should cite sources (a specific file, a
specific line) when making non-trivial factual claims; this makes
hallucination easier to spot.

**Agent generates code that compiles but is wrong.** The CI catches
many of these. Tests catch more. Operator's manual review catches
the rest. This is why tests are written alongside features, not
deferred.

**Agent and operator disagree on a design decision.** The agent
states its position with reasoning, then yields to operator's
final call. The disagreement, if substantive, is recorded in a
design doc revision so future agents in future sessions see the
resolution.

**Agent breaks tooling.** If the agent's commands (build, test,
lint) leave the workspace in a broken state, the agent reports
the failure honestly and walks back the changes. The agent does
not paper over failures.

**Long context window exhaustion.** When the conversation grows
large, the agent's earliest context fades. Mitigation: the
operator periodically asks the agent to summarise the session so
far; the summary becomes a fresh anchor.
