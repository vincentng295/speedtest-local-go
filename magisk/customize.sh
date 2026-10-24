#!/system/bin/sh
# Pick the binary matching the device ABI (Magisk/KernelSU set $ARCH).
case "$ARCH" in
  arm64) SRC=speedtest-arm64 ;;
  x64)   SRC=speedtest-x64 ;;
  *)     abort "! Unsupported ARCH: $ARCH" ;;
esac
[ -f "$MODPATH/$SRC" ] || abort "! $SRC not included in this zip"

ui_print "- Installing $SRC"
mv "$MODPATH/$SRC" "$MODPATH/speedtest"
rm -f "$MODPATH"/speedtest-arm64 "$MODPATH"/speedtest-x64
set_perm "$MODPATH/speedtest" 0 0 0755
set_perm "$MODPATH/service.sh" 0 0 0755

# Config lives outside the module so updates don't wipe it.
DATA=/data/adb/speedtest
mkdir -p "$DATA"
if [ ! -f "$DATA/.env" ]; then
  cat > "$DATA/.env" <<EOT
PORT=53808
SPEEDTEST_ISP=Local Network
SPEEDTEST_LOCATION=LAN
EOT
fi
ui_print "- Config: $DATA/.env  (log: $DATA/speedtest.log)"
