# Use prepared release-note files as the v0 release gate

- Status: Proposed
- Created: 2026-06-26T00:00:00Z

## Context

`rellog` should prevent the most important failure mode: attempting to release before release notes have been prepared.

An earlier design considered a command such as:

```text
rellog require entries
```

This would fail when there are no pending entries.

However, requiring entries as an independent gate is not clearly necessary for v0. Pending entries are temporary preparation inputs. After release-note preparation, entries are consumed and the pending entry directory becomes empty again. It is also acceptable for entries to be added late, because the meaningful release explanation may only become clear after multiple pieces of work are considered together.

The more important v0 gate is whether a release-note file for the release id exists.

This decision should be recorded as an ADR because it defines where `rellog` enforces release workflow correctness.

## Decision

In v0, `rellog` will not provide `rellog require entries`.

Instead, `rellog prepare <release-id>` will fail if release-note preparation is impossible or unsafe, including when:

- there are no pending entries;
- normal entries and an empty entry coexist;
- pending entries are malformed;
- the target release-note file already exists;
- the release id is not path-safe.

For publish-oriented workflow steps, `rellog` will provide:

```text
rellog ready <release-id>
```

This command fails unless the prepared release-note file for the release id exists.

An empty release is represented by an explicit empty entry created with:

```text
rellog add-empty
```

This is not an `allow-empty` option or hidden bypass. It is a repository record meaning that there is nothing changelog-worthy for the next release.

## Alternatives Considered

### Provide `rellog require entries` in v0

This was not selected because there is no strong v0 use case for requiring pending entries outside release-note preparation. Ordinary development may legitimately have no pending entries. The meaningful release-note unit may emerge late.

### Use an `--allow-empty` option

This was not selected because it treats empty releases as a workflow exception rather than a repository record. An explicit empty entry is simpler: release-note preparation always consumes entries, whether normal or empty.

### Require entries on every pull request

This was not selected because it assumes that the pull request is the correct release-note unit. In practice, several pull requests may combine into one meaningful changelog entry, or one pull request may contain changes that are not changelog-worthy.

### Let publishing proceed without a release-note gate

This was not selected because it fails to address the primary omission `rellog` is meant to prevent: releasing before release notes have been prepared.

## Consequences

### Positive Consequences

- v0 has a small and clear guard model.
- Ordinary development is not blocked when pending entries are absent.
- Release-note preparation still fails if entries are absent or invalid.
- Publish-oriented workflows can reliably require a prepared release-note file.
- Empty releases are explicit repository states, not hidden workflow overrides.

### Negative Consequences

- Projects that want to enforce pending entries earlier will need to wait for a future command or implement a custom check.
- Some teams may prefer stricter PR-time or release-branch-time enforcement.

### Neutral Consequences

- `rellog require entries` may be added later if concrete use cases emerge.
- `rellog check` can still validate pending entries when they exist.
