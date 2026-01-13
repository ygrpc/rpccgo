#!/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/"

echo "=== Building test packages ==="

echo ""
echo ">>> Building all test packages..."
./build-connect.sh
./build-grpc.sh
./build-connect-suffix.sh
./build-mix.sh

echo ""
echo "=== Running all tests ==="

echo ""
echo ">>> Testing ConnectRPC..."
(cd connect && go test -v ./)

echo ""
echo ">>> Testing gRPC..."
(cd grpc && go test -v ./)

echo ""
echo ">>> Testing mix (grpc+connectrpc)..."
(cd mix && go test -v ./)

echo ""
echo ">>> Testing connect_suffix (connectrpc)..."
(cd connect_suffix && go test -v ./)

echo ""
echo "=== All tests completed ==="
