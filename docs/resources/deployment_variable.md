---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_deployment_variable"
sidebar_current: "docs-bitbucket-resource-deployment-variable"
description: |-
  Manage variables for your pipelines deployment environments
---


# bitbucket\_deployment\_variable

This resource allows you to configure deployment variables.

* OAuth2 Scopes: `pipeline:variable`
* API token permissions: `read:pipeline:bitbucket` and `admin:pipeline:bitbucket`

## Example Usage

```hcl
resource "bitbucket_repository" "monorepo" {
  owner             = "gob"
  name              = "illusions"
  pipelines_enabled = true
}
resource "bitbucket_deployment" "test" {
  repository = bitbucket_repository.monorepo.id
  name       = "test"
  stage      = "Test"
}
resource "bitbucket_deployment_variable" "country" {
  deployment = bitbucket_deployment.test.id
  key        = "COUNTRY"
  value      = "Kenya"
  secured    = false
}
```

## Argument Reference

* `deployment` - (Required) The deployment ID (`bitbucket_deployment.example.id`, not its `uuid`) you want to assign this variable to.
* `key` - (Required) The unique name of the variable.
* `value` - (Optional) Exactly one of `value` or `value_wo` must be set. The value of the variable.
* `value_wo` - (Optional) Write-only alternative to `value`, never stored in Terraform state or plan output. Requires Terraform 1.11 or later. Conflicts with `value`. See [write-only arguments](https://developer.hashicorp.com/terraform/language/manage-sensitive-data/write-only).
* `value_wo_version` - (Optional) Required with `value_wo`. Terraform cannot detect changes to write-only values, change this value (e.g. `"2"`) to send an updated `value_wo`.
* `secured` - (Optional)  If true, this variable will be treated as secured. The value will never be exposed in the logs or the REST API.

## Attributes Reference

* `uuid` - (Computed) The UUID identifying the variable.

## Import

Deployment Variables can be imported using their `workspace/repo-slug:{deployment-uuid}/{variable-uuid}` ID, e.g.

```sh
terraform import bitbucket_deployment_variable.example 'my-workspace/my-repo:{c8d9e8d4-5e2f-4b55-9f3a-3c4d5e6f7a8b}/{1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d}'
```
