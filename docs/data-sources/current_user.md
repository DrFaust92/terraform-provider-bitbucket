---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_current_user"
sidebar_current: "docs-bitbucket-data-current-user"
description: |-
  Provides data for the current Bitbucket user
---

# bitbucket\_current\_user

Provides a way to fetch data of the current user.

* OAuth2 Scopes: `account`
* API token permissions: `read:user:bitbucket`

## Example Usage

```hcl
data "bitbucket_current_user" "example" {}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

* `username` - The Username.
* `uuid` - the uuid that bitbucket users to connect a user to various objects
* `display_name` - the display name that the user wants to use for GDPR
* `email` - A Set of emails associated to current user. See [Email](#email) below.

### Email

* `is_primary` - Whether is primary email for the user.
* `is_confirmed` - Whether the email is confirmed.
* `email` - The email address.
