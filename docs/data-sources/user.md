---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_user"
sidebar_current: "docs-bitbucket-data-user"
description: |-
  Provides data for a Bitbucket user
---

# bitbucket\_user

Provides a way to fetch data on a user.

* OAuth2 Scopes: `account`
* API token permissions: `read:user:bitbucket`

## Example Usage

```hcl
data "bitbucket_user" "reviewer" {
  uuid = "{account UUID}"
}
```

## Argument Reference

The following arguments are supported (At least of of is required):

* `uuid` - (Optional) The UUID that bitbucket users to connect a user to various objects

## Attributes Reference

* `uuid` - the uuid that bitbucket users to connect a user to various objects
* `display_name` - the display name that the user wants to use for GDPR
