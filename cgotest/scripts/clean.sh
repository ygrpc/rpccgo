#!/usr/bin/env bash
# Clean build artifacts
# Keeps hand-written tests (adaptor_test.go, registry.go)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CGOTEST_DIR="$(dirname "$SCRIPT_DIR")"

cd "$CGOTEST_DIR"

echo "Cleaning build artifacts..."

for protocol in grpc connect mix connect_suffix; do
    echo "  Cleaning $protocol..."
    find "./$protocol" -mindepth 1 -maxdepth 1 -type f ! -name 'adaptor_test.go' -delete 2>/dev/null || true
    find "./$protocol" -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' + 2>/dev/null || true
    find "./cgo_$protocol" -mindepth 1 -maxdepth 1 -type f ! -name 'registry.go' -delete 2>/dev/null || true
    find "./cgo_$protocol" -mindepth 1 -maxdepth 1 -type d -exec rm -rf '{}' + 2>/dev/null || true
done

echo "  Cleaning c_tests artifacts..."
rm -f ./c_tests/libygrpc.so ./c_tests/libygrpc.h ./c_tests/ygrpc_cgo_common.h
rm -f ./c_tests/unary_test ./c_tests/client_stream_test ./c_tests/server_stream_test ./c_tests/bidi_stream_test

echo "✓ Clean completed"
