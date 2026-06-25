# rellog workflow

This document describes the intended CHANGELOG and release-note-file workflow for `rellog`.

## Principle

`rellog` separates development history from release explanation.

Git history records how work was divided and performed. A CHANGELOG records what changed in a release and how that change should be communicated to users, operators, or maintainers.

`rellog` therefore uses explicit changelog entries as its primary input. It does not reconstruct final release notes from commit history.

## Repository layout

A typical repository using `rellog` has the following files:

```text
.rellog/
  config.kdl
  entries/
    <entry-id>.md
  release-notes/
    <release-id>.md
CHANGELOG.md
```

The important distinction is that pending changelog entries, prepared release-note files, and the cumulative `CHANGELOG.md` are separate artifacts.

- `.rellog/entries/` stores pending changelog entries for the next release preparation.
- `.rellog/release-notes/` stores prepared plain Markdown release-note files by release id.
- `CHANGELOG.md` stores the cumulative changelog assembled from prepared release notes.

## Changelog entry

A changelog entry is a small Markdown file that describes one release-note-level change.

Example:

```md
---
targets:
  - rellog
kind: changed
scope: cli
breaking: false
issues:
  - 12
---

Added validation for pending changelog entries before release preparation.
```

The entry should describe the change at the level that should appear in release notes. It should not merely restate a commit message, issue title, or implementation step.

## Empty changelog entry

An empty changelog entry explicitly records that there is nothing changelog-worthy for the next release.

It is created with:

```sh
rellog add-empty
```

This is not a validation bypass option. It is a repository record.

The rule is simple:

```text
release preparation requires at least one pending entry
```

That entry may be a normal changelog entry or an empty changelog entry.

Recommended rules:

- `rellog add-empty` creates an empty entry only when there are no pending entries.
- If an empty entry already exists, `rellog add-empty` is a no-op.
- If normal entries already exist, `rellog add-empty` fails.
- If an empty entry exists, `rellog add` fails.

A normal entry and an empty entry should not coexist, because they represent contradictory release states.

## Development workflow

### 1. Implement, review, or finalize a change

A project may add a changelog entry during implementation, during review, or when a change policy is finalized.

The entry is not a full design document. It is a short record of how the change should be explained in a release note and CHANGELOG.

### 2. Add a pending entry

A contributor creates an entry:

```sh
rellog add
```

Non-interactive usage may also be supported:

```sh
rellog add \
  --kind changed \
  --target rellog \
  --scope cli \
  --issue 12 \
  --body "Added validation for pending changelog entries before release preparation."
```

### 3. Add an empty entry when there is nothing to mention

If the next release has no changelog-worthy changes, create an explicit empty entry:

```sh
rellog add-empty
```

This records that the project intentionally has nothing to mention. It avoids treating an empty release as a hidden workflow exception.

### 4. Validate entries in CI

Pull requests should validate pending entries:

```sh
rellog check
```

The check should fail if entries have invalid metadata, unknown kinds, unknown targets, empty bodies, or contradictory states such as normal entries coexisting with an empty entry.

## Release preparation workflow

### 1. Require pending entries

Before preparing a release, require at least one pending entry:

```sh
rellog require entries
```

If `.rellog/entries/` is empty, the command fails.

The failure message should direct the user to either add a normal entry or add an explicit empty entry:

```text
No pending rellog entries found.

Add a changelog entry:
  rellog add

If this release has no changelog-worthy changes, add an explicit empty entry:
  rellog add-empty
```

### 2. Prepare release notes and CHANGELOG

Release preparation receives a release id from the outside:

```sh
rellog prepare v1.0.1
```

`rellog` does not decide this id. A release id may be a semantic version, date, Git tag name, distribution label, or another repository-defined id.

For v0, release ids should be path-safe because they are used as filenames under `.rellog/release-notes/`. A conservative allowed form is:

```text
[A-Za-z0-9._-]+
```

`rellog prepare <release-id>` should:

1. validate pending entries;
2. fail if there are no pending entries;
3. fail if normal entries and an empty entry coexist;
4. create `.rellog/release-notes/<release-id>.md` from pending entries;
5. append the prepared release-note content to `CHANGELOG.md`;
6. delete consumed files from `.rellog/entries/`.

The command should not create Git tags, update package manifests, create GitHub Releases, or publish artifacts.

### 3. Require prepared release notes

After release preparation, later release workflow steps may require the prepared release-note file:

```sh
rellog require release v1.0.1
```

This command fails unless the following file exists:

```text
.rellog/release-notes/v1.0.1.md
```

This is useful when publish jobs should only proceed after release notes have been prepared.

## Release-note files

`rellog` release notes are plain Markdown files.

They are not GitHub Release Notes, and `rellog` does not create GitHub Releases. Other release tooling may choose to reuse `.rellog/release-notes/<release-id>.md`, but that is outside `rellog`'s core responsibility.

A normal release-note file may look like this:

```md
## v1.0.1

### Changed

- Added validation for pending changelog entries before release preparation.
```

An empty release-note file may look like this:

```md
## v1.0.1

No changelog-worthy changes.
```

## GitHub Actions workflow

`rellog` should provide a GitHub Action for a changesets-like experience without version management.

### Validate entries on pull requests

```yaml
name: Check rellog entries

on:
  pull_request:

jobs:
  rellog-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: tooppoo/rellog/action/check@v0
```

### Prepare a CHANGELOG update pull request

```yaml
name: Prepare changelog

on:
  workflow_dispatch:
    inputs:
      release_id:
        required: true
        type: string

jobs:
  prepare-changelog:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: tooppoo/rellog/action/prepare@v0
        with:
          release-id: ${{ inputs.release_id }}
          create-pr: true
```

The resulting pull request should contain only rellog-managed documentation artifacts:

- create `.rellog/release-notes/<release-id>.md`;
- update `CHANGELOG.md`;
- delete consumed files from `.rellog/entries/`.

It should not update versions or publish anything.

## Future AI-assisted workflow

AI support may be considered after the core workflow is stable.

The acceptable role of AI is to suggest candidate changelog entries from issues, pull requests, diffs, or commit history. The final changelog entry should still be reviewed and accepted as an explicit project record.

In other words, AI may help draft entries, but `rellog` should not silently infer final release notes from Git history.
