# Use explicit changelog entries instead of Git history as the primary input

- Status: Proposed
- Created: 2026-06-26T00:00:00Z

## Context

`rellog` is a CHANGELOG and release-note-file management tool inspired by the changeset-file workflow.

The central problem is that Git history is not a reliable explanation of the totality of a release.

Git records how work happened. It records commits, issue splits, pull request boundaries, review fixes, refactors, documentation adjustments, CI changes, and other implementation or maintenance details. These records are useful as development history, but they do not necessarily describe what changed at the level users, operators, or maintainers need in a CHANGELOG.

This is especially important when work is intentionally split into small issues or pull requests. The resulting history may be good for implementation and review, but poor as release explanation. Several small changes may only become one meaningful release-note unit when considered together.

Version management is also excluded from `rellog`'s core scope. Versioning practices differ substantially across ecosystems such as Node, Rust, Go, Python, CLI binary distribution, documentation sites, and web applications.

This decision should be recorded as an ADR because it defines the central responsibility boundary of `rellog`: it is not a Git-history-based changelog generator and not a version manager.

## Decision

`rellog` will use explicit changelog entries as the primary input for release notes and CHANGELOG updates.

A changelog entry is a small Markdown file that describes a change at release-note granularity. It should describe how the change should be communicated, not merely restate a commit message, issue title, or pull request title.

`rellog` will not infer final release notes from Git history.

`rellog` will not decide versions, update package manifests, create Git tags, create GitHub Releases, or publish artifacts.

`rellog` may refer to issues, pull requests, diffs, or commits as supporting context, but they are not the primary release-note source.

## Alternatives Considered

### Generate CHANGELOG from Git history

This is the model used by commit-history-based changelog generators.

It was not selected because Git history is shaped by work management and implementation process. It often contains noisy, fragmented, or overly mechanical information. It does not reliably explain the release as a whole.

### Require Conventional Commits

This was not selected because it still makes commit history the primary changelog substrate. It also requires contributors and automation to maintain a disciplined commit style throughout development, which conflicts with the premise that release explanation often becomes clear only after multiple changes are considered together.

### Use Changesets directly

This was not selected because Changesets is tied to Node-oriented workflows and version management. `rellog` is inspired by the changeset-file workflow, but it is not compatible with Changesets and does not manage versions.

### Combine changelog management and version management

This was not selected because version management is ecosystem-specific. Combining version bumping, manifest updates, tags, publishing, and changelog management would make `rellog` a release automation framework rather than a focused changelog tool.

## Consequences

### Positive Consequences

- `rellog` can be used across ecosystems without requiring Node, npm, or package manifest conventions.
- Release notes are based on explicit release-note-level records instead of inferred Git history.
- The tool scope remains narrow and easier to reason about.
- The design supports late changelog entry creation, including release-preparation-time summarization.

### Negative Consequences

- `rellog` cannot automatically produce final release notes from Git history alone.
- Projects must explicitly create changelog entries.
- Users looking for fully automatic changelog generation will need another tool.

### Neutral Consequences

- AI-assisted entry suggestion may be added later, but final entries remain explicit project records.
- Git history may still be useful as supporting context, but not as the authoritative source.
