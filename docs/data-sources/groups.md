---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_groups"
sidebar_current: "docs-bitbucket-data-groups"
description: |-
  Provides a data for Bitbucket groups
---

# bitbucket\_groups

Provides a way to fetch data of groups in a workspace.

* OAuth2 Scopes: `account`
* API token permissions: `read:permission:bitbucket`

## Example Usage

```hcl
data "bitbucket_groups" "example" {
  workspace = "example"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The UUID that bitbucket groupss to connect a groups to various objects

## Attributes Reference

* `groups` - The list of groups in the workspace. See Group below for structure of each element

### Group

* `name` - The name of the groups.
* `auto_add` - Whether to automatically add users the groups
* `permission` - One of `read`, `write`, and `admin`.
* `slug` - The groups's slug.
* `email_forwarding_disabled` - Whether to disable email forwarding for group.
