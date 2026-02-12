# Terraform Provider for TrueNAS Scale

A Terraform provider for managing [TrueNAS Scale](https://www.truenas.com/truenas-scale/) resources via its WebSocket JSON-RPC API.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/dl/) >= 1.24 (to build the provider)
- A TrueNAS Scale instance with an API key

## Usage

```hcl
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
  insecure = true  # skip TLS verification for self-signed certs
}

variable "truenas_api_key" {
  type      = string
  sensitive = true
}
```

### Provider arguments

| Argument   | Description | Required | Default |
|------------|-------------|----------|---------|
| `host`     | WebSocket URL of the TrueNAS server (e.g. `wss://truenas.local`). If no scheme is provided, `wss://` is assumed. Also settable via `TRUENAS_HOST`. | Yes | — |
| `api_key`  | API key for authentication. Also settable via `TRUENAS_API_KEY`. | Yes | — |
| `insecure` | Skip TLS certificate verification. | No | `false` |

## Resources

### `truenas_api_key`

Manages a TrueNAS API key.

```hcl
resource "truenas_api_key" "example" {
  name = "terraform-managed-key"
}

output "api_key_value" {
  value     = truenas_api_key.example.key
  sensitive = true
}
```

#### Arguments

| Argument     | Description | Required |
|--------------|-------------|----------|
| `name`       | The name of the API key. | Yes |
| `username`   | The user to create the key for. Defaults to the authenticated user. | No |
| `expires_at` | Expiration date in ISO 8601 format. If omitted, the key does not expire. | No |

#### Attributes

| Attribute    | Description |
|--------------|-------------|
| `id`         | Numeric ID of the API key. |
| `key`        | The API key secret. Only available after creation; cannot be re-read. |
| `created_at` | Creation timestamp. |
| `revoked`    | Whether the key has been revoked. |

> **Note:** The `key` attribute is only returned when the API key is first created. On subsequent reads, the value is preserved from Terraform state. After `terraform import`, `key` will be null.

## Data Sources

### `truenas_api_key`

Looks up an existing API key by name or ID.

```hcl
data "truenas_api_key" "example" {
  name = "terraform-managed-key"
}
```

#### Arguments

At least one of `id` or `name` must be provided.

#### Attributes

| Attribute    | Description |
|--------------|-------------|
| `id`         | Numeric ID of the API key. |
| `name`       | Name of the API key. |
| `username`   | Username associated with the key. |
| `expires_at` | Expiration date, if set. |
| `created_at` | Creation timestamp. |
| `revoked`    | Whether the key has been revoked. |

## Development

```sh
make build     # build the provider binary
make install   # install to ~/.terraform.d/plugins
make test      # run unit tests
make testacc   # run acceptance tests (requires TRUENAS_HOST, TRUENAS_API_KEY, TF_ACC=1)
make fmt       # format Go code
make lint      # run golangci-lint
```

### Running acceptance tests

Acceptance tests run against a QEMU-based TrueNAS VM. Use `scripts/testacc.sh` as the single entry point:

```sh
scripts/testacc.sh                          # Restore cached snapshot, run tests
scripts/testacc.sh --vm=running             # Reuse a running VM as-is
scripts/testacc.sh --vm=reinstall           # Boot from cached ISO, re-run setup
scripts/testacc.sh --vm=full                # Download ISO, full rebuild from scratch
scripts/testacc.sh --vm=running -run TestAccGroup  # Pass extra go test flags
```

The first run requires `--vm=full` (or `--vm=reinstall` with `TRUENAS_ISO` set) to build the VM. Subsequent runs use `snapshot` mode (the default), which restores a cached VM image in seconds.

Requires QEMU, zstd, and socat (`apt install qemu-system-x86 qemu-utils zstd socat`).
