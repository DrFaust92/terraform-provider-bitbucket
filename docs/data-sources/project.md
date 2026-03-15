---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_project"
sidebar_current: "docs-bitbucket-data-bitbucket-project"
description: |-
  Datasource to retrieve project information
---

# bitbucket\_project

Datasource to retrieve project information

* OAuth2 Scopes: `project`
* API token permissions: `read:project:bitbucket`

## Example Usage

```terraform
# Basic example
data "bitbucket_project" "test" {
	workspace = "myworkspace"
	key = "MYREPO"
}

# Basic example using UUID
data "bitbucket_project" "test" {
	workspace = "{f772f004-c268-4698-b49f-9d8415981464}"
	key = "MYREPO"
}
```

## Argument Reference

The following arguments are supported:

- `key` - (Required) Project key
- `workspace` - (Required) Project workspace slug or {UUID}
- `description` -  Project description
- `is_private` -  Project is private
- `link` - Project link information

## Attributes Reference

- `has_publicly_visible_repos` -  Repositories are publicly visible
- `id` -  The ID of this resource.
- `name` -  Project name
- `owner` - Project owner information (see [below for nested schema](#nestedblock--owner))
- `uuid` -  Project UUID

<a id="nestedblock--link"></a>
### Nested Schema for `link`

- `avatar` - Avatar link information (see [below for nested schema](#nestedblock--link--avatar))

<a id="nestedblock--link--avatar"></a>
### Nested Schema for `link.avatar`

- `href` - URL link

<a id="nestedblock--owner"></a>
### Nested Schema for `owner`

- `display_name` -  Owner display name
- `username` -  Owner username
- `uuid` -  Owner UUID
