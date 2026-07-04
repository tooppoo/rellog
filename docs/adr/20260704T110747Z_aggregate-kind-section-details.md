
# Aggregate Details/Targets/Links once per kind section

- Status: Accepted
- Created: 2026-07-04T11:07:47Z

## Context

`rellog prepare` and `rellog amend` render one `#### Details`, `#### Targets`, and
`#### Links` block per entry. When a kind section (e.g. `### added`) contains more than
one entry, this produces the same heading text repeated multiple times in a row, as seen
in `.rellog/release-notes/v0.1.0.md`: three `#### Details` headings and two `#### Targets`
headings directly under one `### added` heading, with the target `cli` listed twice.

This was previously documented as intentional (`docs/release-notes.md`: "Targets and
links belong to each entry. They are not aggregated into release-wide sections."), but
reads as a defect to users: a same-level, same-text heading repeated multiple times under
one parent section, and a target/link value duplicated verbatim, rather than a merged
release note.

This must be an ADR because it changes a documented output contract
(`docs/release-notes.md`) that other tooling may parse, and it changes how `rellog
amend`'s append mode (the fallback used when the consumed entry cache is unusable) must
reconstruct a kind section's content.

## Decision

A kind section must emit `#### Details` at most once, `#### Targets` at most once, and
`#### Links` at most once, regardless of how many entries the kind section contains.

- `#### Details` contains one marker-delimited body block per contributing entry, in
  filename order, separated by one blank line.
- `#### Targets`, if any entry has targets, lists the union of every contributing entry's
  targets, in first-seen order (entries in filename order; within one entry, that entry's
  own list order), with duplicates removed.
- `#### Links` follows the same union/first-seen/dedup rule as `#### Targets`.

`rellog amend`'s append-mode path (used when the consumed cache is unusable, so the
original entries backing an existing kind section are not available) must recover enough
structure from the already-rendered kind section — its body blocks, target list, and link
list — to merge new entries in and re-render the whole section, rather than appending a
new repeated heading block as before.

## Alternatives Considered

### Keep per-entry `#### Details`, aggregate only `#### Targets`/`#### Links`

Rejected: this still repeats the same-text `#### Details` heading once per entry, which
is the most visible instance of the reported problem.

### Aggregate everything into a single blob (drop per-entry body boundaries)

Rejected: losing the per-entry marker-delimited boundary would make it impossible to tell
where one entry's body ends and the next begins, and would break `rellog amend`'s ability
to merge new entries into an existing section without re-deriving entry boundaries from
free text.

## Consequences

### Positive Consequences

- A kind section with multiple entries no longer repeats the same heading text, and a
  target/link shared by multiple entries appears once.
- The rendered release note reads as one merged section per kind, matching how changelogs
  are normally written by hand.

### Negative Consequences

- `rellog amend`'s append-mode splicer is more complex: instead of appending a rendered
  entry block at the end of a kind section, it must parse the existing aggregated section
  back into bodies/targets/links, merge in the new entries, and re-render the whole
  section in place.
- The already-released `.rellog/release-notes/v0.1.0.md` and its `CHANGELOG.md` section
  needed a one-time re-render from `.rellog/consumed/v0.1.0/` to stay consistent with the
  new format; otherwise a future `rellog amend v0.1.0` would fail the regenerate-mode
  byte-comparison against the old per-entry format.

### Neutral Consequences

- `docs/release-notes.md` no longer states that targets and links belong exclusively to
  each entry; it now documents the per-kind-section union/dedup rule instead.
