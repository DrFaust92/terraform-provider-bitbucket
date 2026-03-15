---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_repository_group_permission"
sidebar_current: "docs-bitbucket-resource-repository-group-permission"
description: |-
  Provides a Bitbucket Repository Group Permission Resource
---

# bitbucket\_repository\_group\_permission

Provides a Bitbucket Repository Group Permission Resource.

This allows you set explicit group permission for a repository.

* OAuth2 Scopes: `repository:admin`
* API token permissions: `read:project:bitbucket`, `admin:project:bitbucket`, `write:permission:bitbucket`, and `delete:permission:bitbucket`

Note: can only be used when authenticating with Bitbucket Cloud using an _API token_. Authenticating via an OAuth flow gives a 403 error due to a [restriction in the Bitbucket Cloud API](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-permissions-config-groups-group-slug-put).

## Example Usage

```hcl
resource "bitbucket_repository_group_permission" "example" {
  workspace  = "example"
  repo_slug  = bitbucket_repository.example.name
  group_slug = bitbucket_group.example.slug
  permission = "read"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace id.
* `repo_slug` - (Required) The repository slug.
* `group_slug` - (Required) Slug of the requested group.
* `permission` - (Required) Permissions can be one of `read`, `write`, and `admin`.

## Import

Repository Group Permissions can be imported using their `workspace:repo-slug:group-slug` ID, e.g.

```sh
terraform import bitbucket_repository_group_permission.example workspace:repo-slug:group-slug
```
