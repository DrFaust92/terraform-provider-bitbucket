---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_pipeline_ssh_key"
sidebar_current: "docs-bitbucket-resource-pipeline-ssh-key"
description: |-
  Provides a Bitbucket Pipeline Ssh Key
---

# bitbucket\_pipeline\_ssh\_key

Provides a Bitbucket Pipeline SSH Key resource.

This allows you to manage your Pipeline SSH Keys for a repository.

* OAuth2 Scopes: `pipeline` and `pipeline:variable`
* API token permissions: `read:pipeline:bitbucket` and `admin:pipeline:bitbucket`

## Example Usage

```hcl
resource "bitbucket_pipeline_ssh_key" "test" {
  workspace   = "example"
  repository  = "example"
  public_key  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKqP3Cr632C2dNhhgKVcon4ldUSAeKiku2yP9O9/bDtY"
  private_key = "test-key"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The Workspace where the repository resides.
* `repository` - (Required) The Repository to create SSH key in.
* `public_key` - (Required) The SSH public key value in OpenSSH format.
* `private_key` - (Optional) The SSH private key value in OpenSSH format.
* `private_key_wo` - (Optional) Write-only alternative to `private_key`, never stored in Terraform state or plan output. Requires Terraform 1.11 or later. Conflicts with `private_key`. See [write-only arguments](https://developer.hashicorp.com/terraform/language/manage-sensitive-data/write-only).
* `private_key_wo_version` - (Optional) Required with `private_key_wo`. Terraform cannot detect changes to write-only values, change this value (e.g. `"2"`) to send an updated `private_key_wo`.

## Attributes Reference

## Import

Pipeline SSH Keys can be imported using their `workspace/repo-slug` ID, e.g.

```sh
terraform import bitbucket_pipeline_ssh_key.key workspace/repo-slug
```
