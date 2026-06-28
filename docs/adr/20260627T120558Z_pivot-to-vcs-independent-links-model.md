# Pivot to a VCS-independent links reference model

* Status: Accepted
* Created: 2026-06-27T12:05:58Z
* Supersedes: [Use JSON files for pending entries](20260626T120000Z_use-json-entry-files.md) for the entry reference fields

## Context

`rellog` was originally described as if a project had a GitHub remote and as if entry references were GitHub issues and pull requests. That made `rellog init` depend on repository state that is not essential to the tool.

The core job of `rellog` is to aggregate explicit human-written release entries into release-note files and `CHANGELOG.md`. GitHub issues and pull requests can be useful context, but they are not the domain model. Projects may want to reference GitLab issues, Forgejo pull requests, Slack messages, design notes, documentation, or arbitrary web pages.

The project is still before a stable release, so the public contract can pivot without carrying compatibility for the old issue and pull-request fields.

## Decision

`rellog` will be VCS-independent for v0.

`rellog init` must not require a Git repository, a configured remote, or a GitHub repository URL.

Configuration must not contain `github-url`.

Pending entries must use `links`, not `issues` or `prs`, for supporting references. `links` is a required array. Entries with no references must write `"links": []`.

In v0, every link must be an absolute URL with scheme `http` or `https` and a non-empty host. Query strings and fragments are allowed.

The CLI uses `--link <url>` for references. `--issue` and `--pr` are not compatibility options; they can fail as unknown options.

Rendering configuration is not exposed in v0. Release-note files start with `## <release-id>`. `CHANGELOG.md` is `# CHANGELOG` followed by prepared release-note sections. Kind sections use level 3 headings, entry metadata subsections use level 4 headings, and empty releases render `No changelog-worthy changes.`.

Normal entry bodies are emitted as raw Markdown inside rellog body marker comments. rellog does not escape, indent, list-wrap, code-block, normalize, or repair entry body Markdown.

`<!-- rellog:` is a reserved marker namespace. Entry bodies containing that marker prefix are invalid.

## Consequences

The configuration model is smaller and initialization can work in ordinary directories, Git repositories without remotes, and non-Git VCS projects.

GitHub issues and pull requests remain possible as ordinary `links`, but rellog does not normalize issue or pull request numbers and does not validate repository ownership.

Generated release notes can expose private URLs. Users must treat `links` as public-output candidates and avoid private references unless appropriate.

The v0 output contract is generated Markdown.
