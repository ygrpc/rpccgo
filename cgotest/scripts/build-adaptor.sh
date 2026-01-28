#!/usr/bin/env bash
# Build adaptor code for specified protocol
# Usage: build-adaptor.sh <protocol>
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

# Check protoc
if ! command -v protoc >/dev/null 2>&1; then
    echo "protoc is not installed. Please install Protocol Buffers compiler."
    exit 1
fi

# Protocol-specific config
case "$PROTOCOL" in
    grpc)
        PROTOCOL_OPT="grpc"
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/grpc;cgotest_grpc"
        ;;
    connect)
        PROTOCOL_OPT="connectrpc"
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect;cgotest_connect"
        ;;
    mix)
        PROTOCOL_OPT="grpc|connectrpc"
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/mix;cgotest_mix"
        ;;
    connect_suffix)
        PROTOCOL_OPT="connectrpc"
        GO_PKG="Munary.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix,Mstream.proto=github.com/ygrpc/rpccgo/cgotest/connect_suffix;cgotest_connect_suffix"
        ;;
esac

mkdir -p "./$PROTOCOL"
# Clean generated files but keep hand-written tests
find "./$PROTOCOL" -mindepth 1 -maxdepth 1 -type f ! -name 'adaptor_test.go' -delete
find "./$PROTOCOL" -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' '+' 2>/dev/null || true

# Generate code based on protocol
case "$PROTOCOL" in
    grpc)
        protoc -Iproto -I../proto \
            --go_out=./grpc --go_opt=paths=source_relative,"${GO_PKG}" \
            --go-grpc_out=./grpc --go-grpc_opt=paths=source_relative,"${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto

        protoc -Iproto -I../proto \
            --rpc-cgo-adaptor_out=./grpc \
            --rpc-cgo-adaptor_opt=paths=source_relative,protocol=grpc,"${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto
        ;;
    connect)
        protoc -Iproto -I../proto \
            --go_out=./connect --go_opt=paths=source_relative,"${GO_PKG}" \
            --connect-go_out=./connect --connect-go_opt=package_suffix="",paths=source_relative,simple=true,"${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto

        protoc -Iproto -I../proto \
            --rpc-cgo-adaptor_out=./connect \
            --rpc-cgo-adaptor_opt=paths=source_relative,protocol=connectrpc,"${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto
        ;;
    mix)
        protoc -Iproto -I../proto \
            --go_out=./mix --go_opt=paths=source_relative,"${GO_PKG}" \
            --go-grpc_out=./mix --go-grpc_opt=paths=source_relative,"${GO_PKG}" \
            --connect-go_out=./mix --connect-go_opt=paths=source_relative,simple=true,"${GO_PKG}" \
            --rpc-cgo-adaptor_out=./mix \
            --rpc-cgo-adaptor_opt=paths=source_relative,"protocol=${PROTOCOL_OPT}","${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto
        ;;
    connect_suffix)
        protoc -Iproto -I../proto \
            --go_out=./connect_suffix --go_opt=paths=source_relative,"${GO_PKG}" \
            --connect-go_out=./connect_suffix --connect-go_opt=paths=source_relative,simple=true,"${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto

        protoc -Iproto -I../proto \
            --rpc-cgo-adaptor_out=./connect_suffix \
            --rpc-cgo-adaptor_opt=paths=source_relative,protocol=connectrpc,"${GO_PKG}" \
            ./proto/unary.proto ./proto/stream.proto
        ;;
esac

echo "✓ Adaptor code generated for $PROTOCOL"
