---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_forked_repository"
sidebar_current: "docs-bitbucket-resource-forked-repository"
description: |-
  Provides a Bitbucket Repository
---

# bitbucket\_forked\_repository

Provides a Bitbucket repository resource that is forked from a parent repo.

This resource allows you manage properties of the fork, if it is
private, how to fork the repository and other options. SCM cannot be overridden,
as it is inherited from the parent repository. Creation will fail if the parent
repo has `no_forks` as its fork policy.

* OAuth2 Scopes: `repository`, `repository:admin`, and `repository:delete`
* API token permissions: `read:repository:bitbucket`, `write:repository:bitbucket`, and `admin:repository:bitbucket`

## Example Usage

```hcl
resource "bitbucket_forked_repository" "infrastructure" {
  owner = "myteam"
  name  = "terraform-code"
}
```

If you want to create a repository with a CamelCase name, you should provide
a separate slug

```hcl
resource "bitbucket_forked_repository" "infrastructure" {
  owner = "myteam"
  name  = "TerraformCode"
  slug  = "terraform-code"

  parent = {
    owner = bitbucket_repository.test.owner
    slug  = bitbucket_repository.test.slug
  }
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) The owner of this repository. Can be you or any team you
  have write access to.
* `name` - (Required) The name of the repository.
* `slug` - (Optional) The slug of the repository.
* `is_private` - (Optional) If this should be private or not. Defaults to `true`. Note that if
  the parent repo has `no_public_forks` as its fork policy, the resource may
  fail to be created.
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
* `parent` - The repository to fork from. See [Parent](#parent) below.

### Link

* `avatar` - (Optional) An avatar link to a resource related to this object. See [Avatar](#avatar) Below.

#### Avatar

* `href` - (Optional) href of the avatar.

### Parent

* `owner` - The owner of the repository we are forking from. Can be you or any other team you
  have write access to.
* `slug` - The slug of the parent repository. Found in the URL, typically this is the repository
  name, all lowercase.

## Attributes Reference

* `clone_ssh` - The SSH clone URL.
* `clone_https` - The HTTPS clone URL.
* `uuid` - The uuid of the repository resource.
* `scm` - The SCM of the resource. Either `hg` or `git`.

## Import

Repositories can be imported using their `owner/name` ID, e.g.

```sh
terraform import bitbucket_forked_repository.my-repo my-account/my-repo
```
