---
page_title: "truenas_api_key Data Source - truenas"
subcategory: ""
description: |-
  Looks up an existing TrueNAS API key by name or ID.
---

# truenas_api_key (Data Source)

Use this data source to look up an existing TrueNAS API key by name or ID.

~> **Note:** This data source does not return the API key secret. The secret is only available at creation time via the `truenas_api_key` resource.

## Example Usage

### Look Up by Name

```terraform
data "truenas_api_key" "example" {
  name = "terraform-managed-key"
}
```

### Look Up by ID

```terraform
data "truenas_api_key" "example" {
  id = 42
}
```

## Schema

At least one of `id` or `name` must be provided.

### Optional

- `id` (Number) Numeric ID of the API key.
- `name` (String) Name of the API key.

### Read-Only

- `username` (String) Username associated with the API key.
- `expires_at` (String) Expiration date in ISO 8601 format, if set.
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `revoked` (Boolean) Whether the API key has been revoked.
