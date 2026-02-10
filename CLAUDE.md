# Terraform Provider for TrueNAS Scale

## Build & Test Commands

```
make build          # Build the provider binary
make install        # Build and install to ~/.terraform.d/plugins
make test           # Run unit tests
make testacc        # Run acceptance tests (requires TF_ACC=1 and a real TrueNAS instance)
make fmt            # Format Go code
make lint           # Run golangci-lint
go vet ./...        # Vet all packages
```

## Architecture

- **Module path:** `github.com/barodeur/terraform-provider-truenas`
- **Provider address:** `registry.terraform.io/barodeur/truenas`
- **Framework:** Terraform Plugin Framework (not SDKv2)
- **API transport:** JSON-RPC 2.0 over WebSocket (`gorilla/websocket`)

### Package layout

- `internal/client/` — WebSocket JSON-RPC 2.0 client. Mutex-serialized (one concurrent reader + one writer). Used by all resources/data sources.
- `internal/provider/` — Provider definition, resources, and data sources. Each resource/data source is a separate file named `<type>_resource.go` or `<type>_data_source.go`.

### Key design notes

- The `api_key.key` attribute is write-only from TrueNAS's perspective: only returned on `api_key.create`. On Read/Update, the key is preserved from prior Terraform state. After import, `key` is null.
- Provider config supports env var fallbacks: `TRUENAS_HOST` (full WebSocket URL, e.g. `wss://truenas.local`), `TRUENAS_API_KEY`.
- The `insecure` provider attribute skips TLS verification for self-signed certs.

## TrueNAS API

- **Only use the WebSocket JSON-RPC 2.0 API.** The REST API (`/api/v2.0/`) is deprecated.
- The provider targets **TrueNAS SCALE 25.10** (Goldeye).
- Some `.create` methods return **job IDs** (e.g. `group.create`, `pool.create`). Use `client.CallJob()` which calls `core.job_wait` then reads back the result. Others (e.g. `user.create`, `sharing.smb.create`, `pool.dataset.create`) return the full object directly.
- ZFS property objects in API responses have `parsed` (lowercase), `value` (uppercase), and `source` fields. Use `value` for display-consistent casing.
- `comments` on datasets is a ZFS user property stored in `user_properties.comments`, not a top-level field.
- Probe the API by sending invalid params — error messages list expected fields and valid enum values.

## Acceptance Test VM

A QEMU-based TrueNAS VM is used for acceptance tests. State lives in `/tmp/truenas-vm`.

```
scripts/truenas-vm.sh status   # Check if VM is running
scripts/truenas-vm.sh start    # Start (needs TRUENAS_ISO on first boot only)
scripts/truenas-vm.sh stop     # Stop
```

After install, `cmd/setup-truenas` bootstraps the VM (creates API key + pool). The API key is written to `/tmp/truenas-api-key`.

### Running acceptance tests against the VM

```
TRUENAS_HOST="wss://127.0.0.1:8443" \
TRUENAS_API_KEY="$(cat /tmp/truenas-api-key)" \
TRUENAS_POOL=tank \
TF_ACC=1 go test ./internal/provider/ -v -timeout 10m
```

### Stale test resources

Acceptance tests for groups and users can leave behind resources on failure. Before re-running tests, clean up stale groups (`tf-acc-test-group`, `tf-acc-test-group-smb`, `tf-acc-test-usergrp`) and users (`tfaccuser`, `tfaccuserupd`, `tfaccusergrp`) via the API.

### Pool creation

The pool `tank` must exist before running tests (the provider doesn't manage pools). If missing, create it via WebSocket API using `pool.create` with the available data disk (`vdb`).

## Adding a new resource

1. Create `internal/provider/<name>_resource.go` implementing `resource.Resource` and `resource.ResourceWithConfigure`.
2. Register it in `provider.go` `Resources()`.
3. Add an example in `examples/resources/truenas_<name>/resource.tf`.

## Git Commits

Do not include a Co-Authored-By line in commit messages.
