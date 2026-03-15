---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_deployment"
sidebar_current: "docs-bitbucket-resource-deployment"
description: |-
  Manage your pipelines repository deployment environments
---


# bitbucket\_deployment

This resource allows you to setup pipelines deployment environments.

* OAuth2 Scopes: `pipeline`
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
```

## Argument Reference

* `name` - (Required) The name of the deployment environment
* `stage` - (Required) The stage (Test, Staging, Production)
* `repository` - (Required) The repository ID to which you want to assign this deployment environment to
* `restrictions` - (Optional) Deployment restrictions. See [Restrictions](#restrictions) below.

### Restrictions

* `admin_only` - (Required) Only Admins can deploy this deployment stage.

## Attributes Reference

* `uuid` - (Computed) The UUID identifying the deployment.

## Import

Deployments can be imported using their `repository/uuid` ID, e.g.

```sh
terraform import bitbucket_deployment.example repository/uuid
```
