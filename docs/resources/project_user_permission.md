---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_project_user_permission"
sidebar_current: "docs-bitbucket-resource-project-user-permission"
description: |-
  Provides a Bitbucket Repository User Permission Resource
---

# bitbucket\_project\_user\_permission

Provides a Bitbucket Repository User Permission Resource.

This allows you set explicit user permission for a project.

* OAuth2 Scopes: `project:admin`
* API token permissions: `read:project:bitbucket`, `admin:project:bitbucket`, `write:permission:bitbucket`, and `delete:permission:bitbucket`

## Example Usage

```hcl
resource "bitbucket_project_user_permission" "example" {
  workspace   = "example"
  project_key = bitbucket_project.example.key
  user_id     = "user-id"
  permission  = "read"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace id.
* `project_key` - (Required) The project key.
* `user_id` - (Required) The UUID of the user.
* `permission` - (Required) Permissions can be one of `read`, `write`, `create-repo`, and `admin`.

## Import

Repository User Permissions can be imported using their `workspace:project-key:user-id` ID, e.g.

```sh
terraform import bitbucket_project_user_permission.example workspace:project-key:user-id
```
