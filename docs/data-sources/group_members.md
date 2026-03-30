---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_group_members"
sidebar_current: "docs-bitbucket-data-group-members"
description: |-
  Provides a data for Bitbucket group members
---

# bitbucket\_group\_members

Provides a way to fetch data of group members.

* OAuth2 Scopes: `account`
* API token permissions: `read:permission:bitbucket`

## Example Usage

```hcl
data "bitbucket_group_members" "example" {
  workspace = "example"
  slug      = "example"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The UUID that bitbucket groups to connect a group to various objects
* `slug` - (Required) The group's slug.

## Attributes Reference

* `members` - A list of group member uuid.
* `group_members` - A set of group member objects. See [Group Member](#group-member) below.

### Group Member

* `uuid` - User UUID.
* `username` - The Username.
* `display_name` - The User display name.
