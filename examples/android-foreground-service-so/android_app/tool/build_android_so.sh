#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LIB_NAME="librpccgo_android_foreground_service.so"
JNI_LIBS_DIR="$ROOT_DIR/android_app/app/src/main/jniLibs"

SDK_ROOT="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Android/Sdk}}"
NDK_ROOT="${ANDROID_NDK_HOME:-}"

if [[ -z "${NDK_ROOT}" ]]; then
  NDK_ROOT="$(find "$SDK_ROOT/ndk" -mindepth 1 -maxdepth 1 -type d | sort -V | tail -n 1)"
fi

if [[ -z "${NDK_ROOT}" || ! -d "${NDK_ROOT}" ]]; then
  echo "ANDROID_NDK_HOME is not set and no NDK was found under $SDK_ROOT/ndk" >&2
  exit 1
fi

case "$(uname -s)" in
  Linux) host_tag="linux-x86_64" ;;
  Darwin)
    case "$(uname -m)" in
      arm64) host_tag="darwin-arm64" ;;
      x86_64) host_tag="darwin-x86_64" ;;
      *) echo "Unsupported macOS host architecture: $(uname -m)" >&2; exit 1 ;;
    esac
    ;;
  *) echo "Unsupported host OS: $(uname -s)" >&2; exit 1 ;;
esac

TOOLCHAIN="$NDK_ROOT/toolchains/llvm/prebuilt/$host_tag/bin"
if [[ ! -d "${TOOLCHAIN}" ]]; then
  echo "Android NDK toolchain not found: $TOOLCHAIN" >&2
  exit 1
fi

build_one() {
  local goarch="$1"
  local abi="$2"
  local cc_name="$3"
  local goarm="${4:-}"
  local out_dir="$JNI_LIBS_DIR/$abi"

  mkdir -p "$out_dir"
  rm -f "$out_dir/$LIB_NAME" "$out_dir/${LIB_NAME%.so}.h"

  env \
    CGO_ENABLED=1 \
    GOOS=android \
    GOARCH="$goarch" \
    GOARM="$goarm" \
    CC="$TOOLCHAIN/$cc_name" \
    GOFLAGS="-mod=mod" \
    go build -buildmode=c-shared -o "$out_dir/$LIB_NAME" ./cmd/rpc
}

cd "$ROOT_DIR"
build_one arm64 arm64-v8a aarch64-linux-android21-clang
build_one amd64 x86_64 x86_64-linux-android21-clang
build_one arm armeabi-v7a armv7a-linux-androideabi21-clang 7

echo "Built Android shared libraries into $JNI_LIBS_DIR"
