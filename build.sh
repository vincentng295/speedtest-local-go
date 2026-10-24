#!/usr/bin/env bash
# Build single static binaries into ./dist (no CGO, UI embedded).
# Also packs a Magisk/KernelSU module when android targets are built.
# Usage: ./build.sh                    -> all targets (+ module)
#        ./build.sh android/arm64      -> one target (+ module)
#        ./build.sh magisk             -> android/arm64 + android/amd64 + module
#        VERSION=1.2.0 ./build.sh magisk
set -euo pipefail
cd "$(dirname "$0")"

VERSION=${VERSION:-1.0.0}
TARGETS=("$@")
[ ${#TARGETS[@]} -eq 0 ] && TARGETS=(linux/amd64 linux/arm64 android/arm64 android/amd64 windows/amd64 darwin/arm64)
if [ "${TARGETS[0]}" = magisk ]; then TARGETS=(android/arm64 android/amd64); fi

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

# ---- Magisk / KernelSU module ----
build_module() {
  local stage=dist/.module zip_out="dist/speedtest-magisk-v${VERSION}.zip" n=0
  rm -rf "$stage"; mkdir -p "$stage"
  cp magisk/service.sh magisk/customize.sh "$stage/"
  sed -e "s/@VERSION@/${VERSION}/" -e "s/@VERSIONCODE@/$(date +%Y%m%d)/" \
      magisk/module.prop > "$stage/module.prop"

  if [ -f dist/speedtest-android-arm64 ]; then cp dist/speedtest-android-arm64 "$stage/speedtest-arm64"; n=$((n+1)); fi
  if [ -f dist/speedtest-android-amd64 ]; then cp dist/speedtest-android-amd64 "$stage/speedtest-x64";   n=$((n+1)); fi
  [ "$n" -eq 0 ] && return 0

  rm -f "$zip_out"
  if command -v zip >/dev/null 2>&1; then
    (cd "$stage" && zip -qr9 "../$(basename "$zip_out")" .)
  else
    python3 - "$stage" "$zip_out" <<'PY'
import os, sys, zipfile
src, out = sys.argv[1:3]
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    for root, _, files in os.walk(src):
        for f in files:
            p = os.path.join(root, f)
            z.write(p, os.path.relpath(p, src))
PY
  fi
  rm -rf "$stage"
  echo "magisk module: $zip_out"
}

for t in "${TARGETS[@]}"; do
  if [[ $t == android/* ]]; then build_module; break; fi
done

ls -lh dist
