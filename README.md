# rellog

`rellog` is a runtime-independent CHANGELOG management tool.

It is inspired by the changeset-file workflow, but it is not compatible with Changesets and does not manage versions. Its scope is limited to collecting, validating, and aggregating explicit changelog entries into `CHANGELOG.md`.

## Concept

Git records how work happened. It does not reliably explain the totality of a release.

A release often contains implementation commits, review fixes, issue splits, documentation adjustments, refactors, CI changes, and small follow-up corrections. Those records are useful as development history, but they are a weak substrate for explaining what changed for users.

`rellog` treats CHANGELOG content as an edited release record, not as a mechanical summary of Git history.

The central unit is a changelog entry: a small Markdown file that describes a change at the level it should be communicated in release notes. The entry is written before release preparation, reviewed with the code or policy change, and later aggregated into `CHANGELOG.md`.

## Why rellog?

### Git history is not the same as a changelog

Commit history, pull request titles, and merge comments are shaped by work management. They often describe how the work was divided, not what the release means as a whole.

`rellog` therefore does not infer release notes from commits. It expects explicit changelog entries that summarize changes at the user-facing or operator-facing level.

### Version management belongs to each ecosystem

Versioning practices vary heavily by ecosystem:

- Node projects use `package.json`, workspaces, and registry-specific conventions.
- Rust projects use `Cargo.toml`, crates, and tags.
- Go projects often rely on module tags.
- CLI tools may release binaries, installers, package-manager manifests, or GitHub Releases.
- Documentation sites and web applications may not have a package version at all.

`rellog` deliberately does not decide versions, update package manifests, create Git tags, or publish artifacts. A release id is supplied from the outside when `CHANGELOG.md` is prepared.

### Human-authored entries should still be enforced

Human-written CHANGELOG entries are more appropriate than raw Git history for explaining the totality of a change. They are also easy to forget.

`rellog` is designed to pair local CLI commands with GitHub Actions so a project can require pending changelog entries before release preparation. For example, a release-preparation job may fail when no pending entry exists unless an explicit empty-release reason is provided.

## Basic workflow

1. Add a changelog entry while implementing or reviewing a change.
2. Validate pending entries in CI.
3. At release preparation time, provide a release id.
4. Generate or update `CHANGELOG.md` from pending entries.
5. Remove or archive consumed entries.

See [docs/workflow.md](docs/workflow.md) for the intended workflow.

## What rellog does

`rellog` is intended to:

- initialize a changelog-entry directory;
- create changelog entry files;
- validate entry format and required metadata;
- list pending entries;
- require at least one pending entry before release preparation;
- render release notes from pending entries;
- update `CHANGELOG.md` for a supplied release id;
- support GitHub Actions that create CHANGELOG update pull requests.

## What rellog does not do

`rellog` does not:

- decide the next version;
- update `package.json`, `Cargo.toml`, `pyproject.toml`, or any other package manifest;
- create Git tags;
- publish packages or artifacts;
- generate release notes directly from commit history;
- require Conventional Commits;
- provide compatibility with Changesets file semantics.

## When rellog does not fit

`rellog` is probably not the right tool when:

- commit history is already clean enough to be the primary changelog source;
- the project wants Conventional Commits based generation;
- version bumping and package publishing should be managed by the same tool;
- the project needs Changesets compatibility;
- the team wants fully automatic release-note inference without explicit changelog entries.

In those cases, tools such as commit-history based changelog generators or ecosystem-specific release automation may be a better fit.

## Documentation

- [Workflow](docs/workflow.md)
- [Commands](docs/commands.md)

## Project status

`rellog` is currently in early design. The initial goal is a small, language- and runtime-independent CLI for managing changelog entries and preparing `CHANGELOG.md` updates.
