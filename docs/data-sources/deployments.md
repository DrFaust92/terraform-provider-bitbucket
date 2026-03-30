---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_deployments"
sidebar_current: "docs-bitbucket-data-deployments"
description: |-
  Provides a data for Bitbucket Deployments
---

# bitbucket\_deployments

Provides a way to fetch data on Deployments.

* OAuth2 Scopes: `pipeline`
* API token permissions: `read:pipeline:bitbucket`

## Example Usage

```hcl
data "bitbucket_deployments" "example" {
  repository = "example"
  workspace  = "example"
}
```

## Argument Reference

The following arguments are supported:

* `repository` - (Required) The repository name.
* `workspace` - (Required) The workspace name.

## Attributes Reference

* `uuids` - UUIDs of deployments for a repository.
* `names` - Names of deployments for a repository.
