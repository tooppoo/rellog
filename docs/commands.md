# rellog commands

This document lists the intended `rellog` CLI commands.

For file layout and file formats, see [files.md](files.md). For lifecycle and workflow guards, see [workflow.md](workflow.md).

`rellog` is currently in early design. Command names and options may change before the first stable release, but the responsibility boundaries should remain stable: `rellog` manages changelog entries, plain Markdown release-note files, and `CHANGELOG.md`. It does not manage versions or publish releases.

## Command overview

```text
rellog init
rellog add
rellog add-empty
rellog check
rellog status
rellog require release <release-id>
rellog prepare <release-id>
```

## `rellog init`

Initialize `rellog` in the current repository.

```sh
rellog init
```

Expected effects:

- create the `rellog` configuration file;
- create the pending entry directory;
- create the release-note directory;
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

When no flags are provided, `rellog add` should run in interactive mode. It
should guide the user through the entry fields in this order:

1. `kind`
2. `target`
3. `body`
4. `issues`
5. `prs`

The interactive guide for `issues` and `prs` must say that the field may be
left empty. It must also say that multiple values can be entered either as a
comma-separated list or as a space-separated list. The guide is user-facing
help text on stdout; tests should not rely on its exact wording.

Non-interactive usage:

```sh
rellog add \
  --kind changed \
  --target rellog \
  --issue 12 \
  --body "Added validation for pending changelog entries before release preparation."
```

Possible options:

```text
--kind <kind>          Changelog category, such as added, changed, fixed, removed, security.
--target <target>      Release target, component, or area affected by the change.
--breaking             Mark the entry as a breaking change.
--issue <number>       Related GitHub issue number. May be repeated.
--pr <number>          Related GitHub pull request number. May be repeated.
--body <text>          Entry body. Useful for non-interactive use.
```

Rules:

- `rellog add` creates a normal entry using the next generated sequence filename.
- If an empty entry already exists, `rellog add` should fail with `ExitEntryConflict`.
- `rellog add` should not silently remove an empty entry.
- Users cannot specify the entry filename.
- When any flag is provided, `rellog add` runs in non-interactive mode.
- Interactive and non-interactive mode must validate `kind` against
  `rellog.entries.kinds`. An undefined kind is an error and no entry file is
  created.
- Interactive and non-interactive mode must handle targets that are not listed
  in `rellog.entries.targets` according to `target-policy`:
  - `deny-unknown`: fail with an error and do not create an entry file.
  - `warn-unknown`: print a warning to stderr and create the entry file.
  - `allow-unknown`: create the entry file without a warning.

## `rellog add-empty`

Create an explicit empty changelog entry.

```sh
rellog add-empty
```

This command is used when there are no changelog-worthy changes for the next release.

The empty entry exists so `rellog prepare <release-id>` can produce an explicit empty release-note file without pretending that an actual change exists.

Rules:

- if no entry exists, create an empty entry;
- if an empty entry already exists, do nothing;
- if a normal entry already exists, fail with `ExitEntryConflict`.

A normal entry and an empty entry should not coexist.

`rellog add` and `rellog add-empty` are the entry points that prevent users from creating an entry conflict through `rellog`.

## `rellog check`

Validate configuration and pending changelog entries.

```sh
rellog check
```

Expected checks:

- configuration file exists and is valid;
- pending entry files are parseable;
- pending entry filenames follow the generated sequence format;
- required metadata is present;
- entry kind is allowed;
- target is known, unless the project allows unknown targets;
- body is not empty;
- breaking-change metadata is consistent;
- normal entries and an empty entry do not coexist.

If normal entries and an empty entry coexist because of manual file edits, this is an entry conflict and `rellog check` should fail with `ExitEntryConflict`.

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
- whether the pending state is normal, empty, absent, or invalid;
- entries grouped by kind or target;
- warnings for invalid or ignored entries;
- whether release preparation would be allowed.

Possible options:

```text
--format <format>      Output format: human or json.
--target <target>      Show entries for a specific target.
--kind <kind>          Show entries for a specific kind.
```

## `rellog require release <release-id>`

Require that a prepared release-note file exists.

```sh
rellog require release v1.0.1
```

This command is intended for publish-oriented release jobs. It should fail unless the prepared release-note file for the given release id exists.

For v0, release ids should be path-safe because they are used as filenames. See [files.md](files.md) for the filename rule.

## `rellog prepare <release-id>`

Preview release-note preparation using pending entries.

```sh
rellog prepare v1.0.1
```

By default, `prepare` is a dry run. It validates pending entries and shows the release-note Markdown and intended file operations without writing files or deleting entries.

To execute the preparation, pass `--run` explicitly:

```sh
rellog prepare v1.0.1 --run
```

Expected behavior:

- validate pending entries;
- fail if there are no pending entries;
- fail with `ExitEntryConflict` if normal entries and an empty entry coexist;
- aggregate pending entries in filename order;
- without `--run`, print the generated release-note content and intended operations without changing files;
- with `--run`, create the release-note file for the release id;
- with `--run`, append the prepared release-note content to `CHANGELOG.md`;
- with `--run`, delete consumed pending entries.

If pending entries are absent, the command should tell the user to either add normal entries or create an explicit empty entry.

Example failure message:

```text
No pending rellog entries found.

Add a changelog entry:
  rellog add

If this release has no changelog-worthy changes, add an explicit empty entry:
  rellog add-empty
```

If the release-note file for the release id already exists, the command should fail by default rather than silently overwriting it.

If manual file edits create an entry conflict, `rellog prepare <release-id>` and `rellog prepare <release-id> --run` should fail before writing a release-note file, appending to `CHANGELOG.md`, or deleting pending entries.

Possible options:

```text
--date <date>              Release date to include in headings or metadata.
--changelog <path>         Override the CHANGELOG path.
--entry-dir <path>         Override the pending entry directory.
--release-note-dir <path>  Override the release-note directory.
--run                      Write files and delete consumed pending entries.
```

`rellog prepare <release-id>` should not:

- decide the next version;
- update package manifests;
- create Git tags;
- create GitHub Releases;
- publish packages, binaries, or artifacts.

## Exit codes

| Code | Constant               | Description                                                             |
|------|------------------------|-------------------------------------------------------------------------|
| 0    | —                      | Success                                                                 |
| 1    | `ExitNotInitialized`   | `rellog` has not been initialized; run `rellog init` first              |
| 2    | `ExitInvalidStructure` | A path that must be a directory exists as a file (e.g. `.rellog/entries`) |
| 3    | `ExitCheckFailed`      | `rellog check` found one or more non-conflict validation errors in pending entries |
| 4    | `ExitReleaseNotFound`  | The required release-note file does not exist; run `rellog prepare <release-id> --run` first |
| 5    | `ExitEntryConflict`    | Empty and normal pending entries would coexist or already coexist        |
