---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_pipeline_schedule"
sidebar_current: "docs-bitbucket-resource-pipeline-schedule"
description: |-
  Provides a Bitbucket Pipeline Schedule
---

# bitbucket\_pipeline\_schedule

Provides a Bitbucket Pipeline Schedule resource.

This allows you to manage your Pipeline Schedules for a repository.

* OAuth2 Scopes: `pipeline` and `pipeline:write`
* API token permissions: `read:pipeline:bitbucket` and `write:pipeline:bitbucket`

## Example Usage

```hcl
resource "bitbucket_pipeline_schedule" "test" {
  workspace     = "example"
  repository   = bitbucket_repository.test.name
  cron_pattern = "0 30 * * * ? *"
  enabled      = true

  target {
    ref_name = "master"
	  ref_type = "branch"
	  selector {
      pattern = "staging"
	  }
  }
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The Workspace where the repository resides.
* `repository` - (Required) The Repository to create schedule in.
* `enabled` - (Required) Whether the schedule is enabled.
* `cron_pattern` - (Required) The cron expression that the schedule applies.
* `target` - (Required) Schedule Target definition. See [Target](#target) below.

### Target

* `ref_name` - (Required) The name of the reference.
* `ref_type` - (Required) The type of reference. Valid values are `branch` and `tag`.
* `selector` - (Required) Selector spec. See [Selector](#selector) below.

#### Selector

* `pattern` - (Required) The name of the matching pipeline definition.
* `type` - (Optional) Selector type. Default value is `branches`.

## Attributes Reference

* `uuid` - The UUID identifying the schedule.

## Import

Pipeline Schedules can be imported using their `workspace/repo-slug/uuid` ID, e.g.

```sh
terraform import bitbucket_pipeline_schedule.schedule workspace/repo-slug/uuid
```
