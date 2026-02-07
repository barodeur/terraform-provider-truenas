---
page_title: "truenas_api_key Resource - truenas"
subcategory: ""
description: |-
  Manages a TrueNAS API key.
---

# truenas_api_key (Resource)

Manages the lifecycle of a TrueNAS API key. API keys are used to authenticate with the TrueNAS API.

~> **Important:** The `key` attribute is only returned when the API key is first created. On subsequent reads, the value is preserved from Terraform state. After `terraform import`, `key` will be null.

## Example Usage

### Basic API Key

```terraform
resource "truenas_api_key" "example" {
  name = "terraform-managed-key"
}

output "api_key_value" {
  value     = truenas_api_key.example.key
  sensitive = true
}
```

### API Key with Expiration

```terraform
resource "truenas_api_key" "expiring" {
  name       = "short-lived-key"
  expires_at = "2025-12-31T23:59:59Z"
}
```

### API Key for a Specific User

```terraform
resource "truenas_api_key" "service_account" {
  name     = "service-key"
  username = "backupuser"
}
```

## Schema

### Required

- `name` (String) The name of the API key.

### Optional

- `username` (String) The user to create the key for. Defaults to the authenticated user.
- `expires_at` (String) Expiration date in ISO 8601 format. If omitted, the key does not expire.

### Read-Only

- `id` (Number) Numeric ID of the API key.
- `key` (String, Sensitive) The API key secret. Only available after creation; cannot be re-read from the TrueNAS API.
- `created_at` (String) Creation timestamp in ISO 8601 format.
- `revoked` (Boolean) Whether the API key has been revoked.

## Import

API keys can be imported using their numeric ID:

```shell
terraform import truenas_api_key.example 42
```

~> **Note:** The `key` attribute will be null after import since TrueNAS does not expose API key secrets after creation.
