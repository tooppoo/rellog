# Treat release note targets as strict structural vocabulary

- Status: Accepted
- Created: 2026-07-06T01:31:25Z

## Context

Release note rendering previously emitted each entry as a block of metadata subsections: `#### Details` for the body, `#### Targets` for the target list, and `#### Links` for references. When multiple entries were prepared or amended into the same release, these headings repeated per entry, making the generated release note read like a dump of entry metadata rather than a reader-oriented release note.

At the same time, entry targets were only loosely validated. `entries.target-policy` selected one of `deny-unknown`, `warn-unknown`, or `allow-unknown`, so depending on configuration an arbitrary, undeclared target string could flow into generated output.

Restructuring the release note around targets makes targets part of the generated Markdown heading structure. Once a target renders as a public heading, a typo in a target is no longer private metadata — it becomes public release-note structure. That changes what level of validation targets require.

This decision affects public output formats and the configuration schema, so it is recorded as an ADR.

## Decision

Release notes group entries as `release -> kind -> target section -> entry`:

- Kind sections keep their `### <kind title>` headings.
- Targets are rendered as `#### <target-set title>` subsections directly under kind sections. The heading uses the effective target title (`target.title` if configured, otherwise the target id).
- An entry with multiple targets renders once, under a combined target-set heading (effective titles joined by ` / `, in `entries.targets` declaration order). Entry bodies are never duplicated across target sections.
- `Details`, `Targets`, and `Links` headings are no longer emitted.
- Links render under the plain `Refs:` label, not a Markdown heading, and only when the entry has links.
- Entry bodies remain wrapped by `<!-- rellog:body:start -->` / `<!-- rellog:body:end -->`.
- Each rendered entry is additionally wrapped by `<!-- rellog:entry:start -->` / `<!-- rellog:entry:end -->`. The entry markers preserve machine-readable entry boundaries that the removed `#### Details` headings used to provide.

Targets are strict structural vocabulary, validated the same way as kinds:

- `entries.targets` is always required and must declare at least one target.
- Every normal entry must declare at least one target.
- Unknown (undeclared) entry targets are errors, for the same quality reason unknown kinds are errors: they would otherwise become public release-note structure.
- `entries.target-policy` is removed from the configuration schema. `warn-unknown` and `allow-unknown` are not supported, and a config that still contains a `target-policy` node is rejected as unknown configuration. No compatibility shim is kept.

## Alternatives Considered

### Keep per-entry metadata subsections and only deduplicate headings

Aggregating repeated `Details`/`Targets`/`Links` headings per kind section would fix the repetition, but the output would still be entry metadata rendered as document structure, not a reader-oriented note. It also keeps targets as second-class metadata while they already describe the most reader-relevant grouping after kind.

### Keep `target-policy` and only default it to `deny-unknown`

A lenient mode could remain available for projects that do not want to maintain a target vocabulary. Rejected because targets now render as headings: any accepted unknown target immediately becomes public structure, so `warn-unknown` and `allow-unknown` would silently let typos ship. Kinds already follow the strict model; targets get the same rule for the same reason.

### Duplicate multi-target entries into each target's section

Rendering one copy per target would keep sections single-target, but duplicated bodies read as duplicated changes and make the changelog longer and wrong for counting. A combined target-set section keeps each change stated once.

## Consequences

### Positive Consequences

- Generated release notes are organized for readers: kind first, then affected area, then the change itself.
- Target typos are caught at entry creation and check time instead of shipping as public headings.
- `Refs:` keeps references attached to their entry without promoting them to document structure.
- Entry markers give tools a stable way to locate entry boundaries after metadata headings were removed.

### Negative Consequences

- Projects using `target-policy` must update their configuration: remove the node, declare their targets, and fix entries with missing or undeclared targets. No migration shim is provided.
- Previously generated release notes and consumed caches use the old structure; `amend` regenerate-mode comparisons against old-format artifacts fail, falling back to append mode or requiring manual reconciliation.
- Every entry must now carry at least one declared target, which adds a small upfront vocabulary-maintenance cost to `entries.targets`.

### Neutral Consequences

- `entries.targets` declaration order becomes structurally meaningful (combined-heading order).
- The entry JSON schema is unchanged ([Use JSON entry files](20260626T120000Z_use-json-entry-files.md)); this decision only tightens the semantics of the existing required `targets` array for normal entries.

## Superseded ADRs

No prior ADR covered target policy, target rendering, or entry metadata rendering; no ADR is superseded by this decision. The entry-file schema ADR ([Use JSON entry files](20260626T120000Z_use-json-entry-files.md)) remains in force unchanged.
