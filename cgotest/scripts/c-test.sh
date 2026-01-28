#!/usr/bin/env bash
# Run C end-to-end tests for specified protocol
# Usage: c-test.sh <protocol>
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

cd "$CGOTEST_DIR"

cc_bin="${CC:-cc}"
cflags=(-O2 -std=c11 -Wall -Wextra -D_POSIX_C_SOURCE=200809L -I./c_tests)
ldflags=(-L./c_tests -lygrpc -Wl,-rpath,'$ORIGIN')

echo "Building C tests ($PROTOCOL)..."
"${cc_bin}" "${cflags[@]}" ./c_tests/unary_test.c -o ./c_tests/unary_test "${ldflags[@]}"
"${cc_bin}" "${cflags[@]}" ./c_tests/client_stream_test.c -o ./c_tests/client_stream_test "${ldflags[@]}"
"${cc_bin}" "${cflags[@]}" ./c_tests/server_stream_test.c -o ./c_tests/server_stream_test "${ldflags[@]}"
"${cc_bin}" "${cflags[@]}" ./c_tests/bidi_stream_test.c -o ./c_tests/bidi_stream_test "${ldflags[@]}"

echo "Running C tests ($PROTOCOL)..."
(cd ./c_tests && ./unary_test && ./client_stream_test && ./server_stream_test && ./bidi_stream_test)

echo "✓ C tests passed ($PROTOCOL)"
