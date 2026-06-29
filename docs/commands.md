# rellog commands

This document lists the intended `rellog` CLI commands.

For file layout and entry file formats, see [files.md](files.md). For generated release-note structure, see [release-notes.md](release-notes.md). For lifecycle and workflow guards, see [workflow.md](workflow.md).

`rellog` is currently in early design. Command names and options may change before the first stable release, but the responsibility boundaries should remain stable: `rellog` manages changelog entries, plain Markdown release-note files, and `CHANGELOG.md`. It does not manage versions or publish releases.

## Command overview

```text
rellog init
rellog add
rellog add-empty
rellog check
rellog status
rellog ready <release-id>
rellog prepare <release-id>
```

## `rellog init`

Initialize `rellog` in the current directory.

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

When no flags are provided, `rellog add` should run in interactive mode. It should guide the user through the entry fields in this order:

1. `kind`
2. `target`
3. `body`
4. `links`

The interactive guide for `links` must say that the field may be left empty. It must also say that each value must be an absolute `http` or `https` URL, and that multiple values can be entered either as a comma-separated list or as a space-separated list. The guide is user-facing help text on stdout; tests should not rely on its exact wording.

Non-interactive usage:

```sh
rellog add \
  --kind changed \
  --target rellog \
  --link https://example.com/design/21 \
  --body "Added validation for pending changelog entries before release preparation."
```

Possible options:

```text
--kind <kind>      Changelog category, such as added, changed, fixed, removed, security.
--target <target>  Release target, component, or area affected by the change.
--breaking         Mark the entry as a breaking change.
--link <url>       Related URL. May be repeated.
--body <text>      Entry body. Useful for non-interactive use.
```

Rules:

- `rellog add` creates a normal JSON entry using the next generated UTC timestamp filename.
- If an empty entry already exists, `rellog add` should fail with `ExitEntryConflict`.
- `rellog add` should not silently remove an empty entry.
- Users cannot specify the entry filename.
- When any flag is provided, `rellog add` runs in non-interactive mode.
- Link values are written exactly as URL strings after validation; rellog does not contact the linked service.
- Link values may be rendered into public release notes and changelogs; avoid private URLs unless that exposure is acceptable.
- Entry JSON always includes `targets` and `links` as arrays. Empty fields are written as `[]`.
- Interactive and non-interactive mode must validate `kind` against `rellog.entries.kinds`. An undefined kind is an error and no entry file is created.
- Interactive and non-interactive mode must handle targets that are not listed in `rellog.entries.targets` according to `target-policy`:
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
- if a normal entry already exists, fail with `ExitEntryConflict`;
- the empty entry is a JSON entry with `targets` and `links` set to empty arrays;
- an entry with `kind: "empty"` and non-empty `targets` or `links` is invalid.

A normal entry and an empty entry should not coexist.

`rellog add` and `rellog add-empty` are the entry points that prevent users from creating an entry conflict through `rellog`.

## `rellog check`

Validate configuration and pending changelog entries.

```sh
rellog check
```

Expected checks:

- configuration file exists and is valid;
- pending entry files are parseable JSON;
- pending entry filenames follow the generated UTC timestamp format;
- required metadata is present;
- `targets` and `links` are present and are arrays;
- `targets` and `links` are empty arrays when `kind` is `empty`;
- entry kind is allowed;
- target is known, unless the project allows unknown targets;
- every link is an absolute `http` or `https` URL with a non-empty host;
- link query strings and fragments are accepted;
- body is not empty;
- body does not contain the reserved `<!-- rellog:` marker prefix;
- breaking-change metadata is consistent;
- normal entries and an empty entry do not coexist.

Invalid links include empty strings, whitespace-only strings, relative paths, strings without a scheme, schemes other than `http` or `https`, and URLs without a host.

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

## `rellog ready <release-id>`

Check that rellog-managed release artifacts are ready for publishing.

```sh
rellog ready v1.0.1
```

Successful human output is exactly one line:

```text
v1.0.1 release ready
```

This command is intended for publish-oriented release jobs. It is read-only and must not create, update, move, or remove files.

`ready` checks only files managed by rellog. It must not inspect or depend on external release state, including GitHub Releases, Git tags, package registry state, remote repository state, or consumed caches.

Expected checks:

- the current directory is a valid rellog project;
- the rellog config exists and can be parsed;
- the prepared release-note file for `<release-id>` exists under the configured release-note directory;
- the configured changelog file exists and contains a level 2 release heading for `<release-id>`;
- the configured pending entry directory contains no pending entry files.

`ready` must detect release headings from the fixed v0 release heading level `2`, rendered as `## <release-id>`. It must not use configurable heading settings.

`ready` must ignore headings inside rellog body marker comments when looking for release headings. A body that contains `## <release-id>` between `<!-- rellog:body:start -->` and `<!-- rellog:body:end -->` must not satisfy the changelog release-heading check.

If body marker comments are malformed, rellog treats the generated Markdown as invalid structure rather than guessing how to recover it.

Pending entry files are checked by presence only. `ready` does not need to parse or validate their JSON content.

For v0, release ids may contain path separators. Each path segment must be non-empty and must not be `.` or `..`. Normal dots inside a segment, such as `v1.0.1`, are allowed. See [files.md](files.md) for the release-id path rule.

If a release note exists but pending entries remain, `ready` should fail with recovery guidance because the pending entries may have been created by mistake, may need to be included in the current release, or may be intended for a future release.

Machine-readable output is available with `--json`:

```sh
rellog ready v1.0.1 --json
```

The JSON shape is defined by [`schema/ready-output.schema.json`](../schema/ready-output.schema.json).

Example ready JSON:

```json
{
  "ok": true,
  "releaseId": "v1.0.1",
  "releaseNote": ".rellog/release-notes/v1.0.1.md",
  "changelog": "CHANGELOG.md",
  "pendingEntries": [],
  "errors": []
}
```

Example not-ready JSON:

```json
{
  "ok": false,
  "releaseId": "v1.0.1",
  "releaseNote": ".rellog/release-notes/v1.0.1.md",
  "changelog": "CHANGELOG.md",
  "pendingEntries": [
    ".rellog/entries/fix-docs.json"
  ],
  "errors": [
    {
      "code": "pending_entries_present",
      "message": "Pending entries remain after the release note was prepared."
    }
  ]
}
```

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

- validate pending entries using the same checks and human-readable stderr diagnostics as `rellog check`;
- fail if there are no pending entries;
- fail with `ExitEntryConflict` if normal entries and an empty entry coexist;
- aggregate pending entries in filename order;
- reject release ids with empty path segments or dot-only segments (`.` or `..`);
- fail if the target release-note file already exists;
- without `--run`, print the generated release-note content and intended operations without changing files;
- with `--run`, create the release-note file for the release id;
- with `--run`, update `CHANGELOG.md` with the prepared release-note content;
- with `--run`, delete consumed pending entries.

Generated release-note content must follow [release-notes.md](release-notes.md).

Dry-run stdout is a human-readable preview. It contains the generated release-note Markdown followed by intended file operations:

```text
## v1.0.1

### Changed

#### Details

<!-- rellog:body:start -->
Added validation for pending changelog entries before release preparation.
<!-- rellog:body:end -->

#### Targets

- rellog

#### Links

- https://example.com/design/21
create .rellog/release-notes/v1.0.1.md
update CHANGELOG.md
delete .rellog/entries/20260626T123456.123456789Z.json
```

When `--run` succeeds, stdout is exactly one line:

```text
v1.0.1 release prepared
```

Successful dry-run and `--run` executions write nothing to stderr.

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

For v0, release ids are used as paths below the configured release-note directory. Path separators are allowed so projects can group release-note files. Each path segment must be non-empty and must not be `.` or `..`. Normal dots inside a segment, such as `v1.0.1`, are allowed. See [files.md](files.md) for the release-id path rule.

When `CHANGELOG.md` already exists, `--run` inserts the release section at the top of the file. If the file starts with an H1 such as `# CHANGELOG`, `--run` inserts the release section directly below that H1 instead of duplicating it. Release-note files and `CHANGELOG.md` must end with a trailing newline.

If manual file edits create an entry conflict, `rellog prepare <release-id>` and `rellog prepare <release-id> --run` should fail before writing a release-note file, updating `CHANGELOG.md`, or deleting pending entries.

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
| 0    | -                      | Success                                                                 |
| 1    | `ExitNotInitialized`   | `rellog` has not been initialized; run `rellog init` first              |
| 2    | `ExitInvalidStructure` | A path that must be a directory exists as a file (e.g. `.rellog/entries`) |
| 3    | `ExitCheckFailed`      | `rellog check` found one or more non-conflict validation errors in pending entries |
| 4    | `ExitReleaseNotFound`  | The required release-note file does not exist; run `rellog prepare <release-id> --run` first |
| 5    | `ExitEntryConflict`    | Empty and normal pending entries would coexist or already coexist        |
| 6    | `ExitNotGitRepo`       | The current directory is not inside a Git repository                    |
| 7    | `ExitInvalidArgument`  | CLI usage or an argument such as `<release-id>` is invalid               |
| 8    | `ExitReleaseNotReady`  | A release note exists, but changelog or pending-entry readiness checks failed |
