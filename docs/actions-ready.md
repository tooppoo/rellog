# rellog ready GitHub Action

`tooppoo/rellog/actions/ready` installs the `rellog` CLI from this action
repository's GitHub Releases, verifies the downloaded archive against the same
release's `checksums.txt`, and runs:

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

- uses: tooppoo/rellog/actions/ready@v0.1.0
  with:
    release-id: v1.2.0
```

If the action is not referenced by a version tag, set the CLI version explicitly:

```yaml
- uses: tooppoo/rellog/actions/ready@main
  with:
    release-id: v1.2.0
    version: v0.1.0
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

The release tag resolves the GitHub Release. The archive name follows the
repository's GoReleaser configuration:

```text
rellog_v<version>_<Os>_<arch>.tar.gz
rellog_v<version>_<Os>_<arch>.zip
checksums.txt
```

For a release tag such as `v0.1.0`, the archive version segment is `0.1.0`, so
the Linux x64 archive is `rellog_v0.1.0_Linux_x86_64.tar.gz`.

## Supported runners

| `runner.os` | `runner.arch` | Asset suffix |
| --- | --- | --- |
| `Linux` | `X64` | `Linux_x86_64.tar.gz` |
| `Linux` | `ARM64` | `Linux_arm64.tar.gz` |
| `macOS` | `X64` | `Darwin_x86_64.tar.gz` |
| `macOS` | `ARM64` | `Darwin_arm64.tar.gz` |
| `Windows` | `X64` | `Windows_x86_64.zip` |
| `Windows` | `ARM64` | `Windows_arm64.zip` |

Other runner OS or architecture combinations fail before any archive download.

## Failure behavior

The action fails before running `rellog ready` when:

- `version` is omitted and the action ref is not a version tag;
- the selected GitHub Release does not exist;
- the platform archive asset does not exist;
- the runner OS or architecture is unsupported;
- archive download fails;
- `checksums.txt` is missing, unreadable, malformed, missing the selected
  archive, or has more than one entry for the selected archive;
- the archive checksum does not match `checksums.txt`;
- archive extraction fails;
- the archive does not contain `rellog` or `rellog.exe`;
- `working-directory` is absolute, resolves outside `github.workspace`, does not
  exist, or is not a directory.

After installation and working-directory validation succeed, the action runs only
`rellog ready <release-id>`. If that command fails, the action fails with the
CLI's output and exit code.
