---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_dynamic_pipelines_provider"
sidebar_current: "docs-bitbucket-resource-dynamic-pipelines-provider"
description: |-
  Manages the Dynamic Pipelines provider selector for a Bitbucket Cloud repository.
---

# bitbucket\_dynamic\_pipelines\_provider

Manages the Dynamic Pipelines provider selected for a Bitbucket Cloud repository.

This resource uses Bitbucket's **undocumented `/internal` API**:

`GET`/`POST` `https://api.bitbucket.org/internal/repositories/{workspace}/{repository}/pipelines-config/dynamic-pipelines-provider`

REST 2.0 does not currently expose this selector. Atlassian may change or remove
the internal endpoint without notice. Use this resource knowing that a Bitbucket
change can break it.

Authentication uses the same provider credentials as every other resource
(Basic Auth or OAuth). Changing the selector typically requires repository admin
permission.

## Example Usage

```hcl
resource "bitbucket_dynamic_pipelines_provider" "example" {
  workspace  = "example-workspace"
  repository = "example-repository"
  app_ari    = "ari:cloud:ecosystem::extension/00000000-0000-0000-0000-000000000001/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/static/example-dynamic-pipelines-provider"
}
```

If the remote selector already matches `app_ari`, create adopts that state and
does **not** POST. If the remote selector is a different ARI, create fails
instead of overwriting it.

## Argument Reference

The following arguments are supported:

* `workspace` - (Required, Forces new resource) The workspace slug.
* `repository` - (Required, Forces new resource) The repository slug.
* `app_ari` - (Required) The Forge Dynamic Pipelines provider ARI to select.

## Attributes Reference

* `fallback_enabled` - Whether Bitbucket fallback to static YAML is enabled. This value is read from the remote API and is not configurable here.

## Delete behavior

Delete is **state-only**.

Terraform will remove `bitbucket_dynamic_pipelines_provider` from state and will
**not** send HTTP DELETE, POST, or an empty `appAri`. The remote Bitbucket
selector is left intact. Removing this resource from configuration does not
disable Dynamic Pipelines and does not revert the provider.

## Import

This resource can be imported using `workspace/repository`, e.g.

```sh
terraform import bitbucket_dynamic_pipelines_provider.example example-workspace/example-repository
```
