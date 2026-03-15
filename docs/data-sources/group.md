---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_group"
sidebar_current: "docs-bitbucket-data-group"
description: |-
  Provides a data for a Bitbucket group
---

# bitbucket\_group

Provides a way to fetch data of a group.

* OAuth2 Scopes: `account`
* API token permissions: `read:permission:bitbucket`

## Example Usage

```hcl
data "bitbucket_group" "example" {
  workspace = "example"
  slug      = "example"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The UUID that bitbucket groups to connect a group to various objects
* `slug` - (Required) The group's slug.

## Attributes Reference

* `name` - The name of the group.
* `auto_add` - Whether to automatically add users the group
* `permission` - One of `read`, `write`, and `admin`.
* `email_forwarding_disabled` - Whether to disable email forwarding for group.
