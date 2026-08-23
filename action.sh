#!/system/bin/sh
# JunkClean action.sh - 管理器 Action 按钮：立即全量清理
MODDIR=${0%/*}
ADR=/data/adb/junk-cleaner
BIN=$MODDIR/bin
ui_print() { echo "$@"; }
if ! pidof cleand >/dev/null 2>&1; then
  "$BIN/cleand" -d "$ADR" -m "$MODDIR" >> "$ADR/daemon.log" 2>&1 &
  sleep 1
fi
"$BIN/curl" -s -X POST "http://127.0.0.1:46780/api/clean" -d '{"cats":"all"}' | "$BIN/curl" -s http://127.0.0.1:46780/api/log | head -c 400
ui_print "• JunkClean 清理完成，详见 WebUI 日志"
