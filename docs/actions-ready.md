# rellog ready GitHub Action

`tooppoo/rellog/actions/ready` installs the `rellog` CLI by running the
repository's [`install.sh`](../install.sh) with the resolved version, which
downloads the release archive, verifies it against the same release's
`checksums.txt`, and extracts the binary. It then runs:

```sh
rellog ready <release-id>
```

The action is a release-readiness wrapper only. It does not run `rellog
prepare`, does not run `rellog replace`, does not create or update rellog-managed
files, and does not commit or push.

## Usage

Check out the caller repository before running the action so the runner has the
repository's `.rellog/` directory and `CHANGELOG.md`.

```yaml
- uses: actions/checkout@v7

- uses: tooppoo/rellog/actions/ready@v0.0.4
  with:
    release-id: v1.2.0
```

If the action is not referenced by a version tag, set the CLI version explicitly:

```yaml
- uses: tooppoo/rellog/actions/ready@main
  with:
    release-id: v1.2.0
    version: v0.0.4
```

## Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `release-id` | yes | | Release id passed to `rellog ready`. |
| `version` | no | | GitHub Release tag of the `rellog` CLI to install. |
| `working-directory` | no | `.` | Directory, relative to `github.workspace`, where `rellog ready` runs. |

## Version resolution

When `version` is set, the action installs `rellog` from that GitHub Release tag.

When `version` is omitted, the action uses `github.action_ref` only if it is a
version tag such as `v0.1.0`. The action never falls back to `latest`. If the CLI
version cannot be resolved, the action fails and asks the caller to set
`version`.

The resolved release tag is passed as `install.sh --version <tag>`, which
resolves the platform archive, downloads it from the matching GitHub Release,
and verifies it against that release's `checksums.txt`.

## Supported runners

The action supports the same platforms as [`install.sh`](../install.sh):

| `runner.os` | `runner.arch` |
| --- | --- |
| `Linux` | `X64` |
| `Linux` | `ARM64` |
| `macOS` | `X64` |
| `macOS` | `ARM64` |

`Windows` runners are not supported. Other runner OS or architecture
combinations fail before any archive download.

## Failure behavior

The action fails before running `rellog ready` when:

- `version` is omitted and the action ref is not a version tag;
- `install.sh` fails to install the resolved version, for example because:
  - the selected GitHub Release or platform archive does not exist;
  - the runner OS or architecture is unsupported;
  - archive download fails;
  - `checksums.txt` is missing, unreadable, malformed, missing the selected
    archive, or has more than one entry for the selected archive;
  - the archive checksum does not match `checksums.txt`;
  - archive extraction fails;
- `working-directory` is absolute, resolves outside `github.workspace`, does not
  exist, or is not a directory.

After installation and working-directory validation succeed, the action runs only
`rellog ready <release-id>`. If that command fails, the action fails with the
CLI's output and exit code.
