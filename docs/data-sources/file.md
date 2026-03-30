---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_file"
sidebar_current: "docs-bitbucket-data-bitbucket-file"
description: |-
  Datasource to retrieve file content or metadata information
---

# bitbucket_file (Data Source)

Datasource to retrieve file content or metadata information

* OAuth2 Scopes: `repository`
* API token permissions: `read:repository:bitbucket`

## Example Usage

```terraform
# Basic example (slugs and branch name) returns file contents
data "bitbucket_file" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
	commit = "main"
	path = "path/to/my/doc.py"
}

# Basic example (using commit hash) returns file contents
data "bitbucket_file" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
	commit = "a1b2c3d4e5f67890abcdef1234567890abcdef"
	path = "path/to/my/doc.py"
}

# Basic example (using UUIDs and commit hash) returns file contents
data "bitbucket_file" "test" {
	workspace = "{f772f004-c268-4698-b49f-9d8415981464}"
	repo_slug = "{2c3a2eee-fa63-4b60-afd4-97319245e79e}"
	commit = "a1b2c3d4e5f67890abcdef1234567890abcdef"
	path = "path/to/my/doc.py"
}

# Basic example return file metadata (no commit/links)
data "bitbucket_file" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
	commit = "main"
	path = "path/to/my/doc.py"
    format = "meta"
}

# Basic example return file metadata with links
data "bitbucket_file" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
	commit = "main"
	path = "path/to/my/doc.py"
    format = "meta"
    include_links = true
}

# Basic example return file metadata with commit information
data "bitbucket_file" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
	commit = "main"
	path = "path/to/my/doc.py"
    format = "meta"
    include_commit = true
}

# Basic example return file metadata with commit and link information
data "bitbucket_file" "test" {
	workspace = "myworkspace"
	repo_slug = "myrepo"
	commit = "main"
	path = "path/to/my/doc.py"
    format = "meta"
    include_commit = true
    include_commit_links = true
}
```

## Argument Reference

The following arguments are supported:

- `commit` - (Required) Commit hash or branch name
- `path` - (Required) Path to file (starting from commit)
- `repo_slug` - (Required) Repo slug or UUID
- `workspace` - (Required) Workspace slug or UUID
- `format` - Format if file to return: content/base64 content or metadata.
- `include_commit` (Boolean) Whether to include the commit for the file metadata or not.
- `include_commit_links` (Boolean) Whether to include the commit links for the file metadata or not.
- `include_links` (Boolean) Whether to include the links for the file metadata or not.

## Attributes Reference

- `content` - Raw string content of path return (not escaped).
- `content_b64` - Base64-encoded version of path return, safe for embedding.
- `id` - The ID of this resource.
- `metadata` (List of Object) Parsed metadata of path (JSON/XML), if available (see [below for nested schema](#nestedatt--metadata))

<a id="nestedatt--metadata"></a>
### Nested Schema for `metadata`

- `commit` - commit information (see [below for nested schema](#nestedobjatt--metadata--commit))
- `escaped_path` - escaped file path
- `link` - commit link information (see [below for nested schema](#nestedobjatt--metadata--link))
- `mime_type` - file MIME type
- `path` - file path
- `size` - file size
- `type` - file type

<a id="nestedobjatt--metadata--commit"></a>
### Nested Schema for `metadata.commit`

- `hash` - Commit hash
- `link` - Commit links (see [below for nested schema](#nestedobjatt--metadata--commit--link))
- `type` - Commit type

<a id="nestedobjatt--metadata--commit--link"></a>
### Nested Schema for `metadata.commit.link`

- `html` - HTML link (see [below for nested schema](#nestedobjatt--metadata--commit--link--html))
- `self` - Self link (see [below for nested schema](#nestedobjatt--metadata--commit--link--self))

<a id="nestedobjatt--metadata--commit--link--html"></a>
### Nested Schema for `metadata.commit.link.html`

- `href` - URL link

<a id="nestedobjatt--metadata--commit--link--self"></a>
### Nested Schema for `metadata.commit.link.self`

- `href` - URL link

<a id="nestedobjatt--metadata--link"></a>
### Nested Schema for `metadata.link`

- `history` - File history link (see [below for nested schema](#nestedobjatt--metadata--link--history))
- `meta` - File metadata link (see [below for nested schema](#nestedobjatt--metadata--link--meta))
- `self` - File self link (see [below for nested schema](#nestedobjatt--metadata--link--self))

<a id="nestedobjatt--metadata--link--history"></a>
### Nested Schema for `metadata.link.history`

- `href` - URL link

<a id="nestedobjatt--metadata--link--meta"></a>
### Nested Schema for `metadata.link.meta`

- `href` - URL link

<a id="nestedobjatt--metadata--link--self"></a>
### Nested Schema for `metadata.link.self`

- `href` - URL link
