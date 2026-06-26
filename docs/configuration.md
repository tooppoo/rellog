# Configuration

rellog reads its repository configuration from `.rellog/config.kdl`.

The configuration file defines repository-level policies used by rellog commands, such as:

* where changelog-related files are stored
* which GitHub repository issue and pull request references belong to
* which entry kinds are allowed
* which entry targets are known
* how release notes and changelog sections are rendered

The configuration file is written in [KDL v2](https://github.com/kdl-org/kdl/blob/main/draft-marchan-kdl2.md).

## Example

```kdl
/- kdl-version 2

rellog config-version=1 {
  github-url "https://github.com/tooppoo/rellog"
  paths {
    changelog "CHANGELOG.md"
    entries ".rellog/entries"
    release-notes ".rellog/release-notes"
  }

  entries {
    target-policy "deny-unknown"

    kinds {
      kind "added" description="New user-visible functionality."
      kind "changed" description="Changes to existing user-visible behavior."
      kind "deprecated" description="Features that will be removed in a future release."
      kind "removed" description="Removed features."
      kind "fixed" description="Bug fixes."
      kind "security" description="Security-related fixes."
    }

    targets {
      target "rellog" description="Changes to rellog itself."
    }
  }

  rendering {
    release-heading-level 2
    section-heading-level 3
    empty-message "No changelog-worthy changes."
  }
}
```

## Document structure

A configuration file must contain exactly one top-level `rellog` node.

```kdl
rellog config-version=1 {
  github-url "https://github.com/tooppoo/rellog"
  ...
}
```

The `rellog` node must have the `config-version` property described under [Property](#property).

The KDL version marker is optional, but `rellog init` should write it.

```kdl
/- kdl-version 2
```

If a KDL version marker is present, it must specify KDL version 2.

## Unknown nodes and properties

Unknown nodes and unknown properties are errors.

This is intentional. Configuration files are declarative and persistent. Accepting unknown fields would make typos difficult to detect.

For example, this is invalid:

```kdl
kind "fixed" tilte="Fixed"
```

The intended property is probably `title`, but rellog must reject this configuration instead of silently ignoring it.

## Property

### `rellog.config-version` (required)

The `rellog` node must have a `config-version` property.

Only `config-version=1` is supported.

### `rellog.github-url` (required)

The `rellog` node must have exactly one `github-url` child node.

```kdl
github-url "https://github.com/tooppoo/rellog"
```

The value must be the canonical HTTPS repository URL:

```text
https://github.com/<owner>/<repo>
```

This URL is used by `rellog add` to normalize numeric issue and pull request
references into canonical GitHub URLs. URL references passed to `rellog add`
must match this repository URL and the expected `/issues/<number>` or
`/pull/<number>` path shape.

`rellog` does not contact GitHub and does not verify whether an issue or pull
request number exists.

Invalid forms include `.git` suffixes, SSH URLs, `http`, trailing slashes,
query strings, fragments, and non-GitHub hosts.

### `entries.target-policy` (optional default = "deny-unknown")

`target-policy` controls how rellog handles entry targets that are not listed in `entries.targets`.

```kdl
target-policy "deny-unknown"
```

Allowed values are:

| Value           | Meaning                                                      |
| --------------- | ------------------------------------------------------------ |
| `deny-unknown`  | Unknown targets are errors.                                  |
| `warn-unknown`  | Unknown targets produce warnings, but commands may continue. |
| `allow-unknown` | Unknown targets are accepted without diagnostics.            |

If `target-policy` is `deny-unknown` or `warn-unknown`, `entries.targets` is required and must contain at least one `target`.

If `target-policy` is `allow-unknown`, `entries.targets` is optional.

### `kind.title` (optional default = "<kind id>")

If `title` is omitted, the kind id is used as the title without case conversion.

For example:

```kdl
kind "fixed"
```

renders as:

```markdown
### fixed
```

To render a different heading, specify `title`.

```kdl
kind "fixed" title="バグ修正"
```

`title`, when present, must be a non-empty string.

Invalid:

```kdl
kind "fixed" title=""
kind "fixed" title=" "
```

Effective kind titles must be unique.

The effective title is:

* `title`, if present
* otherwise, the kind id

For example, this is invalid:

```kdl
kinds {
  kind "fixed"
  kind "bugfix" title="fixed"
}
```

Both kinds would have the effective title `fixed`.

### `kind.description` (optional default = "")

```kdl
kind "fixed" description="Bug fixes."
```

`description` is used to describe what the kind means. It is metadata for humans and tools. It is not necessarily rendered into every generated changelog.

`description` must be a string when present.

An empty description is allowed:

```kdl
kind "fixed" description=""
```

### `target.title` (optional default = "<target id>")

If `title` is omitted, the target id is used as the title without case conversion.

```kdl
target "cli"
target "cli" title="CLI"
```

`title`, when present, must be a non-empty string.

Effective target titles must be unique.

The effective title is:

* `title`, if present
* otherwise, the target id

### `target.description` (optional default = "")

```kdl
target "config" description="Configuration file schema and validation."
```

`description` must be a string when present.

An empty description is allowed.

### `rendering.release-heading-level` (optional default = "2")

The value must be an integer from 1 to 6.

It controls the Markdown heading level used for each release.

For example, level 2 renders as:

```markdown
## 0.1.0
```

### `rendering.section-heading-level` (optional default = "3")

The value must be an integer from 1 to 6.

It controls the Markdown heading level used for kind sections.

For example, level 3 renders as:

```markdown
### fixed
```

`section-heading-level` must be greater than `release-heading-level`.

### `rendering.empty-message` (optional default = "No changelog-worthy changes.")

The value must be a string.

It is used when rendering an empty release.

### `paths` (required)

The `paths` section is required.

```kdl
paths {
  changelog "CHANGELOG.md"
  entries ".rellog/entries"
  release-notes ".rellog/release-notes"
}
```

#### Required paths

The following path nodes are required:

| Node            | Meaning                              |
| --------------- | ------------------------------------ |
| `changelog`     | Path to the changelog file.          |
| `entries`       | Path to the pending entry directory. |
| `release-notes` | Path to the release note directory.  |

Each path node must have exactly one string argument.

#### Path rules

Configuration paths are repository-root-relative logical paths.

They must be written in canonical form.

A configuration path must:

* be non-empty
* be relative to the repository root
* use `/` as the path separator
* not be an absolute path
* not contain `\`
* not contain empty path segments
* not contain `.` segments
* not contain `..` segments
* not end with `/`

Valid examples:

```kdl
changelog "CHANGELOG.md"
entries ".rellog/entries"
release-notes ".rellog/release-notes"
```

Invalid examples:

```kdl
changelog ""
changelog "./CHANGELOG.md"
changelog "docs/./CHANGELOG.md"
changelog "../CHANGELOG.md"
changelog "docs/../CHANGELOG.md"
changelog "docs//CHANGELOG.md"
changelog "docs/"
changelog "/tmp/CHANGELOG.md"
changelog "C:\\repo\\CHANGELOG.md"
```

`./` is rejected even though it is not dangerous by itself. rellog configuration paths are logical paths, not shell-style input paths. The same path should have only one valid representation.

#### Path conflicts

The configured paths must not conflict with each other.

At minimum, rellog should reject configurations where:

* two configured paths are identical
* `changelog` is inside `entries`
* `changelog` is inside `release-notes`
* `entries` is inside `release-notes`
* `release-notes` is inside `entries`

### `entries` (required)

The `entries` section is required.

```kdl
entries {
  target-policy "deny-unknown"

  kinds {
    kind "added"
    kind "changed"
    kind "fixed"
  }

  targets {
    target "rellog"
  }
}
```

### `entries.kinds` (required)

The `kinds` section is required.

It defines the set of normal entry kinds allowed in entry files.

```kdl
kinds {
  kind "added"
  kind "changed"
  kind "fixed"
}
```

`kinds` must contain at least one `kind` node.

The order of `kind` nodes defines the rendering order of changelog sections.

#### `kind` (required)

A `kind` node defines one normal changelog entry kind.

```kdl
kind "fixed" title="Fixed" description="Bug fixes."
```

Shape:

```kdl
kind "<id>" [title="<title>"] [description="<description>"]
```

A `kind` node must:

* have exactly one argument
* use a string argument as its id
* not have children
* not contain unknown properties

#### Kind id

A kind id must match:

```text
[A-Za-z][a-z0-9]*(?:-[a-z0-9]+)*
```

Valid examples:

```kdl
kind "added"
kind "Added"
kind "fixed"
kind "security"
kind "breaking-change"
```

Invalid examples:

```kdl
kind ""
kind " "
kind "fix_bug"
kind "fixBug"
kind "fix bug"
kind "-fixed"
kind "fixed-"
kind "修正"
```

Kind ids must be unique.

#### Reserved kind ids

`empty` is reserved.

It must not be defined in `entries.kinds`.

Invalid:

```kdl
kinds {
  kind "empty"
}
```

`empty` is used by rellog for empty release entries. It is not a normal changelog section kind.

Entry validation must treat `empty` as a reserved kind before checking configured kinds.

Conceptually:

```text
if kind == "empty":
  validate as an empty entry
else:
  require kind to be listed in entries.kinds
```

### `entries.targets` (required when `target-policy` is "deny-unknown" or "warn-unknown")

The `targets` section defines known entry targets.

```kdl
targets {
  target "cli"
  target "config"
}
```

Whether `targets` is required depends on `target-policy`.

| `target-policy` | `targets` |
| --------------- | --------- |
| `deny-unknown`  | Required  |
| `warn-unknown`  | Required  |
| `allow-unknown` | Optional  |

If `targets` is present, it must contain at least one `target` node.

#### `target` (required)

A `target` node defines one known target.

```kdl
target "config" title="Configuration" description="Configuration file schema and validation."
```

Shape:

```kdl
target "<id>" [title="<title>"] [description="<description>"]
```

A `target` node must:

* have exactly one argument
* use a string argument as its id
* not have children
* not contain unknown properties

#### Target id

A target id must match:

```text
[a-z][a-z0-9-]*
```

Valid examples:

```kdl
target "cli"
target "config"
target "release-notes"
```

Invalid examples:

```kdl
target ""
target " "
target "CLI"
target "release_notes"
target "release notes"
target "-config"
target "config-"
target "設定"
```

Target ids must be unique.

### `rendering` (optional default = "{}")

The `rendering` section is optional.

It configures how release notes and changelog sections are rendered.

```kdl
rendering {
  release-heading-level 2
  section-heading-level 3
  empty-message "No changelog-worthy changes."
}
```

If `rendering` is omitted, all rendering defaults are used.

An empty `rendering` node is allowed.

```kdl
rendering {
}
```

## Minimal valid configuration

A minimal configuration using strict target validation:

```kdl
/- kdl-version 2

rellog config-version=1 {
  github-url "https://github.com/tooppoo/rellog"
  paths {
    changelog "CHANGELOG.md"
    entries ".rellog/entries"
    release-notes ".rellog/release-notes"
  }

  entries {
    kinds {
      kind "added"
      kind "changed"
      kind "fixed"
    }

    targets {
      target "rellog"
    }
  }
}
```

`target-policy` is omitted here, so it defaults to `deny-unknown`.

A minimal configuration allowing arbitrary targets:

```kdl
/- kdl-version 2

rellog config-version=1 {
  github-url "https://github.com/tooppoo/rellog"
  paths {
    changelog "CHANGELOG.md"
    entries ".rellog/entries"
    release-notes ".rellog/release-notes"
  }

  entries {
    target-policy "allow-unknown"

    kinds {
      kind "added"
      kind "changed"
      kind "fixed"
    }
  }
}
```

## Validation summary

### Root

Error codes use the dotted path to the configuration node or property where the problem occurred, followed by an error id.

| Code                                | Condition                                               |
| ----------------------------------- | ------------------------------------------------------- |
| `config.parse_error`                | The file cannot be parsed as KDL v2.                    |
| `kdl-version.unsupported`           | The KDL version marker is present but is not version 2. |
| `rellog.missing`                    | The `rellog` root node is missing.                      |
| `rellog.duplicate`                  | More than one `rellog` root node exists.                |
| `rellog.config-version.missing`     | `config-version` is missing.                            |
| `rellog.config-version.unsupported` | `config-version` is not supported.                      |
| `rellog.github-url.missing`         | `github-url` is missing.                                |
| `rellog.github-url.argument_count`  | `github-url` does not have exactly one argument.        |
| `rellog.github-url.type`            | `github-url` is not a string.                           |
| `rellog.github-url.invalid`         | `github-url` is not a canonical GitHub repository URL.  |
| `<path>.unknown_node`               | An unknown node is present.                             |
| `<path>.unknown_property`           | An unknown property is present.                         |
| `<path>.<property>.duplicate`       | A property appears more than once in the same node.     |

### Paths

| Code                                      | Condition                                       |
| ----------------------------------------- | ----------------------------------------------- |
| `rellog.paths.missing`                    | `paths` is missing.                             |
| `rellog.paths.<path>.missing`             | A required path node is missing.                |
| `rellog.paths.<path>.argument_count`      | A path node does not have exactly one argument. |
| `rellog.paths.<path>.type`                | A path value is not a string.                   |
| `rellog.paths.<path>.empty`               | A path value is empty.                          |
| `rellog.paths.<path>.absolute`            | A path is absolute.                             |
| `rellog.paths.<path>.backslash`           | A path contains `\`.                            |
| `rellog.paths.<path>.empty_segment`       | A path contains an empty segment.               |
| `rellog.paths.<path>.dot_segment`         | A path contains a `.` segment.                  |
| `rellog.paths.<path>.traversal`           | A path contains a `..` segment.                 |
| `rellog.paths.<path>.trailing_slash`      | A path ends with `/`.                           |
| `rellog.paths.<path>.conflict`            | A configured path conflicts with another path.  |
| `rellog.paths.<path>.unexpected_children` | A path node has children.                       |

### Entries

| Code                                            | Condition                                             |
| ----------------------------------------------- | ----------------------------------------------------- |
| `rellog.entries.missing`                        | `entries` is missing.                                 |
| `rellog.entries.target-policy.invalid`          | `target-policy` has an unsupported value.             |
| `rellog.entries.target-policy.type`             | `target-policy` is not a string.                      |
| `rellog.entries.target-policy.duplicate`        | `target-policy` appears more than once.               |
| `rellog.entries.targets.required`               | `targets` is required by `target-policy` but missing. |
| `rellog.entries.unexpected_children`            | `entries` has unexpected children.                    |

### Kinds

| Code                                             | Condition                                         |
| ------------------------------------------------ | ------------------------------------------------- |
| `rellog.entries.kinds.missing`                   | `kinds` is missing.                               |
| `rellog.entries.kinds.empty`                     | `kinds` contains no `kind` nodes.                 |
| `rellog.entries.kinds.kind.argument_count`       | A `kind` node does not have exactly one argument. |
| `rellog.entries.kinds.kind.id.type`              | A kind id is not a string.                        |
| `rellog.entries.kinds.kind.id.empty`             | A kind id is empty or whitespace-only.            |
| `rellog.entries.kinds.kind.id.invalid`           | A kind id does not match the required format.     |
| `rellog.entries.kinds.kind.id.reserved`          | A kind id is reserved.                            |
| `rellog.entries.kinds.kind.id.duplicate`         | A kind id is duplicated.                          |
| `rellog.entries.kinds.kind.title.type`           | `title` is not a string.                          |
| `rellog.entries.kinds.kind.title.empty`          | `title` is empty or whitespace-only.              |
| `rellog.entries.kinds.kind.title.duplicate`      | An effective title is duplicated.                 |
| `rellog.entries.kinds.kind.description.type`     | `description` is not a string.                    |
| `rellog.entries.kinds.kind.unknown_property`     | A `kind` node has an unknown property.            |
| `rellog.entries.kinds.kind.unexpected_children`  | A `kind` node has children.                       |

### Targets

| Code                                                 | Condition                                            |
| ---------------------------------------------------- | ---------------------------------------------------- |
| `rellog.entries.targets.empty`                       | `targets` is present but contains no `target` nodes. |
| `rellog.entries.targets.target.argument_count`       | A `target` node does not have exactly one argument.  |
| `rellog.entries.targets.target.id.type`              | A target id is not a string.                         |
| `rellog.entries.targets.target.id.empty`             | A target id is empty or whitespace-only.             |
| `rellog.entries.targets.target.id.invalid`           | A target id does not match the required format.      |
| `rellog.entries.targets.target.id.duplicate`         | A target id is duplicated.                           |
| `rellog.entries.targets.target.title.type`           | `title` is not a string.                             |
| `rellog.entries.targets.target.title.empty`          | `title` is empty or whitespace-only.                 |
| `rellog.entries.targets.target.title.duplicate`      | An effective target title is duplicated.             |
| `rellog.entries.targets.target.description.type`     | `description` is not a string.                       |
| `rellog.entries.targets.target.unknown_property`     | A `target` node has an unknown property.             |
| `rellog.entries.targets.target.unexpected_children`  | A `target` node has children.                        |

### Rendering

| Code                                            | Condition                                                            |
| ----------------------------------------------- | -------------------------------------------------------------------- |
| `rellog.rendering.release-heading-level.type`   | `release-heading-level` is not an integer.                           |
| `rellog.rendering.release-heading-level.range`  | `release-heading-level` is outside 1 to 6.                           |
| `rellog.rendering.section-heading-level.type`   | `section-heading-level` is not an integer.                           |
| `rellog.rendering.section-heading-level.range`  | `section-heading-level` is outside 1 to 6.                           |
| `rellog.rendering.section-heading-level.order`  | `section-heading-level` is not greater than `release-heading-level`. |
| `rellog.rendering.empty-message.type`           | `empty-message` is not a string.                                     |
| `rellog.rendering.unknown_node`                 | An unknown rendering node is present.                                |
| `rellog.rendering.<property>.unknown_property`  | An unknown rendering property is present.                            |

## References

- [KDL v2](https://github.com/kdl-org/kdl/blob/main/draft-marchan-kdl2.md)
- [e2e tests](../e2e/)
