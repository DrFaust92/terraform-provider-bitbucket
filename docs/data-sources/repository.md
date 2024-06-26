---
layout: bitbucket
page_title: "Bitbucket: bitbucket_repository"
sidebar_current: "docs-bitbucket-data-repository"
description: |-
  Provides a data source for a Bitbucket repository
---

# bitbucket\_repository

Provide a way to fetch data about a repository.

OAuth2 Scopes: `repository`

## Example Usage

```hcl
data "bitbucket_repository" "my_repo" {
  workspace = "myworspace"
  slug = "my-repo-slug"
}
```

## Argument Reference

* `workspace` - (Required) This can either be the workspace ID (slug) or the workspace UUID surrounded by curly-braces
* `slug`: - (Required) This can either be the repository slug or the UUID of the repository, surrounded by curly-braces

## Attribute Reference

* `owner` - The owner of this repository.
* `name` - The name of the repository.
* `slug` - The slug of the repository.
* `scm` - The SCM (`git` or `hg`) of the repository.
* `is_private` - If this repository is private or not.
* `website` - URL of website associated with this repository.
* `language` - The programming language of this repository.
* `has_issues` - If this repository has issues turned on.
* `has_wiki` - If this repository has wiki turned on.
* `project_key` - The key of a project the repository is linked to, if applicable.
* `fork_policy` - The repository fork policy. Valid values are
  `allow_forks`. Valid values are `allow_forks`, `no_public_forks`, `no_forks`.
* `description` - The description of the repo.
* `pipelines_enabled` - If this repository has pipelines turned on.
