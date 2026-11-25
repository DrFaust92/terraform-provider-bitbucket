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

~> **Note:** groups are managed through Bitbucket's legacy 1.0 groups API, which Atlassian has deprecated in favour of [organization-level groups](https://support.atlassian.com/bitbucket-cloud/docs/groups-endpoint/). Creating or changing a group needs the `account:write` OAuth scope, and Atlassian API tokens can't be granted it ([#258](https://github.com/DrFaust92/terraform-provider-bitbucket/issues/258)): use an OAuth consumer with the **Account: Write** permission, or create the group in Atlassian Administration and read it with the `bitbucket_group` data source.

## Example Usage

```hcl
data "bitbucket_workspace" "test" {
  workspace = "example"
}

resource "bitbucket_group" "test" {
  workspace  = data.bitbucket_workspace.test.slug
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

* `workspace` - (Required) The workspace slug or UUID. A UUID, such as the `id` of the `bitbucket_workspace` data source, is resolved to the slug, which the groups API requires.
* `group_slug` - (Required) The slug of the group.
* `uuid` - (Required) The member UUID to add to the group.

## Import

Group Members can be imported using their `workspace/group-slug/member-uuid` ID, e.g.

```sh
terraform import bitbucket_group_membership.group my-workspace/group-slug/member-uuid
```
