# rellog Philosophy

rellog does not assume that changelogs or release notes can be reliably reconstructed from commit logs.

A commit log is a record of work.
A changelog or release note is a record of meaningful change.

These two records are related, but they are not the same.

rellog separates work history from change explanation.
Work history belongs in the commit log.
The meaning of a change should be written by humans, at the appropriate level of detail.

## Core principle

Work history should remain work history.
The meaning of change should be written as the meaning of change.

Changelog entries do not need to be written for every commit.
They do not need to be written for every pull request.
They may be written later, when they become necessary.

But they must not be forgotten at release time.

rellog exists to support that single point.

## Commit logs are not reliable primary sources for changelogs

Commit logs are not reliable primary sources for changelogs.

They often contain implementation adjustments, follow-up fixes, CI changes, minor documentation edits, refactorings, and traces of trial and error.
These details may be useful as development history, but they are not necessarily appropriate for release notes.

Pull requests and commits are also often split into small units to make development smoother.
As a result, commit logs tend to be more granular than changelogs need to be.

The opposite can also happen.
Several commits may only become meaningful when understood together as a single change.
In such cases, reading individual commits does not clearly explain the actual change.

A commit log is an accumulation of work.
A changelog needs an accumulation of meaning.

For this reason, rellog does not treat commit logs as the primary source for changelogs.

## The people who made the change usually know what changed

The general meaning of a change is often best understood by the people who implemented it, requested it, or decided its direction.

Those people can describe the change at the right level of detail.
This is often faster, more accurate, and less noisy than mechanically aggregating commit logs.

For simple release notes, humans can often summarize the changes directly.
For detailed release notes, commit logs are usually not enough.
Issues, pull requests, design notes, ADRs, documentation, and discussion history may all be necessary.

In either case, commit logs are a weak center of gravity.

rellog assumes that the meaning of a change should be written by humans.

## Commit discipline is not always sustainable

Conventional Commits is a useful convention.
It gives structure to commit messages and makes change types easier to process mechanically.

However, rellog does not make changelog quality depend on commit message discipline.

Conventional Commits attempts to treat commit logs not merely as work history, but as accumulated change knowledge.
This is a reasonable idea, but it is not always sustainable in real development.

When people are rushed, tired, interrupted, or working through uncertain changes, commit messages easily become imprecise.
That is not merely a moral failure.
It is a normal imperfection of real development work.

AI may be able to follow commit conventions to some extent.
But it is not guaranteed.
There is also a separate question: whether AI tokens and attention should be spent on shaping commit messages primarily for changelog generation.

rellog does not try to turn commit logs into a knowledge base.
Work history may remain work history.
Information needed for changelogs should be written elsewhere, at a different granularity.

## Inspired by changesets

rellog is strongly influenced by changesets.

changesets is a valuable tool because it asks humans to write change descriptions and then aggregates those descriptions later.
This idea is close to rellog: release notes should be based on human-written change explanations, not inferred from commit logs.

However, rellog intentionally has a narrower scope.

changesets is mainly used in the Node.js and JavaScript ecosystem, and it also includes version management features.
It is possible to use it in non-Node projects by adding Node.js to the environment, but some projects should not need Node.js only to manage changelog entries.

Version management also differs significantly across ecosystems.
npm, Cargo, Go modules, NuGet, PyPI, and GitHub Releases all have different assumptions and conventions.

Therefore, rellog does not manage versions.
Version numbers, tags, package publishing, and release procedures should be handled by the standard tools and conventions of each ecosystem.

rellog only handles the preparation of human-written change explanations and verifies that they exist at release time.

## Configuration should be small, readable, and commentable

rellog uses KDL for configuration.

Configuration files should support comments.
Configuration is not only a collection of values.
It is also a place to record why certain choices were made and what each setting is intended to express.

JSON is widely used, but it does not support comments.
It also tends to be verbose as a configuration format.

JSONC supports comments, but it is not JSON itself.
It also does not solve JSON's verbosity.

TOML is simple and practical, but nested structures can become awkward.
When namespaces are different, rellog prefers expressing that structure directly rather than flattening it through naming conventions.

YAML is a strong candidate, but it is too feature-rich for rellog's needs.
Its specification is complex, and parser behavior can vary.
rellog should not need to depend heavily on parser-specific behavior for a configuration file.

KDL supports comments.
It represents nesting naturally.
Its specification is comparatively small.

rellog adopts KDL because it provides the expression needed for configuration without introducing unnecessary complexity.

## Designed for forgetful humans

rellog does not require changelog entries to be written for every commit or pull request.

Write them when you remember.
Write them when you notice.
Write them later, when release notes become necessary.

Development should not be constantly interrupted by changelog maintenance.
Developers should not need to maintain a perfectly structured change history throughout the entire development process.

It is acceptable to write change entries shortly before release.
The people involved can review closed issues, pull requests, design notes, and documentation, then summarize only what matters.

rellog assumes that writing a changelog is not the core difficulty.
The core difficulty is forgetting to write one.

## Release time is the only strict point

rellog is permissive during development.
It does not enforce continuous changelog maintenance.

Release time is different.

If a release is being made, the material for the changelog or release notes must exist.
rellog provides a guard for that point.

When rellog is integrated into a release flow, such as GitHub Actions, it can stop the release job if release notes have not been prepared.

The remedy is simple: write the missing change entries, then run the release job again.

rellog does not demand constant discipline.
It only requires that the necessary change explanation exists at release time.

## An empty changelog is a valid state

rellog officially accepts an empty changelog.

Sometimes a release contains no changes worth mentioning in a changelog.
Internal updates, minor documentation changes, build configuration adjustments, or dependency updates may not require user-facing release notes.

In such cases, there is no need to pretend that a meaningful change exists.
It is enough to explicitly state that there are no changes worth recording for the release.

rellog treats that as a valid state.

The important point is not that nothing changed.
The important point is that the decision to record nothing is explicit.

## What rellog solves

rellog does not try to generate changelogs automatically.
It does not try to reconstruct the meaning of changes from commit logs.
It does not enforce perfect commit message conventions.

rellog solves one specific problem: forgetting to prepare changelog or release note material before release.

Humans write the meaning of changes.
They write it when needed, at the necessary level of detail.
If there is nothing to write, they explicitly mark the release as empty.

rellog verifies that this record exists at release time.
It tries to do as little else as possible.

## Non-goals

rellog is not a tool for converting commit logs into canonical changelogs.

rellog is not a replacement for Conventional Commits.

rellog is not a versioning tool.

rellog is not a package publishing tool.

rellog is not a tool that fully reconstructs the meaning of changes without human judgment.

rellog is not a tool that requires constant discipline from developers.

rellog is a small harness that verifies the existence of release-note material at the moment it matters.
