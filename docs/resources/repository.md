---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_repository"
sidebar_current: "docs-bitbucket-resource-repository"
description: |-
  Provides a Bitbucket Repository
---

# bitbucket\_repository

Provides a Bitbucket repository resource.

This resource allows you manage your repositories such as scm type, if it is
private, how to fork the repository and other options.

OAuth2 Scopes: `repository`, `repository:admin`, and `repository:delete`

## Example Usage

```hcl
resource "bitbucket_repository" "infrastructure" {
  owner = "myteam"
  name  = "terraform-code"
}
```

If you want to create a repository with a CamelCase name, you should provide
a separate slug

```hcl
resource "bitbucket_repository" "infrastructure" {
  owner = "myteam"
  name  = "TerraformCode"
  slug  = "terraform-code"
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) The owner of this repository. Can be you or any team you
  have write access to.
* `name` - (Required) The name of the repository.
* `slug` - (Optional) The slug of the repository.
* `scm` - (Optional) What SCM you want to use. Valid options are `hg` or `git`.
  Defaults to `git`.
* `is_private` - (Optional) If this should be private or not. Defaults to `true`.
* `website` - (Optional) URL of website associated with this repository.
* `language` - (Optional) What the language of this repository should be.
* `has_issues` - (Optional) If this should have issues turned on or not.
* `has_wiki` - (Optional) If this should have wiki turned on or not.
* `project_key` - (Optional) If you want to have this repo associated with a
  project.
* `fork_policy` - (Optional) What the fork policy should be. Defaults to
  `allow_forks`. Valid values are `allow_forks`, `no_public_forks`, `no_forks`.
* `description` - (Optional) What the description of the repo is.
* `pipelines_enabled` - (Optional) Turn on to enable pipelines support.
* `link` - (Optional) A set of links to a resource related to this object. See [Link](#link) Below.
* `inherit_default_merge_strategy` - (Optional) Whether to inherit default merge strategy from project.
* `inherit_branching_model` - (Optional) Whether to inherit branching model from project.
* `mainbranch` - (Optional) Name of main branch.

### Link

* `avatar` - (Optional) An avatar link to a resource related to this object. See [Avatar](#avatar) Below.

#### Avatar

* `href` - (Optional) href of the avatar.

## Attributes Reference

* `clone_ssh` - The SSH clone URL.
* `clone_https` - The HTTPS clone URL.
* `uuid` - the uuid of the repository resource.

## Import

Repositories can be imported using their `owner/name` ID, e.g.

```sh
terraform import bitbucket_repository.my-repo my-account/my-repo
```
