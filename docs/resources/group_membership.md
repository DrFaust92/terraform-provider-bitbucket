---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_group_membership"
sidebar_current: "docs-bitbucket-resource-group-membership"
description: |-
  Provides support for setting Bitbucket Group Membership
---

# bitbucket\_group\_membership

Provides a Bitbucket group membership resource.

This allows you to manage your group membership.

* OAuth2 Scopes: `account:write`
* API token permissions: `read:permission:bitbucket` and `write:permission:bitbucket`

## Example Usage

```hcl
data "bitbucket_workspace" "test" {
  workspace = "example"
}

resource "bitbucket_group" "test" {
  workspace  = data.bitbucket_workspace.test.id
  name       = "example"
}

data "bitbucket_current_user" "test" {}

resource "bitbucket_group_membership" "test" {
  workspace  = bitbucket_group.test.workspace
  group_slug = bitbucket_group.test.slug
  uuid       = data.bitbucket_current_user.test.id
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace of this repository.
* `group_slug` - (Required) The slug of the group.
* `uuid` - (Required) The member UUID to add to the group.

## Import

Group Members can be imported using their `workspace/group-slug/member-uuid` ID, e.g.

```sh
terraform import bitbucket_group_membership.group my-workspace/group-slug/member-uuid
```
