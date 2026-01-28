#!/usr/bin/env bash
# Install local workspace plugins (protoc-gen-rpc-cgo-adaptor, protoc-gen-rpc-cgo)
# This should be called once before build tasks

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "Installing local protoc plugins from workspace..."
cd "$ROOT_DIR"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

go install ./cmd/protoc-gen-rpc-cgo-adaptor
go install ./cmd/protoc-gen-rpc-cgo

echo "✓ Local plugins installed"
