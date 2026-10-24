---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_workspace_variable"
sidebar_current: "docs-bitbucket-resource-workspace-variable"
description: |-
  Manage variables for your pipelines workspace environments
---


# bitbucket\_workspace\_variable

This resource allows you to configure workspace variables.

* OAuth2 Scopes: `pipeline` and `pipeline:variable`
* API token permissions: `read:pipeline:bitbucket` and `admin:pipeline:bitbucket`

## Example Usage

```hcl
resource "bitbucket_workspace_variable" "country" {
  workspace = bitbucket_workspace.test.id
  key       = "COUNTRY"
  value     = "Kenya"
  secured   = false
}
```

## Argument Reference

* `workspace` - (Required) The workspace ID you want to assign this variable to.
* `key` - (Required) The unique name of the variable.
* `value` - (Optional) Exactly one of `value` or `value_wo` must be set. The value of the variable.
* `value_wo` - (Optional) Write-only alternative to `value`, never stored in Terraform state or plan output. Requires Terraform 1.11 or later. Conflicts with `value`. See [write-only arguments](https://developer.hashicorp.com/terraform/language/manage-sensitive-data/write-only).
* `value_wo_version` - (Optional) Required with `value_wo`. Terraform cannot detect changes to write-only values, change this value (e.g. `"2"`) to send an updated `value_wo`.
* `secured` - (Optional)  If true, this variable will be treated as secured. The value will never be exposed in the logs or the REST API.

## Attributes Reference

* `uuid` - (Computed) The UUID identifying the variable.

## Import

Workspace Variables can be imported using their `workspace-id/uuid` ID, e.g.

```sh
terraform import bitbucket_workspace_variable.example workspace-id/uuid
```
