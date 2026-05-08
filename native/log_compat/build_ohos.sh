#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

: "${OHOS_SDK:?Set OHOS_SDK to the HarmonyOS SDK root, for example /home/me/ohos-sdk/linux.}"

OHOS_ARCH="${OHOS_ARCH:-arm64-v8a}"
BUILD_DIR="$SCRIPT_DIR/build/$OHOS_ARCH"
OUT_DIR="$ROOT_DIR/entry/libs/$OHOS_ARCH"

cmake -S "$SCRIPT_DIR" \
  -B "$BUILD_DIR" \
  -DCMAKE_TOOLCHAIN_FILE="$OHOS_SDK/native/build/cmake/ohos.toolchain.cmake" \
  -DOHOS_ARCH="$OHOS_ARCH" \
  -DCMAKE_BUILD_TYPE=Release

cmake --build "$BUILD_DIR" --config Release

LOG_SO="$(find "$BUILD_DIR" -name liblog.so -type f | head -n 1)"
if [[ -z "$LOG_SO" ]]; then
  echo "liblog.so was not produced under $BUILD_DIR" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
cp "$LOG_SO" "$OUT_DIR/liblog.so"

echo "Built $OUT_DIR/liblog.so"
