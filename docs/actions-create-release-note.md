# rellog create release note GitHub Action

`tooppoo/rellog/actions/create-release-note` creates a GitHub Release using a
rellog-generated release note as the release body. It reads
`.rellog/release-notes/<release-id>.md`, strips the leading version header
line (the automatically generated `## <release-id>` line, since GitHub uses
the tag/release name as the title already), and passes the remainder to
[`softprops/action-gh-release`](https://github.com/softprops/action-gh-release)
as `body_path`.

The action is a release-creation wrapper only. It does not build or upload
release assets, and does not run `rellog ready`, `rellog prepare`, or `rellog
replace`.

## Usage

Check out the caller repository before running the action so the runner has
the repository's `.rellog/release-notes/` directory. The calling job must
grant `contents: write` permission (or supply an equivalent PAT via
`GITHUB_TOKEN`), because `softprops/action-gh-release` needs it to create the
release.

```yaml
permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - uses: tooppoo/rellog/actions/create-release-note@v0.0.4
        with:
          release-id: v1.2.0
```

## Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `release-id` | yes | | Release id to publish, in `vMAJOR.MINOR.PATCH` form (with optional pre-release/build metadata). Used as the release tag and to locate `.rellog/release-notes/<release-id>.md`. |
| `working-directory` | no | `.` | Directory, relative to `github.workspace`, containing the `.rellog` directory. |

## Failure behavior

The action fails before creating a release when:

- `release-id` does not match `vMAJOR.MINOR.PATCH` (optionally followed by
  `-` or `+` metadata), for example `v1.2.0` or `v1.2.0-rc.1`;
- `working-directory` is absolute, resolves outside `github.workspace`, does
  not exist, or is not a directory;
- `.rellog/release-notes/<release-id>.md` does not exist under
  `working-directory`.

After the release note is prepared, the action delegates to
`softprops/action-gh-release`. If that step fails, for example because the
caller lacks `contents: write` permission, the action fails with that step's
output.
