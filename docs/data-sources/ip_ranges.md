---
layout: "bitbucket"
page_title: "Bitbucket: bitbucket_ip_ranges"
sidebar_current: "docs-bitbucket-data-ip-ranges"
description: |-
  Provides a data for Bitbucket IP Ranges
---

# bitbucket\_ip\_ranges

Provides a way to fetch IP Ranges for whitelisting.

* OAuth2 Scopes: `none`
* API token permissions: `none`

## Example Usage

```hcl
data "bitbucket_ip_ranges" "example" {}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

* `ranges` - A Set of IP Ranges. See [Ranges](#ranges) below.

### Ranges

* `network` - The network of the range.
* `mask_len` - The make length of the range.
* `cidr` - The CIDR of the range.
* `mask` - More mask of the range.
* `regions` - A Set of regions the range is associated with.
* `products` - A Set of Atlasian products (Bitbucket, Jira, etc) the range is associated with.
* `directions` - A Set of directions (Ingress/Egress) the range is associated with.
* `perimeter` - The allowed perimeter of the range.
