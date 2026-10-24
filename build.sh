#!/usr/bin/env bash
# Build single static binaries into ./dist (no CGO, UI embedded).
# Usage: ./build.sh            -> all targets
#        ./build.sh android/arm64 linux/amd64
set -euo pipefail
cd "$(dirname "$0")"

TARGETS=("$@")
[ ${#TARGETS[@]} -eq 0 ] && TARGETS=(linux/amd64 linux/arm64 android/arm64 android/amd64 windows/amd64 darwin/arm64)

mkdir -p dist
for t in "${TARGETS[@]}"; do
  os=${t%/*}; arch=${t#*/}
  out="dist/speedtest-${os}-${arch}"
  [ "$os" = windows ] && out="$out.exe"
  goos=$os
  # Go only supports android/arm64 without cgo. For other Android archs, build a
  # static linux binary, which runs fine on Android (no libc dependency).
  if [ "$os" = android ] && [ "$arch" != arm64 ]; then goos=linux; fi
  echo "building $out (GOOS=$goos)"
  CGO_ENABLED=0 GOOS=$goos GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "$out" .
done
ls -lh dist
