---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_pipeline_ssh_known_host"
sidebar_current: "docs-bitbucket-resource-pipeline-ssh-known-host"
description: |-
  Provides a Bitbucket Pipeline SSH Known Host
---

# bitbucket\_pipeline\_ssh\_known_host

Provides a Bitbucket Pipeline SSH Known Host resource.

This allows you to manage your Pipeline SSH Known Hosts for a repository.

* OAuth2 Scopes: `pipeline` and `pipeline:variable`
* API token permissions: `read:pipeline:bitbucket` and `admin:pipeline:bitbucket`

## Example Usage

```hcl
resource "bitbucket_pipeline_ssh_known_host" "test" {
  workspace  = "example"
  repository = bitbucket_repository.test.name
  hostname   = "[example.com]:22"

  public_key {
    key_type = "ssh-ed25519"
    key      = "AAAAC3NzaC1lZDI1NTE5AAAAIKqP3Cr632C2dNhhgKVcon4ldUSAeKiku2yP9O9/bDtY"
  }
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The Workspace where the repository resides.
* `repository` - (Required) The Repository to create config for the known host in.
* `hostname` - (Required) The hostname of the known host.
* `public_key` - (Required) The Public key config for the known host.

### Public Key

* `key_type` - The type of the public key. Valid values are `ssh-ed25519`, `ecdsa-sha2-nistp256`, `ssh-rsa`, and `ssh-dss`.
* `key` - The plain public key.

## Attributes Reference

* `uuid` - The UUID identifying the known host.
* `public_key.0.md5_fingerprint` - The MD5 fingerprint of the public key.
* `public_key.0.sha256_fingerprint` - The SHA-256 fingerprint of the public key.

## Import

Pipeline SSH Known Hosts can be imported using their `workspace/repo-slug/uuid` ID, e.g.

```sh
terraform import bitbucket_pipeline_ssh_known_host.key workspace/repo-slug/uuid
```
