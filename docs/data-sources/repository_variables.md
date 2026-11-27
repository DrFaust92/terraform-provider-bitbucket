---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_repository_variables"
sidebar_current: "docs-bitbucket-data-repository-variables"
description: |-
  Provides the pipeline variables of a Bitbucket repository
---

# bitbucket\_repository\_variables

Lists the pipeline variables of a repository.

* OAuth2 Scopes: `pipeline` and `pipeline:variable`
* API token permissions: `read:pipeline:bitbucket`

## Example Usage

```hcl
data "bitbucket_repository_variables" "example" {
  workspace  = "example"
  repository = "example"
}
```

Adopt a variable that may already exist, for example one created outside Terraform or by another provider:

```hcl
locals {
  existing = { for v in data.bitbucket_repository_variables.example.variables : v.key => v.uuid }
}

import {
  for_each = contains(keys(local.existing), "DEBUG") ? toset(["DEBUG"]) : toset([])

  to = bitbucket_repository_variable.debug
  id = "example/example/${each.key}/${local.existing[each.key]}"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace slug.
* `repository` - (Required) The repository slug.

## Attributes Reference

* `variables` - The repository's pipeline variables, sorted by key. Each has:
  * `uuid` - The UUID identifying the variable.
  * `key` - The variable's key.
  * `value` - (Sensitive) The variable's value. Empty for secured variables, whose value the API never returns.
  * `secured` - Whether the variable is secured.
