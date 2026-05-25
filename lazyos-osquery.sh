#!/usr/bin/env bash
set -e

echo "======================================================"
echo " LazyOS (Ephemeral Evaluation Mode)"
echo "======================================================"
echo "This script will download everything into memory/tmp."
echo "When you close the app, NO binaries, NO daemons, and"
echo "NO logs will be left on your machine."
echo "======================================================"

DOWNLOAD_DIR=$(mktemp -d /tmp/lazyos_download_XXXXXX)

trap 'rm -rf "$DOWNLOAD_DIR"' EXIT

echo "[*] Fetching osquery (5.11.0)..."
curl -L -s -f https://pkg.osquery.io/linux/osquery-5.11.0_1.linux_x86_64.tar.gz | tar -xz -C "$DOWNLOAD_DIR"
OSQUERYD=$(find "$DOWNLOAD_DIR" -type f -name "osqueryd" | head -n 1)
chmod +x "$OSQUERYD"

echo "[*] Fetching LazyOS binary..."
env GO111MODULE=on GOPROXY=direct GOBIN="$DOWNLOAD_DIR" go install github.com/ajitm722/LazyOS/cmd/lazyos@latest

echo "[*] Fetching Daemon Wrapper..."
curl -L -s -f -o "$DOWNLOAD_DIR/osquery_daemon_wrapper.sh" \
    https://raw.githubusercontent.com/ajitm722/LazyOS/main/scripts/osquery_daemon_wrapper.sh
chmod +x "$DOWNLOAD_DIR/osquery_daemon_wrapper.sh"

echo "[*] Ephemeral Sandbox Active: Zero bloat. All files and background processes will vanish on exit."
"$DOWNLOAD_DIR/osquery_daemon_wrapper.sh" "$OSQUERYD" "$DOWNLOAD_DIR/lazyos" --osquery-socket=LAZYOS_TEST_SOCKET || true
echo "[*] Sandbox completely wiped. Goodbye!"
