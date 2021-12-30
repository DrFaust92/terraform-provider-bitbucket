---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_branching_model"
sidebar_current: "docs-bitbucket-resource-branching-model"
description: |-
  Provides a Bitbucket Branching Model
---

# bitbucket\_branching\_model

Provides a Bitbucket branching model resource.

This allows you for setting up branching models for your repository.

## Example Usage

```hcl
# Manage your repositories branching models
resource "bitbucket_repository" "test" {
  owner = "example"
  name  = "example"
}
resource "bitbucket_branching_model" "test" {
  owner      = "example"
  repository = bitbucket_repository.test.name

  development {
    use_mainbranch = true
  }

  branch_type {
    enabled = true
    kind    = "feature"
    prefix  = "test/"
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

* `owner` - (Required) The owner of this repository. Can be you or any team you
  have write access to.
* `repository` - (Required) The name of the repository.
* `development` - (Optional) The development branch can be configured to a specific branch or to track the main branch. When set to a specific branch it must currently exist. Only the passed properties will be updated. The properties not passed will be left unchanged. A request without a development property will leave the development branch unchanged. See [Development](#development) below.
* `production` - (Optional) The production branch can be a specific branch, the main branch or disabled. When set to a specific branch it must currently exist. The enabled property can be used to enable (true) or disable (false) it. Only the passed properties will be updated. The properties not passed will be left unchanged. A request without a production property will leave the production branch unchanged. See [Production](#production) below.
* `branch_type` - (Required) A set of branch type to define `feature`, `bugfix`, `release`, `hotfix` prefixes. See [Branch Type](#branch-type) below.

### Development

* `name` - (Optional) The configured branch. It must be null when `use_mainbranch` is true. Otherwise it must be a non-empty value. It is possible for the configured branch to not exist (e.g. it was deleted after the settings are set).
* `use_mainbranch` - (Optional) Indicates if the setting points at an explicit branch (`false`) or tracks the main branch (`true`). When `true` the name must be null or not provided. When `false` the name must contain a non-empty branch name.
* `branch_does_not_exist` - (Optional) Optional and only returned for a repository's branching model. Indicates if the indicated branch exists on the repository (`false`) or not (`true`). This is useful for determining a fallback to the mainbranch when a repository is inheriting its project's branching model.

### Production

* `enabled` - (Optional) Indicates if branch is enabled or not.
* `name` - (Optional) The configured branch. It must be null when `use_mainbranch` is true. Otherwise it must be a non-empty value. It is possible for the configured branch to not exist (e.g. it was deleted after the settings are set).
* `use_mainbranch` - (Optional) Indicates if the setting points at an explicit branch (`false`) or tracks the main branch (`true`). When `true` the name must be null or not provided. When `false` the name must contain a non-empty branch name.
* `branch_does_not_exist` - (Optional) Optional and only returned for a repository's branching model. Indicates if the indicated branch exists on the repository (`false`) or not (`true`). This is useful for determining a fallback to the mainbranch when a repository is inheriting its project's branching model.

### Branch Type

* `enabled` - (Optional) Whether the branch type is enabled or not. A disabled branch type may contain an invalid `prefix`.
* `kind` - (Required) The kind of the branch type. Valid values are `feature`, `bugfix`, `release`, `hotfix`.
* `prefix` - (Optional) The prefix for this branch type. A branch with this prefix will be classified as per kind. The prefix of an enabled branch type must be a valid branch prefix. Additionally, it cannot be blank, empty or null. The prefix for a disabled branch type can be empty or invalid.
