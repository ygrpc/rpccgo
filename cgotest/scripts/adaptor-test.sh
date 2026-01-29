#!/usr/bin/env bash
# Run adaptor tests for specified protocol
# Usage: adaptor-test.sh <protocol>
# Protocols: grpc, connect, mix, connect_suffix

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CGOTEST_DIR="$(dirname "$SCRIPT_DIR")"

PROTOCOL="${1:-}"

if [[ -z "$PROTOCOL" ]]; then
    echo "Usage: $0 <protocol>"
    echo "Protocols: grpc, connect, mix, connect_suffix"
    exit 1
fi

case "$PROTOCOL" in
    grpc|connect|mix|connect_suffix) ;;
    *)
        echo "Invalid protocol: $PROTOCOL"
        echo "Protocols: grpc, connect, mix, connect_suffix"
        exit 1
        ;;
esac

echo ">>> Testing $PROTOCOL..."
(cd "$CGOTEST_DIR/$PROTOCOL" && go test -v ./)
echo "✓ Adaptor tests passed ($PROTOCOL)"
