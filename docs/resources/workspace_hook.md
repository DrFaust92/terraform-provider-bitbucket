---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_workspace_hook"
sidebar_current: "docs-bitbucket-resource-workspace-hook"
description: |-
  Provides a Bitbucket Workspace Webhook
---

# bitbucket\_workspace\_hook

Provides a Bitbucket workspace hook resource.

This allows you to manage your webhooks on a workspace.

* OAuth2 Scopes: `webhook`
* API token permissions: `read:webhook:bitbucket`, `write:webhook:bitbucket`, and `delete:webhook:bitbucket`

## Example Usage

```hcl
resource "bitbucket_workspace_hook" "deploy_on_push" {
  workspace   = "myteam"
  url         = "https://mywebhookservice.mycompany.com/deploy-on-push"
  description = "Deploy the code via my webhook"

  events = [
    "repo:push",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace of this repository. Can be you or any team you
  have write access to.
* `url` - (Required) Where to POST to.
* `description` - (Required) The name / description to show in the UI.
* `events` - (Required) The events this webhook is subscribed to. Valid values can be found at [Bitbucket Webhook Docs](https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-post).
* `active` - (Optional) Whether the webhook configuration is active or not (Default: `true`).
* `skip_cert_verification` - (Optional) Whether to skip certificate verification or not (Default: `true`).
* `secret` - (Optional) A Webhook secret value. Passing a null or empty secret or not passing a secret will leave the webhook's secret unset. This value is not returned on read and cannot resolve diffs or be imported as its not returned back from bitbucket API.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `uuid` - The UUID of the workspace webhook.
* `secret_set` - Whether a webhook secret is set.
* `history_enabled` - Whether a webhook history is enabled.

## Import

Hooks can be imported using their `workspace/hook-id` ID, e.g.

```sh
terraform import bitbucket_workspace_hook.hook my-account/hook-id
```
