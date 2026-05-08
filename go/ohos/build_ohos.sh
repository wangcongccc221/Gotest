#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

: "${OHOS_SDK:?Set OHOS_SDK to the HarmonyOS SDK root, for example /home/me/ohos-sdk/linux.}"

OHOS_ARCH="${OHOS_ARCH:-arm64-v8a}"

case "$OHOS_ARCH" in
  arm64-v8a)
    export GOARCH=arm64
    TARGET="aarch64-linux-ohos"
    SYSROOT_LIB_DIR="aarch64-linux-ohos"
    ;;
  *)
    echo "Unsupported OHOS_ARCH=$OHOS_ARCH. This POC currently supports arm64-v8a." >&2
    exit 1
    ;;
esac

LOG_LIB="$ROOT_DIR/entry/libs/$OHOS_ARCH/liblog.so"
if [[ ! -f "$LOG_LIB" ]]; then
  "$ROOT_DIR/native/log_compat/build_ohos.sh"
fi

export GOOS=android
export CGO_ENABLED=1
export CGO_CFLAGS_ALLOW=".*"
export CGO_LDFLAGS_ALLOW=".*"

LLVMCONFIG="$OHOS_SDK/native/llvm/bin/llvm-config"
export CGO_CFLAGS="-g -O2 $("$LLVMCONFIG" --cflags) --target=$TARGET --sysroot=$OHOS_SDK/native/sysroot -I$ROOT_DIR/native/log_compat/include"
export CGO_LDFLAGS="--target=$TARGET --sysroot=$OHOS_SDK/native/sysroot -fuse-ld=lld -L$ROOT_DIR/entry/libs/$OHOS_ARCH -L$OHOS_SDK/native/sysroot/usr/lib/$SYSROOT_LIB_DIR"

export CC="$OHOS_SDK/native/llvm/bin/clang"
export CXX="$OHOS_SDK/native/llvm/bin/clang++"
export AR="$OHOS_SDK/native/llvm/bin/llvm-ar"
export LD="$OHOS_SDK/native/llvm/bin/lld"

mkdir -p "$ROOT_DIR/entry/libs/$OHOS_ARCH"

pushd "$SCRIPT_DIR" >/dev/null
go build -tags netgo -buildmode=c-shared -trimpath -ldflags="-s -w" -o "$ROOT_DIR/entry/libs/$OHOS_ARCH/libohos.so" .
popd >/dev/null

cp "$ROOT_DIR/entry/libs/$OHOS_ARCH/libohos.h" "$ROOT_DIR/entry/src/main/cpp/include/libohos.h"

echo "Built $ROOT_DIR/entry/libs/$OHOS_ARCH/libohos.so"
