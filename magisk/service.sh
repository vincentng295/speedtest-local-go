#!/system/bin/sh
MODDIR=${0%/*}
DATA=/data/adb/speedtest

until [ "$(getprop sys.boot_completed)" = "1" ]; do sleep 5; done

mkdir -p "$DATA"
cd "$DATA" || exit 1   # speedtest reads .env from the working directory
nohup "$MODDIR/speedtest" >> "$DATA/speedtest.log" 2>&1 &
