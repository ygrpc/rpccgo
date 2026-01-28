#!/usr/bin/env bash
# Build C ABI export for specified protocol
# Usage: build-cgo.sh <protocol>
# Protocols: grpc, connect, mix, connect_suffix
#
# NOTE: This script assumes protoc plugins are already installed.
# Use 'task install-local-plugins' or 'scripts/install-local-plugins.sh' first.

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

cd "$CGOTEST_DIR"

# Set CGO dir and package mapping based on protocol
CGO_DIR="./cgo_$PROTOCOL"
case "$PROTOCOL" in
    grpc)
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc"
        ;;
    connect)
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect"
        ;;
    mix)
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix"
        ;;
    connect_suffix)
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix"
        ;;
esac

mkdir -p "$CGO_DIR"
find "$CGO_DIR" -mindepth 1 -maxdepth 1 -type f ! -name 'registry.go' -delete
find "$CGO_DIR" -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+' 2>/dev/null || true

protoc -Iproto -I../proto \
    --rpc-cgo_out="$CGO_DIR" \
    --rpc-cgo_opt=paths=source_relative,"${GO_PKG}" \
    ./proto/unary.proto ./proto/stream.proto

echo "Building c-shared library ($PROTOCOL)..."
go build -buildmode=c-shared -o ./c_tests/libygrpc.so "$CGO_DIR"
cp "$CGO_DIR/ygrpc_cgo_common.h" ./c_tests/

echo "✓ C ABI export built ($PROTOCOL)"
