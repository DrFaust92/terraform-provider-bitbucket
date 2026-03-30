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

* `deployment` - (Required) The deployment ID you want to assign this variable to.
* `key` - (Required) The unique name of the variable.
* `value` - (Required) The value of the variable.
* `secured` - (Optional)  If true, this variable will be treated as secured. The value will never be exposed in the logs or the REST API.

## Attributes Reference

* `uuid` - (Computed) The UUID identifying the variable.

## Import

Deployment Variables can be imported using their `deployment-id/uuid` ID, e.g.

```sh
terraform import bitbucket_deployment_variable.example deployment-id/uuid
```
