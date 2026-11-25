---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_pipeline_runner"
sidebar_current: "docs-bitbucket-resource-pipeline-runner"
description: |-
  Provides a Bitbucket Pipelines self-hosted runner
---

# bitbucket\_pipeline\_runner

Manages a [self-hosted runner](https://support.atlassian.com/bitbucket-cloud/docs/runners/) for Bitbucket Pipelines, for a workspace or for one repository.

Creating the runner only registers it with Bitbucket. Start the runner on your own host with the OAuth credentials this resource exports.

* OAuth2 Scopes: `runner` and `runner:write`
* API token permissions: `read:runner:bitbucket` and `write:runner:bitbucket`

## Example Usage

```hcl
# A workspace runner, available to every repository in the workspace.
resource "bitbucket_pipeline_runner" "linux" {
  workspace = "example"
  name      = "linux-large"
  labels    = ["self.hosted", "linux", "large"]
}

# A repository runner.
resource "bitbucket_pipeline_runner" "repo" {
  workspace  = "example"
  repository = bitbucket_repository.app.name
  name       = "app-runner"
  labels     = ["self.hosted", "linux"]
}
```

## Argument Reference

* `workspace` - (Required) The workspace of the runner. Changing it creates a new runner.
* `repository` - (Optional) The repository slug, for a repository runner. Omit it for a workspace runner. Changing it creates a new runner.
* `name` - (Required) The name of the runner.
* `labels` - (Required) Labels that pipeline steps use to select the runner with `runs-on`, for example `self.hosted` and `linux`.

## Attributes Reference

* `id` - `workspace/uuid` for a workspace runner, `workspace/repository/uuid` for a repository runner.
* `uuid` - The UUID of the runner.
* `status` - The status of the runner, for example `UNREGISTERED` or `ONLINE`.
* `cordoned` - Whether the runner is prevented from accepting new steps.
* `oauth_client_id` - The OAuth client ID the runner authenticates with.
* `oauth_client_secret` - (Sensitive) The OAuth client secret the runner authenticates with. Bitbucket returns it only when the runner is created, so it is stored in state and is empty after an import.
* `oauth_token_endpoint` - The OAuth token endpoint the runner uses.
* `oauth_audience` - The audience of the runner's OAuth tokens.

## Import

Workspace runners can be imported with `workspace/uuid`, and repository runners with `workspace/repository/uuid`:

```sh
terraform import bitbucket_pipeline_runner.linux 'example/{6b4d5d7e-0000-0000-0000-000000000000}'
terraform import bitbucket_pipeline_runner.repo 'example/app/{6b4d5d7e-0000-0000-0000-000000000000}'
```

`oauth_client_secret` is not available after an import.
