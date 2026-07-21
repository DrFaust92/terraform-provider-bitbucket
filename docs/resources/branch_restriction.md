---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_branch_restriction"
sidebar_current: "docs-bitbucket-resource-branch-restriction"
description: |-
  Provides a Bitbucket Branch Restriction
---

# bitbucket\_branch\_restriction

Provides a Bitbucket branch restriction resource.

This allows you for setting up branch restrictions for your repository.

* OAuth2 Scopes: `repository:admin`; using `users` additionally requires `account`
* API token permissions: `read:repository:bitbucket` and `admin:repository:bitbucket`; using `users` additionally requires `read:workspace:bitbucket`

## Example Usage

```hcl
resource "bitbucket_branch_restriction" "master" {
  owner      = "myteam"
  repository = "terraform-code"

  kind = "push"
  pattern = "master"

  users = [ "My Display Name" ]

  groups {
    slug = "my-group"
    owner = "my-owner"
  }
}
```

## Argument Reference

The following arguments are supported:

* `owner` - (Required) The owner of this repository. Can be you or any team you
  have write access to.
* `repository` - (Required) The name of the repository.
* `kind` - (Required) The type of restriction that is being applied. Valid values can be found in [docs](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-branch-restrictions/#api-group-branch-restrictions).
* `branch_match_kind` - (Optional) Indicates how the restriction is matched against a branch. The default is `glob`. Valid values: `branching_model`, `glob`.
* `branch_type` - (Optional) Apply the restriction to branches of this type. Active when `branch_match_kind` is `branching_model`. The branch type will be calculated using the branching model configured for the repository. Valid values: `feature`, `bugfix`, `release`, `hotfix`, `development`, `production`.
* `pattern` - (Optional) Apply the restriction to branches that match this pattern. Active when `branch_match_kind` is `glob`. Will be empty when `branch_match_kind` is `branching_model`.
* `users` - (Optional) A list of user display names to use. On create and update, each display name is mapped to its workspace member UUID. When the restriction is refreshed or imported, each user account returned by Bitbucket is stored in state using its `display_name`.
* `groups` - (Optional) A list of groups to use.
* `value` - (Optional) A value applied to the restriction kind. Currently only applicable to `require_passing_builds_to_merge`, `require_default_reviewer_approvals_to_merge` and `require_approvals_to_merge`.

## Import

Branch Restrictions can be imported using their `owner/repo-name/branch-restriction-id` ID, e.g.

```sh
terraform import bitbucket_branch_restriction.example my-account/my-repo/branch-rest-id
```
