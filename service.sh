#!/system/bin/sh
# JunkClean daemon 自启
MODDIR=${0%/*}
BIN="$MODDIR/system/bin/junkclean"
ADR=/data/adb/junkclean
[ -d "$ADR" ] || mkdir -p "$ADR"
# 等系统服务就绪再启动（避免权限/存储未挂载）
sleep 10
if ! pgrep -f "junkclean daemon" >/dev/null 2>&1; then
  "$BIN" daemon >> "$ADR/daemon.log" 2>&1 &
fi
