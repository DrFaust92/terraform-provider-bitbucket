---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_ssh_key"
sidebar_current: "docs-bitbucket-resource-ssh-key"
description: |-
  Provides a Bitbucket SSH Key
---

# bitbucket\_ssh\_key

Provides a Bitbucket SSH Key resource.

This allows you to manage your SSH Keys for a user.

* OAuth2 Scopes: `account` and `account:write`
* API token permissions: `read:ssh-key:bitbucket`, `write:ssh-key:bitbucket`, and `delete:ssh-key:bitbucket`

## Example Usage

```hcl
resource "bitbucket_ssh_key" "test" {
  user  = data.bitbucket_current_user.test.uuid
  key   = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKqP3Cr632C2dNhhgKVcon4ldUSAeKiku2yP9O9/bDtY"
  label = "test-key"
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) This can either be the UUID of the account, surrounded by curly-braces, for example: {account UUID}, OR an Atlassian Account ID.
* `key` - (Required) The SSH public key value in OpenSSH format.
* `label` - (Optional) The user-defined label for the SSH key

## Attributes Reference

* `uuid` - The SSH key's UUID value.
* `comment` - The comment parsed from the SSH key (if present)
* `fingerprint` - The SHA-256 fingerprint of the key.
* `expires_on` - When the key expires, in RFC 3339 format. Empty if it does not expire.
* `last_used` - When the key was last used, in RFC 3339 format. Empty if it has not been used.

## Import

SSH Keys can be imported using their `user-id/key-id` ID, e.g.

```sh
terraform import bitbucket_ssh_key.key user-id/key-id
```
