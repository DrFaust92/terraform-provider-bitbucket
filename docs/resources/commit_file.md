---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_commit_file"
sidebar_current: "docs-bitbucket-resource-commit-file"
description: |-
  Commit a file
---

# bitbucket\_commit\_file

Commit a file.

This resource allows you to create a commit within a Bitbucket repository.

* OAuth2 Scopes: `repository:write`
* API token permissions: `write:repository:bitbucket`

## Example Usage

```hcl
resource "bitbucket_commit_file" "test" {
  filename       = "README.md"
  content        = "abc"
  repo_slug      = "test"
  workspace      = "test"
  commit_author  = "Test <test@test.local>"
  branch         = "main"
  commit_message = "test"
}
```

## Argument Reference

The following arguments are supported:

* `workspace` - (Required) The workspace id.
* `repo_slug` - (Required) The repository slug.
* `filename` - (Required) The path of the file to manage.
* `content` - (Required) The file content.
* `commit_author` - (Required) Committer author to use.
* `branch` - (Required) Git branch.
* `commit_message` - (Required) The message of the commit.
