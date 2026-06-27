# e2e tests

## Files

In this directory, e2e tests which are written by [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) are defined.

Tests are grouped by subject directory. The Go test harness must list each
subject directory that should be executed.

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

## Harness setup

The e2e harness must not create a Git repository, add a remote, or inject a
GitHub URL into the test workdir.

Tests that need Git repository metadata must run `git init` inside the
testscript. Tests that need a GitHub origin remote must also add that remote
inside the testscript.

This keeps VCS-independent scenarios honest: `rellog init` and the ordinary
entry workflow must not pass only because the harness secretly created a Git
repository.

## Pending contracts

When an e2e contract describes behavior that is specified but not implemented
yet, put a `skip` command at the beginning of the script:

```text
skip pending: #22 contract, enable in #23
```

Pending tests are committed as executable contracts but do not run in CI until
the implementation issue enables them.

## Links

Entry references use `links`.

`links` must be an array of absolute `http` or `https` URLs. Query strings and
fragments are allowed. The e2e fixtures must not require a GitHub remote or a
configured `github-url` for link validation.

Issue and pull request URLs may appear as ordinary link values. They must not be
treated as first-class entry fields.
