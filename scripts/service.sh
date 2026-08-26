#!/system/bin/sh
# JunkClean daemon 启动
MODDIR=${0%/*}
if ! pgrep -f "junkclean daemon" >/dev/null 2>&1; then
  nohup "$MODDIR/system/bin/junkclean" daemon >/dev/null 2>&1 &
fi
