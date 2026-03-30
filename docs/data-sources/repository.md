---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_repository"
sidebar_current: "docs-bitbucket-data-bitbucket-repository"
description: |-
  Datasource to retrieve repository information
---

# bitbucket\_repository

Datasource to retrieve repository information

* OAuth2 Scopes: `repository`
* API token permissions: `read:repository:bitbucket`

## Example Usage

```terraform
# Basic example
data "bitbucket_repository" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
}

# Basic example using UUIDs
data "bitbucket_repository" "test" {
	workspace = "{f772f004-c268-4698-b49f-9d8415981464}"
	repo_slug = "{2c3a2eee-fa63-4b60-afd4-97319245e79e}"
}
```

## Argument Reference

The following arguments are supported:

- `repo_slug` - (Required) Repository slug or UUID
- `workspace` - (Required) Workspace slug or UUID

## Attributes Reference

- `is_private` - If repository is private
- `description` - Repository description
- `fork_policy` - Repository fork policy
- `full_name` - Repository full name
- `has_issues` - If repository currently has JIRA issues assigned to it
- `has_wiki` - Repository has a Confluence page
- `id` - The ID of this resource.
- `language` - Repository language
- `link` Repository links (see [below for nested schema](#nestedblock--link))
- `main_branch` - Main branch name
- `name` - Repository name
- `owner` Repository owner information (see [below for nested schema](#nestedatt--owner))
- `project` Project information (see [below for nested schema](#nestedblock--project))
- `scm` - Repository SCM
- `uuid` - Repository UUID

<a id="nestedblock--link"></a>
### Nested Schema for `link`

- `avatar` - Repository avatar (see [below for nested schema](#nestedatt--link--avatar))

<a id="nestedatt--link--avatar"></a>
### Nested Schema for `link.avatar`

- `href` - URL link

<a id="nestedatt--owner"></a>
### Nested Schema for `owner`

- `display_name` -  Owner display name
- `username` -  Owner username
- `uuid` -  Owner UUID

<a id="nestedblock--project"></a>
### Nested Schema for `project`

Read-Only:

- `description` - Project description
- `is_private` - If project is private
- `key` - Project key
- `name` - Project name
