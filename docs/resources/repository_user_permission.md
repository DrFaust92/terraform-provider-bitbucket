---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_repository_user_permission"
sidebar_current: "docs-bitbucket-resource-repository-user-permission"
description: |-
  Provides a Bitbucket Repository User Permission Resource
---

# bitbucket\_repository\_user\_permission

Provides a Bitbucket Repository User Permission Resource.

This allows you set explicit user permission for a repository.

* OAuth2 Scopes: `repository:admin`
* API token permissions: `read:repository:bitbucket`, `admin:repository:bitbucket`, `write:permission:bitbucket`, and `delete:permission:bitbucket`

## Example Usage

```hcl
resource "bitbucket_repository_user_permission" "example" {
  workspace  = "example"
  repo_slug  = bitbucket_repository.example.name
  user_id    = "user-id"
  permission = "read"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace id.
* `repo_slug` - (Required) The repository slug.
* `user_id` - (Required) The UUID of the user.
* `permission` - (Required) Permissions can be one of `read`, `write`, `none`, and `admin`.

## Import

Repository User Permissions can be imported using their `workspace:repo-slug:user-id` ID, e.g.

```sh
terraform import bitbucket_repository_user_permission.example workspace:repo-slug:user-id
```
