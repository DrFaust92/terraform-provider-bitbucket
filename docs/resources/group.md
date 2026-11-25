---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_group"
sidebar_current: "docs-bitbucket-resource-group"
description: |-
  Provides a Bitbucket Group
---

# bitbucket\_group

Provides a Bitbucket group resource.

This allows you to manage your groups.

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
  auto_add   = true
  permission = "read"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace slug or UUID. A UUID, such as the `id` of the `bitbucket_workspace` data source, is resolved to the slug, which the groups API requires.
* `name` - (Required) The name of the group.
* `auto_add` - (Optional) Whether to automatically add users the group
* `permission` - (Optional) One of `read`, `write`, and `admin`.
* `email_forwarding_disabled` - Whether to disable email forwarding for group.

## Attributes Reference

* `slug` - The groups slug.

## Import

Groups can be imported using their `workspace/group-slug` ID, e.g.

```sh
terraform import bitbucket_group.group my-workspace/group-slug
```
