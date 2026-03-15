---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_project_default_reviewers"
sidebar_current: "docs-bitbucket-resource-project-default-reviewers"
description: |-
  Provides support for setting up project default reviews for bitbucket.
---

# bitbucket\_project\_default\_reviewers

Provides support for setting up default reviewers for your project. You must however have the UUID of the user available. Since Bitbucket has removed usernames from its APIs the best case is to use the UUID via the data provider.

* OAuth2 Scopes: `project:admin`
* API token permissions: `read:pullrequest:bitbucket` and `admin:project:bitbucket`

## Example Usage

```hcl
data "bitbucket_user" "reviewer" {
  uuid = "{account UUID}"
}

resource "bitbucket_project_default_reviewers" "infrastructure" {
  workspace = "myteam"
  project   = "TERRAFORM"
  reviewers = [data.bitbucket_user.reviewer.uuid]
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace of this project. Can be you or any team you
  have write access to.
* `project` - (Required) The key of the project.
* `reviewers` - (Required) A list of reviewers to use.

## Import

Project Default Reviewers can be imported using the workspace and project separated by a (`/`) and the end, e.g.,

```sh
terraform import bitbucket_project_default_reviewers.example myteam/terraform-code
```
