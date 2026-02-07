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
- Provider config supports env var fallbacks: `TRUENAS_HOST`, `TRUENAS_API_KEY`, `TRUENAS_SCHEME`.
- The `insecure` provider attribute skips TLS verification for self-signed certs.

## Adding a new resource

1. Create `internal/provider/<name>_resource.go` implementing `resource.Resource` and `resource.ResourceWithConfigure`.
2. Register it in `provider.go` `Resources()`.
3. Add an example in `examples/resources/truenas_<name>/resource.tf`.

## Git Commits

Do not include a Co-Authored-By line in commit messages.
