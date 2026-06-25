# rellog

Git log is not a release log.

`rellog` is a runtime-independent CHANGELOG and release-note-file management tool.

It is inspired by the [changesets](https://github.com/changesets/changesets) workflow, but it is not compatible with `changesets` and does not manage versions. Its scope is limited to collecting explicit changelog entries, preparing plain Markdown release-note files, and appending those release notes to `CHANGELOG.md`.

## Concept

Git records how work happened. It does not reliably explain the totality of a release.

A release often contains implementation commits, review fixes, issue splits, documentation adjustments, refactors, CI changes, and small follow-up corrections. Those records are useful as development history, but they are a weak substrate for explaining what changed for users.

`rellog` treats CHANGELOG content as an edited release record, not as a mechanical summary of Git history.

The central unit is a changelog entry: a small Markdown file that describes a change at the level it should be communicated in release notes. Entries are accumulated before release preparation and then consumed into a release-note file and `CHANGELOG.md`.

## Why rellog?

### Git history is not the same as a changelog

Commit history, pull request titles, and merge comments are shaped by work management. They often describe how the work was divided, not what the release means as a whole.

`rellog` therefore does not infer final release notes from commits. It expects explicit changelog entries that summarize changes at the user-facing, operator-facing, or maintainer-facing level.

### Version management belongs to each ecosystem

Versioning practices vary heavily by ecosystem:

- Node projects use `package.json`, workspaces, and registry-specific conventions.
- Rust projects use `Cargo.toml`, crates, and tags.
- Go projects often rely on module tags.
- CLI tools may release binaries, installers, package-manager manifests, or GitHub Releases.
- Documentation sites and web applications may not have a package version at all.

`rellog` deliberately does not decide versions, update package manifests, create Git tags, or publish artifacts. A release id is supplied from the outside when release notes and `CHANGELOG.md` are prepared.

### Empty releases should also be explicit

Sometimes a release has no changelog-worthy changes. That should still be an explicit repository state, not a hidden workflow override.

`rellog add-empty` creates an empty changelog entry. This entry means: the project intentionally records that there is nothing to mention in the changelog for the next release. Because it is still an entry, `rellog require entries` can remain simple: release preparation requires at least one pending entry.

### Release notes are plain Markdown files

`rellog` can prepare release notes as plain Markdown files under `.rellog/release-notes/`.

These files are not GitHub Release Notes. They are repository-managed Markdown artifacts that can be appended to `CHANGELOG.md`, reviewed in pull requests, and reused by other release tooling.

## Repository layout

A typical repository using `rellog` has this layout:

```text
.rellog/
  config.kdl
  entries/
    <entry-id>.md
  release-notes/
    <release-id>.md
CHANGELOG.md
```

## Basic workflow

1. Add a changelog entry while implementing, reviewing, or finalizing a change.
2. If there is nothing to mention, add an explicit empty entry.
3. Validate pending entries in CI.
4. Require at least one pending entry before release preparation.
5. Prepare release notes for a supplied release id.
6. Append the prepared release notes to `CHANGELOG.md`.
7. Remove consumed pending entries.

See [docs/workflow.md](docs/workflow.md) for the intended workflow.

## What rellog does

`rellog` is intended to:

- initialize `.rellog/`;
- create changelog entry files;
- create an explicit empty entry when there is nothing to mention;
- validate entry format and required metadata;
- list pending entries;
- require at least one pending entry before release preparation;
- prepare `.rellog/release-notes/<release-id>.md` from pending entries;
- append prepared release notes to `CHANGELOG.md`;
- require a prepared release-note file for a release id;
- support GitHub Actions that create CHANGELOG update pull requests.

## What rellog does not do

`rellog` does not:

- decide the next version;
- update `package.json`, `Cargo.toml`, `pyproject.toml`, or any other package manifest;
- create Git tags;
- publish packages, binaries, or artifacts;
- create GitHub Releases;
- treat `.rellog/release-notes/*.md` as GitHub Release Notes;
- generate final release notes directly from commit history;
- require Conventional Commits;
- provide compatibility with Changesets file semantics.

## When not to use rellog

`rellog` is probably not the right tool when:

- commit history is already clean enough to be the primary changelog source;
- the project wants Conventional Commits based generation;
- version bumping and package publishing should be managed by the same tool;
- the project needs Changesets compatibility;
- the team wants fully automatic release-note inference without explicit changelog entries.

In those cases, commit-history based changelog generators or ecosystem-specific release automation may be a better fit.

## Documentation

- [Workflow](docs/workflow.md)
- [Commands](docs/commands.md)

## Project status

`rellog` is currently in early design. The initial goal is a small, language- and runtime-independent CLI for managing changelog entries, preparing plain Markdown release-note files, and updating `CHANGELOG.md`.
