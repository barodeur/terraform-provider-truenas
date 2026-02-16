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

- `truenas_api_key` — API keys
- `truenas_cronjob` — Cron jobs
- `truenas_group` — Groups
- `truenas_nfs_share` — NFS shares
- `truenas_pool_dataset` — ZFS datasets
- `truenas_smb_share` — SMB shares
- `truenas_user` — Users

## Data Sources

- `truenas_api_key` — Look up an API key
- `truenas_cronjob` — Look up a cron job
- `truenas_pool` — Look up a storage pool

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
