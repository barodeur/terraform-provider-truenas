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

A QEMU-based TrueNAS VM is used for acceptance tests. Use `scripts/testacc.sh` as the single entry point:

```
scripts/testacc.sh                          # Default: restore snapshot, test
scripts/testacc.sh --vm=running             # Reuse running VM as-is
scripts/testacc.sh --vm=reinstall           # Boot from cached ISO, re-run setup
scripts/testacc.sh --vm=full                # Purge caches, download ISO, full rebuild
scripts/testacc.sh --vm=running -run TestAccGroup  # Pass extra go test flags
scripts/testacc.sh --version=25.10.2 --vm=full     # Use a different TrueNAS version
```

**VM modes:**

| Mode | What it does | When to use |
|------|-------------|-------------|
| `snapshot` (default) | Stop + clean VM, restore cached snapshot, wait for API | Normal development iteration |
| `running` | Skip all VM lifecycle, test against running VM | Quick re-runs when VM is already good |
| `reinstall` | Clean VM, boot from cached ISO, run setup-truenas, snapshot | Setup code or provider changed |
| `full` | Purge all caches, download ISO, boot, setup, snapshot | New TrueNAS version or start from scratch |

The script exports the required env vars and stops the VM on exit (except in `running` mode). Snapshot restoration gives a clean state, so no sweepers are needed.

### Low-level VM management

For manual VM control, use `scripts/truenas-vm.sh` directly:

```
scripts/truenas-vm.sh status   # Check if VM is running
scripts/truenas-vm.sh start    # Start (restores from cache, or needs TRUENAS_ISO)
scripts/truenas-vm.sh stop     # Stop
scripts/truenas-vm.sh clean    # Stop + remove /tmp/truenas-vm
scripts/truenas-vm.sh snapshot # Save VM state to ~/.cache/truenas-vm
```

`cmd/setup-truenas` bootstraps a fresh VM (installer, API key, pool). The API key is written to `/tmp/truenas-api-key`.

### Pool creation

The pool `tank` must exist before running tests (the provider doesn't manage pools). `scripts/testacc.sh` handles this automatically in `reinstall` and `full` modes via `setup-truenas`.

## Adding a new resource

1. Create `internal/provider/<name>_resource.go` implementing `resource.Resource` and `resource.ResourceWithConfigure`.
2. Register it in `provider.go` `Resources()`.
3. Add an example in `examples/resources/truenas_<name>/resource.tf`.

## Git Commits

Do not include a Co-Authored-By line in commit messages.
