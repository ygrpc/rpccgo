#!/usr/bin/env bash
# Regenerate nanopb C code from proto files
#
# This script bootstraps a local Python venv (cgotest/.venv) and installs
# dependencies from c_tests/nanopb/requirements.txt automatically.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CGOTEST_DIR="$(dirname "$SCRIPT_DIR")"

cd "$CGOTEST_DIR"

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is not installed. Please install Python 3." >&2
  exit 1
fi

VENV_DIR="$CGOTEST_DIR/.venv"
VENV_PY="$VENV_DIR/bin/python3"
REQ_FILE="$CGOTEST_DIR/c_tests/nanopb/requirements.txt"
STAMP_FILE="$VENV_DIR/.nanopb_requirements.sha256"

if [[ ! -f "$VENV_PY" ]]; then
  python3 -m venv "$VENV_DIR"
fi

"$VENV_PY" -m pip install --quiet --upgrade pip

req_sha=""
if command -v sha256sum >/dev/null 2>&1; then
  req_sha="$(sha256sum "$REQ_FILE" | awk '{print $1}')"
fi

need_install=1
if [[ -n "$req_sha" ]] && [[ -f "$STAMP_FILE" ]] && [[ "$(cat "$STAMP_FILE")" == "$req_sha" ]]; then
  need_install=0
fi

if [[ "$need_install" -eq 1 ]]; then
  "$VENV_PY" -m pip install --quiet -r "$REQ_FILE"
  if [[ -n "$req_sha" ]]; then
    echo "$req_sha" > "$STAMP_FILE"
  fi
fi

"$VENV_PY" c_tests/nanopb/generator/nanopb_generator.py \
  -I proto -I ../proto -D c_tests/pb \
  proto/unary.proto proto/stream.proto

echo "✓ nanopb C code regenerated"
