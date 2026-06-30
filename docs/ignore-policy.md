# Ignore policy

This document defines which `rellog` files should be committed and which files may be ignored.

`rellog` is VCS-independent, but the files it manages are normally stored in a project repository. A project may use Git, another VCS, or a generated source package. The same policy applies: persistent release inputs and outputs should be kept, while reproducible cache data may be excluded.

## Do not ignore

Do not ignore these files:

- `.rellog/config.kdl`
- `.rellog/entries/*.json`
- `.rellog/release-notes/**/*.md`
- `CHANGELOG.md`

These paths may be customized in `.rellog/config.kdl`. If a project uses custom paths, apply the same policy to the configured paths.

## Why these files are committed

`.rellog/config.kdl` defines the project policy for entries, release notes, changelog updates, known entry kinds, known targets, and consumed-cache failure behavior. It should be reviewed with the project.

`.rellog/entries/*.json` files are pending changelog entries. They are release-note inputs written by humans at the level the change should be communicated.

`.rellog/release-notes/**/*.md` files are prepared release-note artifacts. They are reviewed release outputs and may be reused by external release tooling.

`CHANGELOG.md` is the cumulative release record updated from prepared release notes.

Ignoring any of these files can make a release look ready or empty when the project has not actually recorded, prepared, or reviewed the intended release content.

## May ignore

Projects may ignore `.rellog/consumed/`.

`.rellog/consumed/` stores consumed cache data created by `rellog prepare <release-id> --run`. It preserves the exact entry set used to prepare a release for later commands that need to reconstruct it, but it is not the source of truth for release readiness.

For example, this is acceptable in `.gitignore`:

```gitignore
.rellog/consumed/
```

Ignoring consumed data means later commands cannot rely on that local cache unless it is regenerated or otherwise restored. It does not change the committed release-note file, `CHANGELOG.md`, or the pending-entry policy.

## Typical `.gitignore`

A project that uses the default `rellog` paths may include:

```gitignore
.rellog/consumed/
```

It should not include broader patterns such as:

```gitignore
.rellog/
.rellog/entries/
.rellog/release-notes/
CHANGELOG.md
```
