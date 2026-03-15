---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_project_group_permission"
sidebar_current: "docs-bitbucket-resource-project-group-permission"
description: |-
  Provides a Bitbucket Repository Group Permission Resource
---

# bitbucket\_project\_group\_permission

Provides a Bitbucket Repository Group Permission Resource.

This allows you set explicit group permission for a project.

* OAuth2 Scopes: `project:admin`
* API token permissions: `read:project:bitbucket`, `write:permission:bitbucket`, `admin:project:bitbucket`, and `delete:permission:bitbucket`

Note: can only be used when authenticating with Bitbucket Cloud using an _API token_. Authenticating via an OAuth flow gives a 403 error due to a [restriction in the Bitbucket Cloud API](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-project-key-permissions-config-groups-group-slug-put).

## Example Usage

```hcl
resource "bitbucket_project_group_permission" "example" {
  workspace   = "example"
  project_key = bitbucket_project.example.key
  group_slug  = bitbucket_group.example.slug
  permission  = "read"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace id.
* `project_key` - (Required) The project key.
* `group_slug` - (Required) Slug of the requested group.
* `permission` - (Required) Permissions can be one of `read`, `write`, `create-repo`, and `admin`.

## Import

Repository Group Permissions can be imported using their `workspace:project-key:group-slug` ID, e.g.

```sh
terraform import bitbucket_project_group_permission.example workspace:project-key:group-slug
```
