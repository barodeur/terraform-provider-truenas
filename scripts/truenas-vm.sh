#!/usr/bin/env bash
set -euo pipefail

TRUENAS_VM_DIR="${TRUENAS_VM_DIR:-/tmp/truenas-vm}"
TRUENAS_VM_MEMORY="${TRUENAS_VM_MEMORY:-4096}"
TRUENAS_VM_CPUS="${TRUENAS_VM_CPUS:-2}"
TRUENAS_VM_PORT="${TRUENAS_VM_PORT:-8080}"
TRUENAS_VM_HTTPS_PORT="${TRUENAS_VM_HTTPS_PORT:-8443}"

TRUENAS_VM_DATA_DISK_SIZE="${TRUENAS_VM_DATA_DISK_SIZE:-8G}"
TRUENAS_CACHE_DIR="${TRUENAS_CACHE_DIR:-$HOME/.cache/truenas-vm}"

DISK="$TRUENAS_VM_DIR/disk.qcow2"
DATA_DISK="$TRUENAS_VM_DIR/data-disk.qcow2"
PIDFILE="$TRUENAS_VM_DIR/qemu.pid"
MONITOR_SOCK="$TRUENAS_VM_DIR/monitor.sock"
SERIAL_LOG="$TRUENAS_VM_DIR/serial.log"
INSTALLED_MARKER="$TRUENAS_VM_DIR/installed"

cmd_start() {
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "VM is already running (PID $(cat "$PIDFILE"))"
        exit 0
    fi

    mkdir -p "$TRUENAS_VM_DIR"

    FRESH_DISK=false
    if [ ! -f "$DISK" ]; then
        # Try restoring from cache before falling back to fresh install
        if [ -f "$TRUENAS_CACHE_DIR/disk.qcow2.zst" ] && [ -f "$TRUENAS_CACHE_DIR/data-disk.qcow2.zst" ]; then
            echo "Restoring VM from cache ($TRUENAS_CACHE_DIR)..."
            zstd -d "$TRUENAS_CACHE_DIR/disk.qcow2.zst" -o "$DISK"
            zstd -d "$TRUENAS_CACHE_DIR/data-disk.qcow2.zst" -o "$DATA_DISK"
            if [ -f "$TRUENAS_CACHE_DIR/api-key" ]; then
                cp "$TRUENAS_CACHE_DIR/api-key" /tmp/truenas-api-key
            fi
            TRUENAS_VM_LOADVM="${TRUENAS_VM_LOADVM:-ready}"
        elif [ -z "${TRUENAS_ISO:-}" ]; then
            echo "Error: TRUENAS_ISO must be set for first boot (no disk image or cache exists)" >&2
            exit 1
        else
            echo "Creating 16G boot disk image..."
            qemu-img create -f qcow2 "$DISK" 16G
            FRESH_DISK=true
            rm -f "$INSTALLED_MARKER"
        fi
    fi

    if [ ! -f "$DATA_DISK" ]; then
        echo "Creating ${TRUENAS_VM_DATA_DISK_SIZE} data disk image..."
        qemu-img create -f qcow2 "$DATA_DISK" "$TRUENAS_VM_DATA_DISK_SIZE"
    fi

    # Detect KVM availability
    ACCEL_ARGS=""
    if [ -w /dev/kvm ]; then
        ACCEL_ARGS="-machine q35,accel=kvm -cpu host"
        ACCEL_LABEL="KVM"
    else
        ACCEL_ARGS="-machine q35 -cpu max"
        ACCEL_LABEL="TCG (no KVM — will be slow)"
    fi

    # Only include ISO/cdrom on first boot (fresh disk)
    CDROM_ARGS=""
    if [ "$FRESH_DISK" = true ] && [ -n "${TRUENAS_ISO:-}" ]; then
        if [ ! -f "$TRUENAS_ISO" ]; then
            echo "Error: ISO not found at $TRUENAS_ISO" >&2
            exit 1
        fi
        CDROM_ARGS="-boot once=d -cdrom $TRUENAS_ISO"
        echo "Starting TrueNAS VM (installer boot)..."
        echo "  ISO:    $TRUENAS_ISO"
    else
        echo "Starting TrueNAS VM (disk boot)..."
    fi

    # Resume from a saved VM snapshot (memory + device state)
    LOADVM_ARGS=""
    if [ -n "${TRUENAS_VM_LOADVM:-}" ]; then
        LOADVM_ARGS="-loadvm $TRUENAS_VM_LOADVM"
        echo "  Restore: snapshot '$TRUENAS_VM_LOADVM'"
    fi

    echo "  Boot:   $DISK"
    echo "  Data:   $DATA_DISK"
    echo "  Memory: ${TRUENAS_VM_MEMORY}MB"
    echo "  CPUs:   $TRUENAS_VM_CPUS"
    echo "  Port:   $TRUENAS_VM_PORT -> 80 (HTTP/installer)"
    echo "  Port:   $TRUENAS_VM_HTTPS_PORT -> 443 (HTTPS/API)"
    echo "  Accel:  $ACCEL_LABEL"

    # shellcheck disable=SC2086
    qemu-system-x86_64 \
        $ACCEL_ARGS \
        -smp "$TRUENAS_VM_CPUS" \
        -m "$TRUENAS_VM_MEMORY" \
        $CDROM_ARGS \
        $LOADVM_ARGS \
        -drive "file=$DISK,format=qcow2,if=none,id=boot0" \
        -device "virtio-blk-pci,drive=boot0,serial=TRUENAS-BOOT-001" \
        -drive "file=$DATA_DISK,format=qcow2,if=none,id=data0" \
        -device "virtio-blk-pci,drive=data0,serial=TRUENAS-DATA-001" \
        -nic "user,hostfwd=tcp::${TRUENAS_VM_PORT}-:80,hostfwd=tcp::${TRUENAS_VM_HTTPS_PORT}-:443" \
        -display none \
        -daemonize \
        -pidfile "$PIDFILE" \
        -serial "file:$SERIAL_LOG" \
        -monitor "unix:$MONITOR_SOCK,server,nowait"

    echo "VM started (PID $(cat "$PIDFILE"))"
    echo "Serial log: $SERIAL_LOG"
}

cmd_stop() {
    if [ ! -f "$PIDFILE" ]; then
        echo "No PID file found, VM may not be running"
        exit 0
    fi

    PID=$(cat "$PIDFILE")
    if kill -0 "$PID" 2>/dev/null; then
        echo "Stopping VM (PID $PID)..."

        # Send ACPI powerdown via monitor socket for graceful guest shutdown
        if [ -S "$MONITOR_SOCK" ]; then
            echo "Sending ACPI powerdown..."
            echo "system_powerdown" | socat - UNIX-CONNECT:"$MONITOR_SOCK" || true
        else
            echo "No monitor socket, sending SIGTERM..."
            kill "$PID"
        fi

        # Wait for process to exit
        for i in $(seq 1 120); do
            if ! kill -0 "$PID" 2>/dev/null; then
                break
            fi
            sleep 1
        done
        if kill -0 "$PID" 2>/dev/null; then
            echo "Force killing VM..."
            kill -9 "$PID" 2>/dev/null || true
        fi
        echo "VM stopped"
    else
        echo "VM is not running (stale PID file)"
    fi
    rm -f "$PIDFILE"
}

cmd_wait_api() {
    local timeout="${TRUENAS_VM_WAIT_TIMEOUT:-300}"
    local url="https://127.0.0.1:${TRUENAS_VM_HTTPS_PORT}"

    echo "Waiting for TrueNAS API at ${url}..."
    local deadline=$((SECONDS + timeout))
    while [ "$SECONDS" -lt "$deadline" ]; do
        if curl -sk -o /dev/null "$url" 2>/dev/null; then
            echo "TrueNAS API is ready"
            return 0
        fi
        sleep 5
    done
    echo "Timed out waiting for TrueNAS API after ${timeout}s"
    return 1
}

cmd_clean() {
    # Stop the VM first if it's running
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "VM is running, stopping it first..."
        cmd_stop
    fi

    echo "Removing VM state directory: $TRUENAS_VM_DIR"
    rm -rf "$TRUENAS_VM_DIR"

    if [ -f /tmp/truenas-api-key ]; then
        echo "Removing /tmp/truenas-api-key"
        rm -f /tmp/truenas-api-key
    fi

    echo "Clean complete"
}

cmd_status() {
    if [ ! -f "$PIDFILE" ]; then
        echo "VM is not running (no PID file)"
        exit 1
    fi

    PID=$(cat "$PIDFILE")
    if kill -0 "$PID" 2>/dev/null; then
        echo "VM is running (PID $PID)"
        exit 0
    else
        echo "VM is not running (stale PID file)"
        rm -f "$PIDFILE"
        exit 1
    fi
}

cmd_snapshot() {
    if [ ! -f "$PIDFILE" ] || ! kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "Error: VM must be running to snapshot" >&2
        exit 1
    fi

    if [ ! -S "$MONITOR_SOCK" ]; then
        echo "Error: Monitor socket not found at $MONITOR_SOCK" >&2
        exit 1
    fi

    echo "Creating VM snapshot..."
    local snapshot_log="$TRUENAS_VM_DIR/snapshot.log"
    (
        printf '%s\n' \
            "savevm ready" \
            "snapshot_blkdev boot0 $TRUENAS_VM_DIR/overlay-boot.qcow2 qcow2" \
            "snapshot_blkdev data0 $TRUENAS_VM_DIR/overlay-data.qcow2 qcow2" \
            "info status"
        sleep 300
    ) | timeout 120 socat - UNIX-CONNECT:"$MONITOR_SOCK" > "$snapshot_log" 2>&1 &
    local socat_pid=$!

    echo "Waiting for savevm + snapshot_blkdev to complete..."
    while ! grep -q "VM status" "$snapshot_log" 2>/dev/null; do
        sleep 2
    done
    kill $socat_pid 2>/dev/null || true
    echo "VM snapshot complete"

    echo "Compressing disk images to $TRUENAS_CACHE_DIR..."
    mkdir -p "$TRUENAS_CACHE_DIR"
    zstd -3 -T0 --force "$DISK" -o "$TRUENAS_CACHE_DIR/disk.qcow2.zst"
    zstd -3 -T0 --force "$DATA_DISK" -o "$TRUENAS_CACHE_DIR/data-disk.qcow2.zst"
    if [ -f /tmp/truenas-api-key ]; then
        cp /tmp/truenas-api-key "$TRUENAS_CACHE_DIR/api-key"
    fi

    echo "Cache saved to $TRUENAS_CACHE_DIR"
    ls -lh "$TRUENAS_CACHE_DIR/"
}

case "${1:-}" in
    start)    cmd_start ;;
    stop)     cmd_stop ;;
    snapshot) cmd_snapshot ;;
    clean)    cmd_clean ;;
    status)   cmd_status ;;
    wait-api) cmd_wait_api ;;
    *)
        echo "Usage: $0 {start|stop|snapshot|clean|status|wait-api}" >&2
        echo "" >&2
        echo "Environment variables:" >&2
        echo "  TRUENAS_ISO              Path to TrueNAS ISO (required for first boot)" >&2
        echo "  TRUENAS_VM_DIR           VM working directory (default: /tmp/truenas-vm)" >&2
        echo "  TRUENAS_VM_MEMORY        VM memory in MB (default: 4096)" >&2
        echo "  TRUENAS_VM_CPUS          VM CPUs (default: 2)" >&2
        echo "  TRUENAS_VM_DATA_DISK_SIZE Size of the data disk (default: 8G)" >&2
        echo "  TRUENAS_VM_PORT          Host port forwarded to VM port 80 (default: 8080)" >&2
        echo "  TRUENAS_VM_HTTPS_PORT    Host port forwarded to VM port 443 (default: 8443)" >&2
        echo "  TRUENAS_VM_WAIT_TIMEOUT  Timeout in seconds for wait-api (default: 300)" >&2
        echo "  TRUENAS_CACHE_DIR        Persistent cache directory (default: ~/.cache/truenas-vm)" >&2
        exit 1
        ;;
esac
