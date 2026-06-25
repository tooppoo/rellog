# rellog workflow

This document describes the intended CHANGELOG management workflow for `rellog`.

## Principle

`rellog` separates development history from release explanation.

Git history records how the work was divided and performed. A CHANGELOG records what changed in a release and how that change should be communicated to users, operators, or maintainers.

`rellog` therefore uses explicit changelog entries as its primary input. It does not reconstruct release notes from commit history.

## Repository layout

A typical repository using `rellog` has the following files:

```text
.rellog/
  config.toml
  entries/
    <entry-id>.md
CHANGELOG.md
```

The exact directory names are part of the initial design and may be adjusted before the first stable release. The important distinction is that pending changelog entries are stored separately from the generated or maintained `CHANGELOG.md`.

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

## Development workflow

### 1. Implement or plan a change

A project may add a changelog entry during implementation, during review, or when a change policy is finalized.

The entry is not a full design document. It is a short record of how the change should be explained in a CHANGELOG.

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

### 3. Validate entries in CI

Pull requests should validate pending entries:

```sh
rellog check
```

The check should fail if entries have invalid metadata, unknown kinds, unknown targets, or empty bodies.

This does not mean every pull request must always include an entry. Some projects may allow documentation-only or internal-only changes without user-facing entries. That policy should be explicit in the repository's workflow.

### 4. Require changelog records before release preparation

At release preparation time, a project can require at least one pending entry:

```sh
rellog require
```

If no pending entry exists, the command should fail by default. This prevents accidental releases with no CHANGELOG record.

If an empty release is intentional, it should require an explicit reason:

```sh
rellog require --allow-empty --empty-reason "Rebuild only; no user-visible changes."
```

The goal is not to force noise into the CHANGELOG. The goal is to distinguish an intentional no-change release from a forgotten changelog update.

## Release preparation workflow

### 1. Provide a release id

`rellog` does not decide versions. The release id is supplied by the release process:

```sh
rellog prepare --release-id v0.1.0
```

A release id may be a semantic version, date, Git tag name, distribution label, or any string that the repository uses for CHANGELOG sections.

### 2. Render release notes

Before writing to `CHANGELOG.md`, release notes can be rendered for inspection:

```sh
rellog render --release-id v0.1.0
```

This should produce the markdown section that would be inserted into `CHANGELOG.md`.

### 3. Update CHANGELOG.md

Release preparation writes the generated section into `CHANGELOG.md`:

```sh
rellog prepare --release-id v0.1.0 --date 2026-06-25
```

This command should not create Git tags, update package manifests, or publish artifacts. It only prepares CHANGELOG changes.

### 4. Consume pending entries

After entries are included in `CHANGELOG.md`, they are no longer pending.

The project may choose one of two policies:

- delete consumed entries;
- move consumed entries to an archive directory.

The default should be conservative and easy to review in pull requests. Deleting consumed entries is simple. Archiving provides more traceability but increases repository noise.

## GitHub Actions workflow

`rellog` should provide a GitHub Action for a changesets-like experience without version management.

### Validate entries on pull requests

```yaml
name: Check changelog entries

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
      allow_empty:
        required: false
        default: "false"

jobs:
  prepare-changelog:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: tooppoo/rellog/action/prepare@v0
        with:
          release-id: ${{ inputs.release_id }}
          require-entry: true
          allow-empty: ${{ inputs.allow_empty }}
          create-pr: true
```

The resulting pull request should contain only CHANGELOG-related changes:

- update `CHANGELOG.md`;
- delete or archive consumed entry files.

It should not update versions or publish anything.

## Future AI-assisted workflow

AI support may be considered after the core workflow is stable.

The acceptable role of AI is to suggest candidate changelog entries from issues, pull requests, diffs, or commit history. The final changelog entry should still be reviewed and accepted as an explicit project record.

In other words, AI may help draft entries, but `rellog` should not silently infer final release notes from Git history.
