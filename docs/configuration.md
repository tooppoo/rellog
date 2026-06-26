# Configuration

rellog reads its repository configuration from `.rellog/config.kdl`.

The configuration file defines repository-level policies used by rellog commands, such as:

* where changelog-related files are stored
* which entry kinds are allowed
* which entry targets are known
* how release notes and changelog sections are rendered

The configuration file is written in [KDL v2](https://github.com/kdl-org/kdl/blob/main/draft-marchan-kdl2.md).

## Example

```kdl
/- kdl-version 2

rellog config-version=1 {
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
[a-z][a-z0-9-]*
```

Valid examples:

```kdl
kind "added"
kind "fixed"
kind "security"
kind "breaking-change"
```

Invalid examples:

```kdl
kind ""
kind " "
kind "Added"
kind "fix_bug"
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

| Code                             | Condition                                               |
| -------------------------------- | ------------------------------------------------------- |
| `config.parse_error`             | The file cannot be parsed as KDL v2.                    |
| `config.unsupported_kdl_version` | The KDL version marker is present but is not version 2. |
| `config.root_missing`            | The `rellog` root node is missing.                      |
| `config.root_duplicate`          | More than one `rellog` root node exists.                |
| `config.version_missing`         | `config-version` is missing.                            |
| `config.version_unsupported`     | `config-version` is not supported.                      |
| `config.unknown_node`            | An unknown node is present.                             |
| `config.unknown_property`        | An unknown property is present.                         |
| `config.duplicate_property`      | A property appears more than once in the same node.     |

### Paths

| Code                         | Condition                                       |
| ---------------------------- | ----------------------------------------------- |
| `config.paths.missing`       | `paths` is missing.                             |
| `config.path.missing`        | A required path node is missing.                |
| `config.path.argument_count` | A path node does not have exactly one argument. |
| `config.path.type`           | A path value is not a string.                   |
| `config.path.empty`          | A path value is empty.                          |
| `config.path.absolute`       | A path is absolute.                             |
| `config.path.backslash`      | A path contains `\`.                            |
| `config.path.empty_segment`  | A path contains an empty segment.               |
| `config.path.dot_segment`    | A path contains a `.` segment.                  |
| `config.path.traversal`      | A path contains a `..` segment.                 |
| `config.path.trailing_slash` | A path ends with `/`.                           |
| `config.path.conflict`       | Configured paths conflict with each other.      |

### Entries

| Code                                   | Condition                                             |
| -------------------------------------- | ----------------------------------------------------- |
| `config.entries.missing`               | `entries` is missing.                                 |
| `config.entries.target_policy.invalid` | `target-policy` has an unsupported value.             |
| `config.entries.targets.required`      | `targets` is required by `target-policy` but missing. |

### Kinds

| Code                                            | Condition                                         |
| ----------------------------------------------- | ------------------------------------------------- |
| `config.entries.kinds.missing`                  | `kinds` is missing.                               |
| `config.entries.kinds.empty`                    | `kinds` contains no `kind` nodes.                 |
| `config.entries.kinds.kind.argument_count`      | A `kind` node does not have exactly one argument. |
| `config.entries.kinds.kind.id_type`             | A kind id is not a string.                        |
| `config.entries.kinds.kind.id_empty`            | A kind id is empty or whitespace-only.            |
| `config.entries.kinds.kind.id_invalid`          | A kind id does not match the required format.     |
| `config.entries.kinds.kind.id_reserved`         | A kind id is reserved.                            |
| `config.entries.kinds.kind.id_duplicate`        | A kind id is duplicated.                          |
| `config.entries.kinds.kind.title_type`          | `title` is not a string.                          |
| `config.entries.kinds.kind.title_empty`         | `title` is empty or whitespace-only.              |
| `config.entries.kinds.kind.title_duplicate`     | An effective title is duplicated.                 |
| `config.entries.kinds.kind.description_type`    | `description` is not a string.                    |
| `config.entries.kinds.kind.unknown_property`    | A `kind` node has an unknown property.            |
| `config.entries.kinds.kind.unexpected_children` | A `kind` node has children.                       |

### Targets

| Code                                                | Condition                                            |
| --------------------------------------------------- | ---------------------------------------------------- |
| `config.entries.targets.empty`                      | `targets` is present but contains no `target` nodes. |
| `config.entries.targets.target.argument_count`      | A `target` node does not have exactly one argument.  |
| `config.entries.targets.target.id_type`             | A target id is not a string.                         |
| `config.entries.targets.target.id_empty`            | A target id is empty or whitespace-only.             |
| `config.entries.targets.target.id_invalid`          | A target id does not match the required format.      |
| `config.entries.targets.target.id_duplicate`        | A target id is duplicated.                           |
| `config.entries.targets.target.title_type`          | `title` is not a string.                             |
| `config.entries.targets.target.title_empty`         | `title` is empty or whitespace-only.                 |
| `config.entries.targets.target.title_duplicate`     | An effective title is duplicated.                    |
| `config.entries.targets.target.description_type`    | `description` is not a string.                       |
| `config.entries.targets.target.unknown_property`    | A `target` node has an unknown property.             |
| `config.entries.targets.target.unexpected_children` | A `target` node has children.                        |

### Rendering

| Code                                           | Condition                                                            |
| ---------------------------------------------- | -------------------------------------------------------------------- |
| `config.rendering.release_heading_level.type`  | `release-heading-level` is not an integer.                           |
| `config.rendering.release_heading_level.range` | `release-heading-level` is outside 1 to 6.                           |
| `config.rendering.section_heading_level.type`  | `section-heading-level` is not an integer.                           |
| `config.rendering.section_heading_level.range` | `section-heading-level` is outside 1 to 6.                           |
| `config.rendering.section_heading_level.order` | `section-heading-level` is not greater than `release-heading-level`. |
| `config.rendering.empty_message.type`          | `empty-message` is not a string.                                     |

## References

- [KDL v2](https://github.com/kdl-org/kdl/blob/main/draft-marchan-kdl2.md)
- [e2e tests](../e2e/)
