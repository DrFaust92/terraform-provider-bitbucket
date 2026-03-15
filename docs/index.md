---
layout: "bitbucket"
page_title: "Provider: Bitbucket"
sidebar_current: "docs-bitbucket-index"
description: |-
  The Bitbucket provider to interact with repositories, projects, etc..
---

# Bitbucket Provider

The Bitbucket provider allows you to manage resources including repositories,
webhooks, and default reviewers.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Bitbucket Provider using OAuth (recommended)
provider "bitbucket" {
  oauth_client_id     = "my-oauth-client-id"
  oauth_client_secret = "my-oauth-client-secret"
}

# Alternatively, using Basic Auth with an API token
# Note: Bitbucket App Passwords are deprecated - use an API token with your Atlassian account email instead
provider "bitbucket" {
  username = "gob@bluth.example.com" # Atlassian account email
  password = "ATATT3x..."            # API token from https://id.atlassian.com/manage-profile/security/api-tokens
}

resource "bitbucket_repository" "illusions" {
  owner      = "theleagueofmagicians"
  name       = "illusions"
  scm        = "hg"
  is_private = true
}

resource "bitbucket_project" "project" {
  owner      = "theleagueofmagicians" # must be a team
  name       = "illusions-project"
  key        = "ILLUSIONSPROJ"
  is_private = true
}
```

## Argument Reference

The following arguments are supported in the `provider` block:

* `username` - (Optional) Username to use for authentication via [Basic
  Auth](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#basic-auth).
  When using an API token, this must be your **Atlassian account email address**.
  You can also set this via the `BITBUCKET_USERNAME` environment variable.
  If configured, requires `password` to be configured as well.

* `password` - (Optional) Password to use for authentication via [Basic
  Auth](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#basic-auth).
  It is recommended to use an [API Token](https://support.atlassian.com/bitbucket-cloud/docs/using-api-tokens/)
  created at [Atlassian Account Settings](https://id.atlassian.com/manage-profile/security/api-tokens)
  as your password, with your Atlassian account email as the username.
  **Note:** Bitbucket App Passwords are deprecated and will stop working on June 9, 2026.
  You can also set this via the `BITBUCKET_PASSWORD` environment variable. If
  configured, requires `username` to be configured as well.

* `oauth_client_id` - (Optional) OAuth client ID to use for authentication via
  [Client Credentials
  Grant](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#3--client-credentials-grant--4-4-).
  You can also set this via the `BITBUCKET_OAUTH_CLIENT_ID` environment
  variable. If configured, requires `oauth_client_secret` to be configured as
  well.

* `oauth_client_secret` - (Optional) OAuth client secret to use for authentication via
  [Client Credentials
  Grant](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#3--client-credentials-grant--4-4-).
  You can also set this via the `BITBUCKET_OAUTH_CLIENT_SECRET` environment
  variable. If configured, requires `oauth_client_id` to be configured as well.

* `oauth_token` - (Optional) An OAuth access token used for authentication via
  [OAuth](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#oauth-2-0).
  You can also set this via the `BITBUCKET_OAUTH_TOKEN` environment variable.

## Permission Scopes

To interact with the Bitbucket API, an [API Token](https://support.atlassian.com/bitbucket-cloud/docs/api-tokens/)
or [OAuth Client Credentials](https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/)
are required.

API tokens and OAuth client credentials are limited in scope, each API
requires certain scope to interact with, each resource doc will specify what
scopes are required to use that resource:

* [OAuth 2.0 scopes](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#bitbucket-oauth-2-0-scopes)
* [API token permissions](https://support.atlassian.com/bitbucket-cloud/docs/api-token-permissions/)
