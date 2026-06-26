# Prefer testscript for CLI end-to-end tests

- Status: Accepted
- Created: 2026-06-26T08:29:34Z

## Context

`rellog` is a Go CLI tool whose important behavior is observable through
commands, exit codes, stdout/stderr, and repository file changes.

The project also expects AI-generated implementation to be common. In that
development style, the project should avoid relying too heavily on manual
inspection of implementation details. The preferred feedback loop is:

1. describe the expected behavior from the user's point of view;
2. run the CLI against a controlled file tree;
3. accept the implementation when the black-box behavior matches the
   specification.

For `rellog`, many specifications are naturally expressed as file-system
scenarios: initialize a repository, add entries, reject invalid layouts,
compare generated release-note files, and verify stable exit-code behavior.

The test foundation therefore needs to make CLI-level, fixture-heavy tests
cheap to write and maintain. It also needs to remain integrated with `go test`
so that coverage can be collected and used to check whether the e2e suite is
exercising the application broadly enough.

This decision should be recorded as an ADR because it defines the preferred
testing strategy for the project, influences how future behavior is specified,
and affects the dependency policy for the test harness.

## Decision

`rellog` will prefer `go test` with
`github.com/rogpeppe/go-internal/testscript` as the primary test foundation for
CLI end-to-end behavior.

CLI behavior should be tested as black-box scenarios whenever practical. A
testscript case should describe the initial file tree, execute `rellog`, and
assert observable results such as:

- exit status;
- stdout and stderr;
- created, changed, or missing files;
- generated release-note and changelog contents.

The e2e suite should be treated as the main acceptance signal for user-visible
CLI behavior. If the implementation is generated or substantially rewritten by
AI, passing e2e tests should carry more weight than matching a previously
expected internal structure.

Coverage should be collected from the `go test` run. Coverage is not itself a
correctness proof, but it should be used as a guardrail to see whether the e2e
suite is exercising the application as a whole rather than only a narrow path.

Ordinary Go `testing` remains allowed and useful for package-level tests,
especially pure functions, parsing helpers, and edge cases that are awkward or
too fine-grained to express as CLI scenarios. It is not the default choice for
CLI e2e behavior.

## Alternatives Considered

### Use only Go's standard testing package

This would keep dependencies minimal and use only the standard `testing`
package, plus custom helpers around temporary directories, `os/exec`, file
creation, stdout/stderr capture, exit-code checks, and golden-file comparison.

This was not selected as the primary e2e approach because it pushes the project
toward bespoke test infrastructure. For `rellog`, the repetitive work is
exactly the part `testscript` already models well: command execution inside a
temporary work directory with inline file fixtures and file comparisons.

### Use shell scripts directly

Shell scripts can run the built CLI and compare files with common Unix tools.

This was not selected because assertion quality, failure reporting, fixture
management, and cross-platform behavior are weaker. Shell scripts are also less
integrated with `go test` and Go coverage collection.

### Use Bats

Bats is useful for testing Unix command-line programs and can be a good fit for
release-smoke tests against a built binary.

This was not selected as the primary test foundation because it sits outside
the Go test runner. For `rellog`, integration with `go test`, coverage, and
Go-native fixtures is more valuable than a Bash-oriented test framework.

### Use Cucumber or another BDD framework

Cucumber-style tests can describe behavior in natural-language scenarios and
may be useful when non-technical stakeholders need to share executable
specifications.

This was not selected because `rellog` primarily needs precise CLI/file-system
acceptance tests, not a step-definition layer. Maintaining a BDD adapter would
add indirection without improving the core feedback loop.

## Consequences

### Positive Consequences

- User-visible CLI behavior can be specified as compact black-box scenarios.
- File fixtures, command execution, and output comparison stay close together.
- AI-generated implementations can be accepted based on observable behavior
  instead of internal shape.
- The main e2e suite runs under `go test` and can participate in normal Go
  coverage reporting.
- Future contributors have a clear default when adding tests for CLI behavior.

### Negative Consequences

- The project accepts a non-standard-library test dependency.
- Contributors need to learn the `testscript` DSL and txtar fixture format.
- Some fine-grained failures may still require package-level unit tests for
  fast diagnosis.

### Neutral Consequences

- Unit tests using ordinary Go `testing` remain appropriate for pure internal
  logic and narrow edge cases.
- Release-smoke tests against packaged binaries may still be added later with
  shell scripts or Bats if distribution validation becomes important.
- Coverage thresholds and reporting policy remain separate decisions.
