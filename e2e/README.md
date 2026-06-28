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

The e2e harness must not create hidden project state in the test workdir.

Tests that need Git repository metadata must run `git init` inside the
testscript. Tests that need an origin remote must also add that remote inside
the testscript.

This keeps tests explicit: commands must not pass only because the harness
secretly created unrelated state.

## Links

Entry references use `links`.

`links` must be an array of absolute `http` or `https` URLs. Query strings and
fragments are allowed. Link validation must depend only on the link value.

Project-management URLs may appear as ordinary link values. They must not be
treated as first-class entry fields.
