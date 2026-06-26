# e2e tests

## Files

In this directory, e2e tests which are written by [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) are defined.

This directory must be flat. Even if `e2e/sub/test.txtar` would be created, the tests would not run.

## Naming

Pattern of e2e test file namings is:

`${subject}_(pos|neg)_${scenario}.txtar`

i.e.

- `init_pos_create_dir_and_config.txtar` : positive test for `rellog init` , it ensure the command to create necessary file and directories
- `add_neg_entries_not_directory.txtar` : negative test for `rellog add`, it verify the command's behavior when `.rellog/entries` is not directory
- `workflow_pos_empty_release.txtar` : positive test for workflow with `rellog`, it ensure the workflow to create empty release note

### Subject

At the head of e2e test file, it is recommended to write subject of the test.

The abstract word "subject" is used intentionaly.

It is allowed to use a command name as subject, or wide context.

i.e.

- `init_*` is test for `rellog init` command
- `workflow_*` is test for workflow which uses some `rellog` command

### Positive / Negative

| naming | role |
| --- | --- |
| `*_pos_*.txtar` | positive-case testing. Tests to ensure the system operates correctly |
| `*_neg_*.txtar` | negative-case testing. Tests to ensure failures are reported correctly |

## GitHub URL fixtures

Tests that validate configuration files should include `github-url
"https://github.com/tooppoo/rellog"` unless the scenario is specifically about
missing or invalid GitHub repository URLs.

Tests for issue and pull request references must treat GitHub URLs as syntax
only. They should not require network access or verify whether a referenced
issue or pull request exists on GitHub.
