---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_pipeline_oidc_config"
sidebar_current: "docs-bitbucket-data-pipeline-oidc-config"
description: |-
  Provides a data for a Bitbucket pipeline OIDC Config
---

# bitbucket\_pipeline\_oidc\_config

Provides a way to fetch data on a pipeline OIDC Config.

* OAuth2 Scopes: `pipeline`
* API token permissions: `read:pipeline:bitbucket`

## Example Usage

```hcl
data "bitbucket_pipeline_oidc_config" "example" {
  workspace = "example"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace to fetch pipeline oidc config.

## Attributes Reference

* `oidc_config` - The Json representing the OIDC config.
