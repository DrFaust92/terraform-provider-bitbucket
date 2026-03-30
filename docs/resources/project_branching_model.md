---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_project_branching_model"
sidebar_current: "docs-bitbucket-resource-project-branching-model"
description: |-
  Provides a Bitbucket Project Branching Model
---

# bitbucket\_project\_branching\_model

Provides a Bitbucket project branching model resource.

This allows you for setting up branching models for your project.

* OAuth2 Scopes: `project:admin`
* API token permissions: `read:repository:bitbucket`, `admin:repository:bitbucket`, and `admin:project:bitbucket`

## Example Usage

```hcl
resource "bitbucket_project" "example" {
  owner = "example"
  name  = "example"
  key   = "FFFFF"
}

resource "bitbucket_project_branching_model" "example" {
  workspace = "example"
  project   = bitbucket_project.example.key

  development {
    use_mainbranch = true
  }

  branch_type {
    enabled = true
    kind    = "feature"
    prefix  = "example/"
  }

  branch_type {
    enabled = true
    kind    = "hotfix"
    prefix  = "hotfix/"
  }

  branch_type {
    enabled = true
    kind    = "release"
    prefix  = "release/"
  }

  branch_type {
    enabled = true
    kind    = "bugfix"
    prefix  = "bugfix/"
  }
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace of this project. Can be you or any team you
  have write access to.
* `project` - (Required) The key of the project.
* `development` - (Optional) The development branch can be configured to a specific branch or to track the main branch. When set to a specific branch it must currently exist. Only the passed properties will be updated. The properties not passed will be left unchanged. A request without a development property will leave the development branch unchanged. See [Development](#development) below.
* `production` - (Optional) The production branch can be a specific branch, the main branch or disabled. When set to a specific branch it must currently exist. The enabled property can be used to enable (true) or disable (false) it. Only the passed properties will be updated. The properties not passed will be left unchanged. A request without a production property will leave the production branch unchanged. See [Production](#production) below.
* `branch_type` - (Required) A set of branch type to define `feature`, `bugfix`, `release`, `hotfix` prefixes. See [Branch Type](#branch-type) below.

### Development

* `name` - (Optional) The configured branch. It must be null when `use_mainbranch` is true. Otherwise it must be a non-empty value. It is possible for the configured branch to not exist (e.g. it was deleted after the settings are set).
* `use_mainbranch` - (Optional) Indicates if the setting points at an explicit branch (`false`) or tracks the main branch (`true`). When `true` the name must be null or not provided. When `false` the name must contain a non-empty branch name.
* `branch_does_not_exist` - (Optional) Optional and only returned for a project's branching model. Indicates if the indicated branch exists on the project (`false`) or not (`true`). This is useful for determining a fallback to the mainbranch when a project is inheriting its project's branching model.

### Production

* `enabled` - (Optional) Indicates if branch is enabled or not.
* `name` - (Optional) The configured branch. It must be null when `use_mainbranch` is true. Otherwise it must be a non-empty value. It is possible for the configured branch to not exist (e.g. it was deleted after the settings are set).
* `use_mainbranch` - (Optional) Indicates if the setting points at an explicit branch (`false`) or tracks the main branch (`true`). When `true` the name must be null or not provided. When `false` the name must contain a non-empty branch name.
* `branch_does_not_exist` - (Optional) Optional and only returned for a project's branching model. Indicates if the indicated branch exists on the project (`false`) or not (`true`). This is useful for determining a fallback to the mainbranch when a project is inheriting its project's branching model.

### Branch Type

* `enabled` - (Optional) Whether the branch type is enabled or not. A disabled branch type may contain an invalid `prefix`.
* `kind` - (Required) The kind of the branch type. Valid values are `feature`, `bugfix`, `release`, `hotfix`.
* `prefix` - (Optional) The prefix for this branch type. A branch with this prefix will be classified as per kind. The prefix of an enabled branch type must be a valid branch prefix. Additionally, it cannot be blank, empty or null. The prefix for a disabled branch type can be empty or invalid.

## Import

Branching Models can be imported using the workspace and project separated by a (`/`), e.g.,

```sh
terraform import bitbucket_project_branching_model.example workspace/project
```
