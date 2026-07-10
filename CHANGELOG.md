# CHANGELOG

## v0.1.4

### changed

#### actions

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
Manage GitHub actions on another repository.
this change is accpeted to separate rellog's version and actions version.
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

## v0.1.3

### changed

#### cli / core

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
create `.rellog/.gitignore` and ignore `.rellog/consumed` as defautl
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

## v0.1.2

### fixed

#### cli / docs

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
update installer version and add installer documents.
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

## v0.1.1

### fixed

#### cli

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
ensure .rellog/entries before add entry
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

### added

#### actions

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
add `tooppoo/rellog/actions/create-release-note`
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

## v0.1.0

### changed

#### actions

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
exclude version header for Github release, because it is alreay contained as release title
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

#### cli

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
implements TUI to add new rellog entry.
it is enabled to use rich interface.
<!-- rellog:body:end -->

Refs:
- https://github.com/tooppoo/rellog/issues/11
<!-- rellog:entry:end -->

### fixed

#### core

<!-- rellog:entry:start -->
<!-- rellog:body:start -->
use target as section and discarded target-policy config
<!-- rellog:body:end -->
<!-- rellog:entry:end -->

## v0.1.0

### changed

#### Details

<!-- rellog:body:start -->
implements TUI for interactive addition command `rellog add`
<!-- rellog:body:end -->

#### Targets

- cli

#### Links

- https://github.com/tooppoo/rellog/issues/11

## v0.0.4

### changed

#### Details

<!-- rellog:body:start -->
resolve version via ldflags
<!-- rellog:body:end -->

#### Targets

- cli

## v0.0.3

### changed

#### Details

<!-- rellog:body:start -->
update release workflow
<!-- rellog:body:end -->

#### Targets

- cli

## v0.0.1

### added

#### Details

<!-- rellog:body:start -->
First Release
<!-- rellog:body:end -->

#### Targets

- cli
