---
page_title: "TrueNAS Provider"
subcategory: ""
description: |-
  The TrueNAS provider manages resources on TrueNAS Scale via its WebSocket JSON-RPC API.
---

# TrueNAS Provider

The TrueNAS provider lets you manage [TrueNAS Scale](https://www.truenas.com/truenas-scale/) resources using the WebSocket JSON-RPC 2.0 API.

## Authentication

The provider authenticates using a TrueNAS API key. You can generate one from the TrueNAS web UI under **Credentials > API Keys**.

The API key can be provided directly in the provider configuration or via the `TRUENAS_API_KEY` environment variable.

## Example Usage

```terraform
terraform {
  required_providers {
    truenas = {
      source = "barodeur/truenas"
    }
  }
}

provider "truenas" {
  host     = "wss://truenas.local"
  api_key  = var.truenas_api_key
  insecure = true
}

variable "truenas_api_key" {
  type      = string
  sensitive = true
}
```

## Schema

### Required

- `host` (String) WebSocket URL of the TrueNAS server (e.g. `wss://truenas.local`). If no scheme is provided, `wss://` is assumed. Can also be set with the `TRUENAS_HOST` environment variable.
- `api_key` (String, Sensitive) API key for authentication with TrueNAS. Can also be set with the `TRUENAS_API_KEY` environment variable.

### Optional

- `insecure` (Boolean) Skip TLS certificate verification. Useful for self-signed certificates. Defaults to `false`.
