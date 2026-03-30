---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_default_reviewers"
sidebar_current: "docs-bitbucket-resource-default-reviewers"
description: |-
  Provides support for setting up default reviews for bitbucket.
---

# bitbucket\_default\_reviewers

Provides support for setting up default reviewers for your repository. You must however have the UUID of the user available. Since Bitbucket has removed usernames from its APIs the best case is to use the UUID via the data provider.

* OAuth2 Scopes: `pullrequest` and `repository:admin`
* API token permissions: `admin:repository:bitbucket`

## Example Usage

```hcl
data "bitbucket_user" "reviewer" {
  uuid = "{account UUID}"
}

resource "bitbucket_default_reviewers" "infrastructure" {
  owner      = "myteam"
  repository = "terraform-code"

  reviewers = [data.bitbucket_user.reviewer.uuid]
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) The owner of this repository. Can be you or any team you
  have write access to.
* `repository` - (Required) The name of the repository.
* `reviewers` - (Required) A list of reviewers to use.

## Import

Default Reviewers can be imported using the owner and repo separated by a (`/`) and the string `reviewers` and the end, e.g.,

```sh
terraform import bitbucket_default_reviewers.example myteam/terraform-code/reviewers
```
