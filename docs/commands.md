# rellog commands

This document lists the intended `rellog` CLI commands.

`rellog` is currently in early design. Command names and options may change before the first stable release, but the responsibility boundaries should remain stable: `rellog` manages changelog entries, plain Markdown release-note files, and `CHANGELOG.md`. It does not manage versions or publish releases.

## Command overview

```text
rellog init
rellog add
rellog add-empty
rellog check
rellog status
rellog require entries
rellog require release <release-id>
rellog prepare <release-id>
```

## `rellog init`

Initialize `rellog` in the current repository.

```sh
rellog init
```

Expected effects:

- create `.rellog/config.kdl`;
- create `.rellog/entries/`;
- create `.rellog/release-notes/`;
- optionally create `CHANGELOG.md` if it does not exist.

The command should not overwrite existing files without an explicit option.

Possible options:

```sh
rellog init --changelog CHANGELOG.md
rellog init --entry-dir .rellog/entries
rellog init --release-note-dir .rellog/release-notes
```

## `rellog add`

Create a new pending changelog entry.

```sh
rellog add
```

The default mode may be interactive. It should guide the user through required metadata such as kind, target, scope, and body.

Non-interactive usage:

```sh
rellog add \
  --kind changed \
  --target rellog \
  --scope cli \
  --issue 12 \
  --body "Added validation for pending changelog entries before release preparation."
```

Possible options:

```text
--kind <kind>          Changelog category, such as added, changed, fixed, removed, security.
--target <target>      Release target or component affected by the change.
--scope <scope>        Optional narrower area within the target.
--breaking             Mark the entry as a breaking change.
--issue <number>       Related GitHub issue number. May be repeated.
--pr <number>          Related GitHub pull request number. May be repeated.
--body <text>          Entry body. Useful for non-interactive use.
--filename <name>      Explicit entry filename.
```

Rules:

- `rellog add` creates a normal entry under `.rellog/entries/`.
- If an empty entry already exists, `rellog add` should fail.
- `rellog add` should not silently remove an empty entry.

## `rellog add-empty`

Create an explicit empty changelog entry.

```sh
rellog add-empty
```

This command is used when there are no changelog-worthy changes for the next release.

The empty entry allows `rellog require entries` to pass without pretending that an actual change exists. It is not a validation bypass option. It is an explicit repository record that means: there is nothing to mention in the changelog for the next release.

Rules:

- if no entry exists, create an empty entry;
- if an empty entry already exists, do nothing;
- if a normal entry already exists, fail.

A normal entry and an empty entry should not coexist.

A possible empty entry format:

```md
---
kind: empty
---

No changelog-worthy changes.
```

## `rellog check`

Validate configuration and pending changelog entries.

```sh
rellog check
```

Expected checks:

- configuration file exists and is valid;
- pending entry files are parseable;
- required metadata is present;
- entry kind is allowed;
- target is known, unless the project allows unknown targets;
- body is not empty;
- breaking-change metadata is consistent;
- normal entries and an empty entry do not coexist.

Possible options:

```text
--entry-dir <path>     Override the pending entry directory.
--strict               Treat warnings as errors.
--format <format>      Output format: human or json.
```

CI should use `rellog check` to detect malformed entries early.

## `rellog status`

Show pending changelog entries.

```sh
rellog status
```

Expected output:

- number of pending entries;
- whether the pending state is normal, empty, or invalid;
- entries grouped by kind or target;
- warnings for invalid or ignored entries;
- whether release preparation would be allowed.

Possible options:

```text
--format <format>      Output format: human or json.
--target <target>      Show entries for a specific target.
--kind <kind>          Show entries for a specific kind.
```

## `rellog require entries`

Require that pending changelog entries exist.

```sh
rellog require entries
```

This command is intended for release-preparation jobs. It should fail when `.rellog/entries/` contains no pending entries.

An empty entry counts as an entry.

Example failure message:

```text
No pending rellog entries found.

Add a changelog entry:
  rellog add

If this release has no changelog-worthy changes, add an explicit empty entry:
  rellog add-empty
```

`rellog require entries` should not update files. It is a gatekeeping command.

## `rellog require release <release-id>`

Require that a prepared release-note file exists.

```sh
rellog require release v1.0.1
```

This command should fail unless the following file exists:

```text
.rellog/release-notes/v1.0.1.md
```

This is useful for later release workflow steps that should only proceed after `rellog prepare <release-id>` has created a release-note file.

For v0, release ids should be path-safe because they are used as filenames. A conservative allowed form is:

```text
[A-Za-z0-9._-]+
```

## `rellog prepare <release-id>`

Prepare a release-note file and update `CHANGELOG.md` using pending entries.

```sh
rellog prepare v1.0.1
```

Expected behavior:

- validate pending entries;
- require at least one pending entry;
- fail if normal entries and an empty entry coexist;
- create `.rellog/release-notes/<release-id>.md` from pending entries;
- append the prepared release-note content to `CHANGELOG.md`;
- delete consumed files from `.rellog/entries/`.

If `.rellog/release-notes/<release-id>.md` already exists, the command should fail by default rather than silently overwriting it.

Possible options:

```text
--date <date>              Release date to include in headings or metadata.
--changelog <path>         Override the CHANGELOG path.
--entry-dir <path>         Override the pending entry directory.
--release-note-dir <path>  Override the release-note directory.
--dry-run                  Show intended changes without writing files.
```

`rellog prepare <release-id>` should not:

- decide the next version;
- update package manifests;
- create Git tags;
- create GitHub Releases;
- publish packages, binaries, or artifacts.

## Release-note files

`rellog` release-note files are plain Markdown files stored under `.rellog/release-notes/`.

They are not GitHub Release Notes. Other release tooling may reuse them, but `rellog` itself only creates and validates repository-managed Markdown files.

Example normal release-note file:

```md
## v1.0.1

### Changed

- Added validation for pending changelog entries before release preparation.
```

Example empty release-note file:

```md
## v1.0.1

No changelog-worthy changes.
```
