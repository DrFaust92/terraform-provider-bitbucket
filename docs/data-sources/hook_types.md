---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_hook_types"
sidebar_current: "docs-bitbucket-data-hook-types"
description: |-
  Provides a data for Bitbucket Hook Event Types
---

# bitbucket\_hook\_types

Provides a way to fetch data of hook types.

* OAuth2 Scopes: `webhook`
* API token permissions: `read:webhook:bitbucket`

## Example Usage

```hcl
data "bitbucket_hook_types" "example" {
  subject_type = "workspace"
}
```

## Argument Reference

* `subject_type` - A resource or subject type. Valid values are `workspace`, `user`, `repository`, `team`.

## Attributes Reference

* `hook_types` - A Set of Hook Event Types. See [Hook Types](#hook-types) below.

### Hook Types

* `event` - The event identifier.
* `category` - The category this event belongs to.
* `label` - Summary of the webhook event type.
* `description` - More detailed description of the webhook event type.
