
# Exit Code Policy

- Status: Accepted
- Created: 2026-06-25T17:06:50Z

## Context

`rellog` is a CLI tool intended for use in CI pipelines and shell scripts.
Callers need to distinguish error categories programmatically, without parsing
human-readable error messages.

Without a stable exit code policy, callers must either ignore the exit code
entirely or parse stderr output, both of which are fragile and couple callers
to implementation details.

## Decision

`rellog` must use distinct, named exit codes for each error category.
Exit codes must be stable across patch releases.

Current assignments:

| Code | Constant               | Meaning                                                         |
|------|------------------------|-----------------------------------------------------------------|
| 0    | —                      | Success                                                         |
| 1    | `ExitNotInitialized`   | `rellog` has not been initialized; `rellog init` must be run first |
| 2    | `ExitInvalidStructure` | A path that must be a directory exists as a file (e.g. `.rellog/entries`) |

Rules:

- Exit code `0` must mean success.
- Each error category must have its own code.
- Codes must not be reused for a different error category within the same major version.
- New error categories must be assigned a new code; they must not reuse an existing code.
- Codes must be defined as named constants in the Go package so that call sites reference the constant, not a bare integer.
- The mapping of codes to categories must be kept in sync across `rellog.go` and `docs/commands.md`.

## Alternatives Considered

### Single non-zero code for all errors

All errors return exit code `1`. Callers distinguish errors by parsing stderr.

Not selected because stderr format is not a stable contract and is subject to
change without notice. Parsing error messages couples callers to phrasing and
localization.

### Exit code reflects error severity

Group errors by severity (warning, error, fatal) rather than by category.

Not selected because severity does not give callers enough information to
decide whether to retry, fall back, or abort. Category-based codes allow
callers to handle `ExitNotInitialized` differently from `ExitInvalidStructure`
without inspecting message text.

## Consequences

### Positive Consequences

- CI scripts and pipeline steps can branch on specific error categories without parsing stderr.
- The exit code contract is visible in `docs/commands.md` and can be referenced in external documentation.
- Named constants in Go prevent accidental reuse of a numeric value.

### Negative Consequences

- The code space must be managed as the command set grows; codes must not be reassigned.
- Any new error category requires updating both the implementation and `docs/commands.md`.

### Neutral Consequences

- Exit code `1` is conventionally "generic failure" in Unix. Using it for `ExitNotInitialized` is slightly non-standard but does not conflict with shell convention, which treats any non-zero code as failure.
