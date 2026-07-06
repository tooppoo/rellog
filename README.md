# rellog

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/tooppoo/rellog/actions/workflows/ci.yml/badge.svg)](https://github.com/tooppoo/rellog/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/tooppoo/rellog/graph/badge.svg?token=E8b5Wgllwi)](https://codecov.io/gh/tooppoo/rellog)
[![CodeQL](https://github.com/tooppoo/rellog/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/tooppoo/rellog/actions/workflows/github-code-scanning/codeql)
[![Dependency Graph](https://github.com/tooppoo/rellog/actions/workflows/dependabot/update-graph/badge.svg)](https://github.com/tooppoo/rellog/actions/workflows/dependabot/update-graph)

Git log is not a release log.

`rellog` is a VCS-independent CHANGELOG and release-note-file management tool.

It is inspired by the [changesets](https://github.com/changesets/changesets) workflow, but it is not compatible with `changesets` and does not manage versions. Its scope is limited to collecting explicit changelog entries, preparing plain Markdown release-note files, and updating `CHANGELOG.md`.

## Quick Start

```sh
curl -fsSL https://raw.githubusercontent.com/tooppoo/rellog/refs/heads/main/install.sh | sh
```

## Position

`rellog` is based on a small distinction: Git history records how work happened, while changelogs and release notes explain what changed.

It does not try to infer final release notes from commits, pull requests, issues, or Conventional Commits. Instead, it expects explicit changelog entries written by humans at the level the change should be communicated.

It also leaves version numbers, package manifests, tags, publishing, and GitHub Releases to the tools and conventions of each project. A release id is supplied from the outside when release notes and `CHANGELOG.md` are prepared.

The background for this position is described in [docs/philosophy.md](docs/philosophy.md).

## Concept

The central unit is a changelog entry: a small JSON file that describes a change at the level it should be communicated in release notes. Entries are accumulated before release preparation and then consumed into a release-note file and `CHANGELOG.md`.

`rellog add-empty` can create an explicit empty changelog entry when there is nothing changelog-worthy to mention. This lets normal releases and intentionally empty releases follow the same preparation flow.

Release notes prepared by `rellog` are plain Markdown files under `.rellog/release-notes/`. They are project-managed artifacts that can be inserted into `CHANGELOG.md`, reviewed, and reused by other release tooling.

Entries can include `links`: explicit `http` or `https` references to issue trackers, pull requests, design notes, messages, documentation, or other supporting context. Links are optional per entry but the `links` key is always present. Because generated release notes and changelogs may be public, private URLs should be added only when that is appropriate for the project.

## Basic workflow

1. Add a changelog entry while implementing, reviewing, or finalizing a change.
2. If there is nothing to mention, add an explicit empty entry.
3. Prepare release notes for a supplied release id.
4. Update `CHANGELOG.md` from the prepared release notes.
5. Remove consumed pending entries.
6. Before publishing, require the prepared release-note file for the release id.

See [docs/workflow.md](docs/workflow.md) for the intended workflow.

## What rellog does

`rellog` is intended to:

- initialize `.rellog/`;
- create changelog entry files;
- create an explicit empty entry when there is nothing to mention;
- validate entry format and required metadata;
- list pending entries;
- reject release-note preparation when there are no pending entries;
- prepare `.rellog/release-notes/<release-id>.md` from pending entries;
- update `CHANGELOG.md` with prepared release notes;
- require a prepared release-note file for a release id;
- support GitHub Actions that create CHANGELOG update pull requests.

## What rellog does not do

`rellog` does not:

- decide the next version;
- update `package.json`, `Cargo.toml`, `pyproject.toml`, or any other package manifest;
- create Git tags;
- publish packages, binaries, or artifacts;
- create GitHub Releases;
- treat `.rellog/release-notes/*.md` as GitHub Release Notes;
- generate final release notes directly from commit history;
- require Conventional Commits;
- provide compatibility with Changesets file semantics.

## When not to use rellog

`rellog` is probably not the right tool when:

- commit history is already clean enough to be the primary changelog source;
- the project wants Conventional Commits based generation;
- version bumping and package publishing should be managed by the same tool;
- the project needs Changesets compatibility;
- the team wants fully automatic release-note inference without explicit changelog entries.

In those cases, commit-history based changelog generators or ecosystem-specific release automation may be a better fit.

## Documentation

- [Commands](docs/commands.md)
- [Configuration](docs/configuration.md)
- [GitHub Action: ready](actions/ready/READMD.md)
- [GitHub Action: create release note](actions/create-release-note/README.md)
- [Workflow](docs/workflow.md)
- [Git Ignore policy](docs/ignore-policy.md)
- [Files](docs/files.md)
- [Generated release notes](docs/release-notes.md)
- [Philosophy](docs/philosophy.md)

## Project status

`rellog` is currently in early design. The initial goal is a small, VCS- and runtime-independent CLI for managing changelog entries, preparing plain Markdown release-note files, and updating `CHANGELOG.md`.
