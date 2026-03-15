---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_workspace"
sidebar_current: "docs-bitbucket-data-workspace"
description: |-
  Provides a data for a Bitbucket workspace
---

# bitbucket\_workspace

Provides a way to fetch data on a workspace.

* OAuth2 Scopes: `none`
* API token permissions: `read:workspace:bitbucket`

## Example Usage

```hcl
data "bitbucket_workspace" "example" {
  workspace = "gob"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) This can either be the workspace ID (slug) or the workspace UUID surrounded by curly-braces

## Attributes Reference

* `name` - The name of the workspace.
* `slug` - The short label that identifies this workspace.
* `is_private` - Indicates whether the workspace is publicly accessible, or whether it is private to the members and consequently only visible to members.
* `id` - The workspace's immutable id.
