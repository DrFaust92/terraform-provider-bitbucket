---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_project"
sidebar_current: "docs-bitbucket-resource-project"
description: |-
  Create and manage a Bitbucket project
---


# bitbucket\_project

This resource allows you to manage your projects in your bitbucket team.

* OAuth2 Scopes: `project` and `project:admin`
* API token permissions: `read:project:bitbucket` and `admin:project:bitbucket`

## Example Usage

```hcl
resource "bitbucket_project" "devops" {
  owner = "my-team"
  name  = "devops"
  key   = "DEVOPS"
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) The owner of this project. Can be you or any team you have write access to.
* `name` - (Required) The name of the project
* `key` - (Required) The key used for this project
* `description` - (Optional) The description of the project
* `is_private` - (Optional) If you want to keep the project private - defaults to `true`
* `link` - (Optional) A set of links to a resource related to this object. See [Link](#link) Below.

### Link

* `avatar` - (Optional) An avatar link to a resource related to this object. See [Avatar](#avatar) Below.

#### Avatar

* `href` - (Optional) href of the avatar.

## Attributes Reference

* `uuid` - The project's immutable id.
* `has_publicly_visible_repos` - Indicates whether the project contains publicly visible repositories. Note that private projects cannot contain public repositories.

## Import

Repositories can be imported using their `owner/key` ID, e.g.

```sh
terraform import bitbucket_project.my_project my-account/project_key
```
