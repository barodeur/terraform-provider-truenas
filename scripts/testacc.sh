#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

TRUENAS_VERSION="${TRUENAS_VERSION:-25.10.1}"
TRUENAS_VM_HTTPS_PORT="${TRUENAS_VM_HTTPS_PORT:-8443}"
VM_MODE="snapshot"
GO_TEST_ARGS=()

usage() {
    cat >&2 <<EOF
Usage: $(basename "$0") [OPTIONS] [GO TEST FLAGS...]

Run TrueNAS acceptance tests with automatic VM lifecycle management.

Options:
  --vm=MODE         VM lifecycle mode (default: snapshot)
                      snapshot   - Restore cached snapshot, wait for API
                      running    - Reuse running VM as-is
                      reinstall  - Boot from cached ISO, re-run setup, snapshot
                      full       - Purge caches, download ISO, full rebuild
  --version=VER     TrueNAS version to use (default: $TRUENAS_VERSION)
  -h, --help        Show this help

Extra arguments are passed to 'go test'. For example:
  $(basename "$0") --vm=running -run TestAccGroup
  $(basename "$0") -count=1 -run TestAccUser
EOF
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --vm=*)
            VM_MODE="${1#--vm=}"
            shift
            ;;
        --version=*)
            TRUENAS_VERSION="${1#--version=}"
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            GO_TEST_ARGS+=("$1")
            shift
            ;;
    esac
done

case "$VM_MODE" in
    snapshot|running|reinstall|full) ;;
    *)
        echo "Error: unknown VM mode '$VM_MODE'" >&2
        echo "Valid modes: snapshot, running, reinstall, full" >&2
        exit 1
        ;;
esac

ISO_URL="https://download.truenas.com/TrueNAS-SCALE-Goldeye/${TRUENAS_VERSION}/TrueNAS-SCALE-${TRUENAS_VERSION}.iso"
ISO_PATH="${TRUENAS_ISO:-$HOME/.cache/truenas-vm/TrueNAS-SCALE-${TRUENAS_VERSION}.iso}"

vm() {
    "$SCRIPT_DIR/truenas-vm.sh" "$@"
}

# Stop the VM on exit (unless running mode)
cleanup() {
    if [[ "$VM_MODE" != "running" ]]; then
        echo ""
        echo "=== Stopping VM ==="
        vm stop || true
    fi
}
trap cleanup EXIT

# --- VM lifecycle ---

case "$VM_MODE" in
    running)
        echo "=== VM mode: running (reusing existing VM) ==="
        if ! vm status >/dev/null 2>&1; then
            echo "Error: VM is not running. Start it first or use a different --vm mode." >&2
            exit 1
        fi
        ;;

    snapshot)
        echo "=== VM mode: snapshot (restoring cached snapshot) ==="
        # Stop + clean, then start (which restores from cache)
        vm stop 2>/dev/null || true
        vm clean 2>/dev/null || true
        vm start
        vm wait-api
        ;;

    reinstall)
        echo "=== VM mode: reinstall (fresh install from cached ISO) ==="
        vm stop 2>/dev/null || true
        vm clean 2>/dev/null || true

        # Ensure ISO exists
        if [[ ! -f "$ISO_PATH" ]]; then
            echo "Error: ISO not found at $ISO_PATH" >&2
            echo "Run with --vm=full to download it, or set TRUENAS_ISO." >&2
            exit 1
        fi

        # Boot from ISO, run setup, snapshot
        TRUENAS_ISO="$ISO_PATH" vm start
        go build -o "$PROJECT_DIR/setup-truenas" "$PROJECT_DIR/cmd/setup-truenas"
        "$PROJECT_DIR/setup-truenas" \
            -host 127.0.0.1 \
            -port "${TRUENAS_VM_PORT:-8080}" \
            -https-port "$TRUENAS_VM_HTTPS_PORT" \
            -output-file /tmp/truenas-api-key
        vm snapshot
        vm wait-api
        ;;

    full)
        echo "=== VM mode: full (purge caches, download ISO, full rebuild) ==="
        vm stop 2>/dev/null || true
        vm clean 2>/dev/null || true

        # Purge cached disk images
        TRUENAS_CACHE_DIR="${TRUENAS_CACHE_DIR:-$HOME/.cache/truenas-vm}"
        echo "Purging cache directory: $TRUENAS_CACHE_DIR"
        rm -rf "$TRUENAS_CACHE_DIR"

        # Download ISO if needed
        mkdir -p "$(dirname "$ISO_PATH")"
        if [[ ! -f "$ISO_PATH" ]]; then
            echo "Downloading TrueNAS SCALE ${TRUENAS_VERSION}..."
            echo "  URL: $ISO_URL"
            echo "  Destination: $ISO_PATH"
            curl -L -o "$ISO_PATH" "$ISO_URL"
        else
            echo "ISO already exists: $ISO_PATH"
        fi

        # Boot from ISO, run setup, snapshot
        TRUENAS_ISO="$ISO_PATH" vm start
        go build -o "$PROJECT_DIR/setup-truenas" "$PROJECT_DIR/cmd/setup-truenas"
        "$PROJECT_DIR/setup-truenas" \
            -host 127.0.0.1 \
            -port "${TRUENAS_VM_PORT:-8080}" \
            -https-port "$TRUENAS_VM_HTTPS_PORT" \
            -output-file /tmp/truenas-api-key
        vm snapshot
        vm wait-api
        ;;
esac

# --- Run tests ---

export TRUENAS_HOST="wss://127.0.0.1:${TRUENAS_VM_HTTPS_PORT}"
export TRUENAS_API_KEY="$(cat /tmp/truenas-api-key)"
export TRUENAS_POOL="${TRUENAS_POOL:-tank}"

echo ""
echo "=== Running acceptance tests ==="

TF_ACC=1 go test "$PROJECT_DIR/internal/provider/" \
    -v -timeout 10m \
    ${GO_TEST_ARGS[@]+"${GO_TEST_ARGS[@]}"}
