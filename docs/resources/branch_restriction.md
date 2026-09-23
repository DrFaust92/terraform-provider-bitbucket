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

* OAuth2 Scopes: `repository:admin`
* API token permissions: `read:repository:bitbucket` and `admin:repository:bitbucket`

## Example Usage

```hcl
resource "bitbucket_branch_restriction" "master" {
  owner      = "myteam"
  repository = "terraform-code"

  kind = "push"
  pattern = "master"

  users = [ "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}" ] # UUID, or a legacy username

  groups {
    slug = "my-group"
    owner = "myteam" # workspace slug (preferred); UUID form also accepted
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
* `users` - (Optional) A list of users to use. Accepts Bitbucket account UUIDs (e.g. `{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}`) or legacy usernames. **The Bitbucket API no longer returns usernames** (removed as part of Bitbucket's GDPR changes) — a config that still uses a username will produce a **perpetual diff** on every `terraform plan`/`apply` because the API accepts usernames on write but always returns UUIDs on read. When this situation is detected, the provider emits a warning during plan listing the UUID and display name of each affected user. To migrate: replace the username(s) in your config with the corresponding UUIDs shown in the warning — this triggers a one-time apply, after which the diff disappears permanently. If you cannot migrate immediately, add `lifecycle { ignore_changes = [users] }` to suppress the diff (note: Terraform will then no longer detect user changes on this resource).
* `groups` - (Optional) A list of groups to use. `owner` identifies the workspace that owns the group. The Bitbucket API returns the workspace **slug** (e.g. `myteam`) for this field; using the workspace UUID (e.g. `{c0ffee00-...}`) is also accepted on the write path and will be round-tripped as a UUID if the API includes the workspace object in its response. If you previously configured `owner` as a UUID and now see a plan diff where the UUID is replaced by the slug, update `owner` to use the slug (one-time migration), or add `lifecycle { ignore_changes = [groups] }` to suppress it.
* `value` - (Optional) A value applied to the restriction kind. Currently only applicable to `require_passing_builds_to_merge`, `require_default_reviewer_approvals_to_merge` and `require_approvals_to_merge`.

## Import

Branch Restrictions can be imported using their `owner/repo-name/branch-restriction-id` ID, e.g.

```sh
terraform import bitbucket_branch_restriction.example my-account/my-repo/branch-rest-id
```
