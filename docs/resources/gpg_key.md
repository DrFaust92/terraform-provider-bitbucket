---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_gpg_key"
sidebar_current: "docs-bitbucket-resource-gpg-key"
description: |-
  Provides a Bitbucket GPG Key
---

# bitbucket\_gpg\_key

Provides a Bitbucket GPG Key resource.

This allows you to manage your GPG Keys for a user.

* OAuth2 Scopes: `account` and `account:write`
* API token permissions: `read:gpg-key:bitbucket`, `write:gpg-key:bitbucket`, and `delete:gpg-key:bitbucket`

## Example Usage

```hcl
resource "bitbucket_gpg_key" "test" {
  user = data.bitbucket_current_user.test.uuid
  key  = file("pubkey.asc")
  name = "test-key"
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) This can either be the UUID of the account, surrounded by curly-braces, for example: {account UUID}, OR an Atlassian Account ID.
* `key` - (Required) The GPG public key value in ASCII-armored format.
* `name` - (Optional) The user-defined label for the GPG key.

## Attributes Reference

* `fingerprint` - The GPG key fingerprint.
* `key_id` - The unique identifier for the GPG key.

## Import

GPG Keys can be imported using their `user/fingerprint` ID, e.g.

```sh
terraform import bitbucket_gpg_key.test user-id/fingerprint
```
