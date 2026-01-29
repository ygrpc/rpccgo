#!/usr/bin/env bash
# Run all tests
# Usage: test.sh

set -euo pipefail

cd "$(dirname "$0")"

if [ ! command -v task &> /dev/null ]; then
    echo "task command not found. try installing it..."
     go install github.com/go-task/task/v3/cmd/task@latest
fi

task test
