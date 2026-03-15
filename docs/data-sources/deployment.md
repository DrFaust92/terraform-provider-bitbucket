---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_deployment"
sidebar_current: "docs-bitbucket-data-deployment"
description: |-
  Provides a data for a Bitbucket Deployment
---

# bitbucket\_deployment

Provides a way to fetch data on a Deployment.

* OAuth2 Scopes: `pipeline`
* API token permissions: `read:pipeline:bitbucket`

## Example Usage

```hcl
data "bitbucket_deployment" "example" {
  uuid       = "example"
  repository = "example"
  workspace  = "example"
}
```

## Argument Reference

The following arguments are supported:

* `uuid` - (Required) The environment UUID.
* `repository` - (Required) The repository name.
* `workspace` - (Required) The workspace name.

## Attributes Reference

* `stage` - The stage (Test, Staging, Production).
* `name` - The name of the environment.
