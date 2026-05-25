#!/usr/bin/env bash
set -e

OSQUERY_BIN="$1"
shift
CMD=("$@")

if [ ! -x "$OSQUERY_BIN" ]; then
    echo "Error: osqueryd not found or not executable at $OSQUERY_BIN"
    exit 1
fi

WORKSPACE=$(mktemp -d /tmp/lazyos_sandbox_XXXXXX)
SOCKET="$WORKSPACE/osquery.em"

trap 'echo "Cleaning up sandbox..."; \
      PID=$(cat "$WORKSPACE/osquery.pid" 2>/dev/null); \
      if [ -n "$PID" ] && kill -0 $PID 2>/dev/null; then kill $PID 2>/dev/null || true; fi; \
      rm -rf "$WORKSPACE"' EXIT

"$OSQUERY_BIN" --ephemeral --disable_database --disable_events \
    --logger_path="$WORKSPACE" \
    --extensions_socket="$SOCKET" > "$WORKSPACE/daemon.log" 2>&1 &

PID=$!
echo $PID > "$WORKSPACE/osquery.pid"

while [ ! -S "$SOCKET" ]; do
    if ! kill -0 $PID 2>/dev/null; then
        echo "Error: osqueryd crashed during boot. Logs:"
        cat "$WORKSPACE/daemon.log"
        exit 1
    fi
    sleep 0.2
done

export LAZYOS_TEST_SOCKET="$SOCKET"

ARGS=("$@")
for i in "${!ARGS[@]}"; do
    ARGS[$i]="${ARGS[$i]/LAZYOS_TEST_SOCKET/$SOCKET}"
done

"${ARGS[@]}"
